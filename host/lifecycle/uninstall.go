package lifecycle

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

// SessionCloser is supplied by the composition layer. Uninstall asks it to
// close the Native Messaging session before removing files.
type SessionCloser interface {
	CloseSession(ctx context.Context, profileID domain.ProfileID) error
}

// UninstallPlan is built from the checked-in cleanup manifest and the trusted
// database path. A request never supplies either value.
type UninstallPlan struct {
	Manifest     Manifest
	DatabasePath string
}

// NewUninstallPlan validates the reviewed manifest and database path once, at
// composition time.
func NewUninstallPlan(manifest Manifest, databasePath string) (UninstallPlan, error) {
	if manifest.Root() == "" || len(manifest.Entries()) == 0 {
		return UninstallPlan{}, ErrInvalidConfiguration
	}
	if databasePath == "" ||
		!filepath.IsAbs(databasePath) ||
		filepath.Base(databasePath) != storage.DatabaseFileName {
		return UninstallPlan{}, ErrInvalidConfiguration
	}
	return UninstallPlan{Manifest: manifest, DatabasePath: filepath.Clean(databasePath)}, nil
}

// SetUninstallPlan injects the reviewed cleanup plan. It is optional for
// load/update/reset-only callers but required before Uninstall can complete.
func (coordinator *Coordinator) SetUninstallPlan(plan UninstallPlan) {
	if coordinator == nil {
		return
	}
	coordinator.mu.Lock()
	coordinator.uninstallPlan = &plan
	coordinator.mu.Unlock()
}

// SetSessionCloser injects the session-close handoff used before cleanup.
func (coordinator *Coordinator) SetSessionCloser(closer SessionCloser) {
	if coordinator == nil {
		return
	}
	coordinator.mu.Lock()
	coordinator.sessionCloser = closer
	coordinator.mu.Unlock()
}

func (coordinator *Coordinator) uninstallConfiguration() (*UninstallPlan, SessionCloser) {
	coordinator.mu.RLock()
	defer coordinator.mu.RUnlock()
	return coordinator.uninstallPlan, coordinator.sessionCloser
}

// Uninstall closes the session, removes the fixed storage artifacts, and then
// removes exactly the reviewed manifest entries. Missing artifacts count as
// complete; permission/path failures are retryable and never reported as a
// verified owned-state result.
func (coordinator *Coordinator) Uninstall(ctx context.Context, profileID domain.ProfileID) (LifecycleResult, error) {
	if coordinator == nil {
		return LifecycleResult{}, errors.New("lifecycle coordinator is required")
	}
	profile, err := domain.ParseProfileID(string(profileID))
	if err != nil {
		return LifecycleResult{}, ErrInvalidConfiguration
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return LifecycleResult{}, err
	}

	coordinator.writeMu.Lock()
	defer coordinator.writeMu.Unlock()

	defaults := DefaultProfileConfig(profile, coordinator.versions.ProtocolVersion, coordinator.versions.RankingModelVersion)
	plan, closer := coordinator.uninstallConfiguration()
	if plan == nil || closer == nil {
		return LifecycleResult{
			Operation:     OperationUninstall,
			ProfileID:     profile,
			Status:        StatusRetryable,
			Retryable:     true,
			Persistence:   PersistenceUnknown,
			Durable:       false,
			ErrorClass:    ErrorUninstallNotConfigured,
			Configuration: defaults,
		}, nil
	}

	if err := closer.CloseSession(ctx, profile); err != nil {
		return LifecycleResult{
			Operation:     OperationUninstall,
			ProfileID:     profile,
			Status:        StatusRetryable,
			Retryable:     true,
			Persistence:   PersistenceUnknown,
			Durable:       false,
			ErrorClass:    ErrorSessionCloseFailed,
			Configuration: defaults,
		}, nil
	}
	steps := []string{StepSessionClosed}

	result := coordinator.store.Uninstall(ctx, plan.DatabasePath)
	if result.Status != storage.StatusDisabled {
		persistence, _ := persistenceFromResult(result)
		errorClass := errorClassFromStorage(result.ErrorClass)
		if errorClass == ErrorNone {
			errorClass = ErrorStorageFailure
		}
		return LifecycleResult{
			Operation:      OperationUninstall,
			ProfileID:      profile,
			Status:         StatusRetryable,
			Retryable:      true,
			Persistence:    persistence,
			Durable:        false,
			ErrorClass:     errorClass,
			CompletedSteps: steps,
			Configuration:  defaults,
		}, nil
	}
	steps = append(steps, StepStorageUninstall)

	cleaner, err := NewCleaner(plan.Manifest)
	if err != nil {
		return LifecycleResult{
			Operation:      OperationUninstall,
			ProfileID:      profile,
			Status:         StatusIncomplete,
			Retryable:      true,
			Persistence:    PersistenceUnavailable,
			Durable:        false,
			ErrorClass:     ErrorCleanupManifestInvalid,
			CompletedSteps: steps,
			Configuration:  defaults,
		}, nil
	}
	cleanup, cleanupErr := cleaner.Remove(ctx)
	coordinator.clearEffective()
	if cleanupErr != nil {
		return LifecycleResult{
			Operation:      OperationUninstall,
			ProfileID:      profile,
			Status:         StatusIncomplete,
			Retryable:      true,
			Persistence:    PersistenceUnavailable,
			Durable:        false,
			ErrorClass:     ErrorCleanupCancelled,
			CompletedSteps: steps,
			Configuration:  defaults,
		}, cleanupErr
	}
	if cleanup.Verified {
		steps = append(steps, StepPathsRemoved)
	}
	status := StatusComplete
	retryable := false
	errorClass := ErrorNone
	if !cleanup.Verified {
		status = StatusIncomplete
		retryable = true
		errorClass = cleanup.ErrorClass
		if errorClass == ErrorNone {
			errorClass = ErrorCleanupFailed
		}
	}
	return LifecycleResult{
		Operation:          OperationUninstall,
		ProfileID:          profile,
		Status:             status,
		Retryable:          retryable,
		Persistence:        PersistenceUnavailable,
		Durable:            false,
		ErrorClass:         errorClass,
		CompletedSteps:     steps,
		OwnedStateVerified: cleanup.Verified,
		Configuration:      defaults,
	}, nil
}

func (coordinator *Coordinator) clearEffective() {
	coordinator.mu.Lock()
	coordinator.effective = make(map[domain.ProfileID]EffectiveConfiguration)
	coordinator.mu.Unlock()
}
