package lifecycle

import (
	"context"
	"fmt"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

// LifecycleStatus is the bounded outcome of one lifecycle operation.
type LifecycleStatus string

const (
	StatusComplete   LifecycleStatus = "complete"
	StatusDegraded   LifecycleStatus = "degraded"
	StatusRetryable  LifecycleStatus = "retryable"
	StatusIncomplete LifecycleStatus = "incomplete"
)

// Valid reports whether the status is one of the closed lifecycle classes.
func (status LifecycleStatus) Valid() bool {
	switch status {
	case StatusComplete, StatusDegraded, StatusRetryable, StatusIncomplete:
		return true
	default:
		return false
	}
}

const (
	OperationLoad      = "load"
	OperationUpdate    = "update"
	OperationReset     = "reset"
	OperationUninstall = "uninstall"
)

const (
	StepStorageReset      = "storage_reset"
	StepIndexCleared      = "index_cleared"
	StepSnapshotRequested = "snapshot_requested"
	StepSessionClosed     = "session_closed"
	StepStorageUninstall  = "storage_uninstall"
	StepPathsRemoved      = "paths_removed"
)

// ErrorClass is a closed, redacted failure category. It never carries SQLite
// text, paths, titles, URLs, queries, or caller-supplied strings.
type ErrorClass string

const (
	ErrorNone                    ErrorClass = ""
	ErrorInvalidConfiguration    ErrorClass = "invalid_configuration"
	ErrorProfileMismatch         ErrorClass = "profile_mismatch"
	ErrorStorageUnavailable      ErrorClass = "storage_unavailable"
	ErrorStorageTimeout          ErrorClass = "storage_timeout"
	ErrorStorageFailure          ErrorClass = "storage_failure"
	ErrorSchemaUpgradeRequired   ErrorClass = "schema_upgrade_required"
	ErrorDatabaseQuarantined     ErrorClass = "database_quarantined"
	ErrorDatabaseCorrupt         ErrorClass = "database_corrupt"
	ErrorSchemaIntegrity         ErrorClass = "schema_integrity_failed"
	ErrorQuarantineFailed        ErrorClass = "quarantine_failed"
	ErrorRecoveryFailed          ErrorClass = "recovery_failed"
	ErrorUnsafeDatabasePath      ErrorClass = "unsafe_database_path"
	ErrorCompatibilityMismatch   ErrorClass = "compatibility_mismatch"
	ErrorResetRuntimeUnavailable ErrorClass = "reset_runtime_unavailable"
	ErrorProjectionResetFailed   ErrorClass = "projection_reset_failed"
	ErrorProjectionIncomplete    ErrorClass = "projection_reset_incomplete"
	ErrorCancelled               ErrorClass = "cancelled"
	ErrorUninstallNotConfigured  ErrorClass = "uninstall_not_configured"
	ErrorSessionCloseFailed      ErrorClass = "session_close_failed"
	ErrorCleanupManifestInvalid  ErrorClass = "cleanup_manifest_invalid"
	ErrorCleanupCancelled        ErrorClass = "cleanup_cancelled"
	ErrorCleanupFailed           ErrorClass = "cleanup_failed"
	ErrorPathSafety              ErrorClass = "path_safety"
	ErrorUnknown                 ErrorClass = "unknown"
)

// Valid reports whether the class is one of the closed lifecycle categories.
func (class ErrorClass) Valid() bool {
	switch class {
	case ErrorNone,
		ErrorInvalidConfiguration,
		ErrorProfileMismatch,
		ErrorStorageUnavailable,
		ErrorStorageTimeout,
		ErrorStorageFailure,
		ErrorSchemaUpgradeRequired,
		ErrorDatabaseQuarantined,
		ErrorDatabaseCorrupt,
		ErrorSchemaIntegrity,
		ErrorQuarantineFailed,
		ErrorRecoveryFailed,
		ErrorUnsafeDatabasePath,
		ErrorCompatibilityMismatch,
		ErrorResetRuntimeUnavailable,
		ErrorProjectionResetFailed,
		ErrorProjectionIncomplete,
		ErrorCancelled,
		ErrorUninstallNotConfigured,
		ErrorSessionCloseFailed,
		ErrorCleanupManifestInvalid,
		ErrorCleanupCancelled,
		ErrorCleanupFailed,
		ErrorPathSafety,
		ErrorUnknown:
		return true
	default:
		return false
	}
}

// ProjectionReset is the observable handoff from the lifecycle owner to the
// projection/index owner after the profile's durable rows are gone.
type ProjectionReset struct {
	Revision          uint64
	IndexCleared      bool
	SnapshotRequested bool
}

// ResetRuntime is injected by the composition layer. It owns in-memory index
// clearing and the full-snapshot request; the lifecycle package never mutates
// projection state directly.
type ResetRuntime interface {
	ResetProjection(ctx context.Context, profileID domain.ProfileID) (ProjectionReset, error)
}

// LifecycleResult is the redacted UI/diagnostics contract for load, update,
// reset, and uninstall. It contains no raw titles, URLs, queries, or arbitrary
// filesystem paths.
type LifecycleResult struct {
	Operation          string           `json:"operation"`
	ProfileID          domain.ProfileID `json:"profile_id"`
	Status             LifecycleStatus  `json:"status"`
	Retryable          bool             `json:"retryable"`
	Persistence        PersistenceState `json:"persistence"`
	Durable            bool             `json:"durable"`
	ErrorClass         ErrorClass       `json:"error_class"`
	CompletedSteps     []string         `json:"completed_steps"`
	OwnedStateVerified bool             `json:"owned_state_verified"`
	ProjectionRevision uint64           `json:"projection_revision"`
	Configuration      ProfileConfig    `json:"configuration"`
}

// Diagnostics is the subset safe to log or attach to a bug report. It excludes
// the configuration payload and contains only closed status/step/error values.
type Diagnostics struct {
	Operation          string           `json:"operation"`
	ProfileID          domain.ProfileID `json:"profile_id"`
	Status             LifecycleStatus  `json:"status"`
	Retryable          bool             `json:"retryable"`
	Persistence        PersistenceState `json:"persistence"`
	Durable            bool             `json:"durable"`
	ErrorClass         ErrorClass       `json:"error_class"`
	CompletedSteps     []string         `json:"completed_steps"`
	OwnedStateVerified bool             `json:"owned_state_verified"`
	ProjectionRevision uint64           `json:"projection_revision"`
}

// RedactedDiagnostics returns the log-safe subset.
func (result LifecycleResult) RedactedDiagnostics() Diagnostics {
	return Diagnostics{
		Operation:          result.Operation,
		ProfileID:          result.ProfileID,
		Status:             result.Status,
		Retryable:          result.Retryable,
		Persistence:        result.Persistence,
		Durable:            result.Durable,
		ErrorClass:         result.ErrorClass,
		CompletedSteps:     append([]string(nil), result.CompletedSteps...),
		OwnedStateVerified: result.OwnedStateVerified,
		ProjectionRevision: result.ProjectionRevision,
	}
}

// Validate rejects a result that escaped the closed status/error/step sets or
// carries an invalid profile. Callers can therefore log the result safely.
func (result LifecycleResult) Validate() error {
	if !result.Status.Valid() {
		return fmt.Errorf("invalid lifecycle status")
	}
	if !result.ErrorClass.Valid() {
		return fmt.Errorf("invalid lifecycle error class")
	}
	if result.ProfileID != "" {
		if _, err := domain.ParseProfileID(string(result.ProfileID)); err != nil {
			return fmt.Errorf("invalid lifecycle profile")
		}
	}
	for _, step := range result.CompletedSteps {
		if !validStep(step) {
			return fmt.Errorf("invalid lifecycle step")
		}
	}
	return nil
}

func validStep(step string) bool {
	switch step {
	case StepStorageReset,
		StepIndexCleared,
		StepSnapshotRequested,
		StepSessionClosed,
		StepStorageUninstall,
		StepPathsRemoved:
		return true
	default:
		return false
	}
}

func errorClassFromStorage(class storage.ErrorClass) ErrorClass {
	switch class {
	case storage.ErrorProfileMismatch:
		return ErrorProfileMismatch
	case storage.ErrorInvalidConfiguration:
		return ErrorInvalidConfiguration
	case storage.ErrorStorageUnavailable:
		return ErrorStorageUnavailable
	case storage.ErrorStorageTimeout:
		return ErrorStorageTimeout
	case storage.ErrorStorageFailure:
		return ErrorStorageFailure
	case storage.ErrorSchemaUpgradeRequired:
		return ErrorSchemaUpgradeRequired
	case storage.ErrorDatabaseQuarantined:
		return ErrorDatabaseQuarantined
	case storage.ErrorDatabaseCorrupt:
		return ErrorDatabaseCorrupt
	case storage.ErrorSchemaIntegrity:
		return ErrorSchemaIntegrity
	case storage.ErrorQuarantineFailed:
		return ErrorQuarantineFailed
	case storage.ErrorRecoveryFailed:
		return ErrorRecoveryFailed
	case storage.ErrorUnsafeDatabasePath:
		return ErrorUnsafeDatabasePath
	case "":
		return ErrorNone
	default:
		return ErrorUnknown
	}
}
