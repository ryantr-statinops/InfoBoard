package lifecycle

import (
	"context"
	"errors"
	"sync"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

// Store is the bounded IP-08 repository surface the lifecycle coordinator may
// use. It never exposes raw database handles or SQL to callers.
type Store interface {
	LoadConfiguration(ctx context.Context, profileID string, versions storage.CompatibilityVersions) (storage.Configuration, storage.Result)
	SaveConfiguration(ctx context.Context, configuration storage.Configuration, versions storage.CompatibilityVersions) storage.Result
	ResetProfile(ctx context.Context, profileID string) storage.Result
}

// Coordinator owns profile-scoped effective settings and the lifecycle
// operations around them. A degraded store never removes the in-memory
// defaults needed by the lexical search path.
type Coordinator struct {
	mu        sync.RWMutex
	store     Store
	versions  storage.CompatibilityVersions
	effective map[domain.ProfileID]EffectiveConfiguration
}

// NewCoordinator validates the bounded repository/version inputs and starts
// with no cached profile state.
func NewCoordinator(store Store, versions storage.CompatibilityVersions) (*Coordinator, error) {
	if store == nil {
		return nil, errors.New("lifecycle store is required")
	}
	if versions.ProtocolVersion <= 0 {
		return nil, ErrInvalidConfiguration
	}
	if len(versions.RankingModelVersion) > MaxRankingModelVersionSize {
		return nil, ErrInvalidConfiguration
	}
	return &Coordinator{
		store:     store,
		versions:  versions,
		effective: make(map[domain.ProfileID]EffectiveConfiguration),
	}, nil
}

// Load returns persisted values only for profileID. Missing rows, invalid
// stored values, compatibility mismatches, and storage failures fall back to
// the contract defaults and expose an explicit persistence state. The returned
// error is reserved for invalid input or cancellation; a degraded store is a
// successful, renderable result.
func (coordinator *Coordinator) Load(ctx context.Context, profileID domain.ProfileID) (EffectiveConfiguration, error) {
	if coordinator == nil {
		return EffectiveConfiguration{}, errors.New("lifecycle coordinator is required")
	}
	profile, err := domain.ParseProfileID(string(profileID))
	if err != nil {
		return EffectiveConfiguration{}, ErrInvalidConfiguration
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return EffectiveConfiguration{}, err
	}

	defaults := DefaultProfileConfig(profile, coordinator.versions.ProtocolVersion, coordinator.versions.RankingModelVersion)
	configuration, result := coordinator.store.LoadConfiguration(ctx, string(profile), coordinator.versions)
	if err := ctx.Err(); err != nil {
		return EffectiveConfiguration{}, err
	}
	persistence, durable := persistenceFromResult(result)
	errorClass := string(result.ErrorClass)
	effective := defaults
	if persistence == PersistenceReady && durable {
		candidate := profileConfigFromStorage(configuration)
		if candidate.ProfileID == profile && candidate.Validate(profile) == nil {
			effective = candidate
		} else {
			persistence, durable = PersistenceDegraded, false
			errorClass = string(storage.ErrorInvalidConfiguration)
		}
	} else if errorClass == "" {
		if result.Status == storage.StatusReady && result.Compatibility != storage.CompatibilityCompatible {
			errorClass = "compatibility_mismatch"
		} else {
			errorClass = string(storage.ErrorStorageUnavailable)
		}
	}

	value := EffectiveConfiguration{
		Configuration: effective,
		Persistence:   persistence,
		Durable:       durable,
		Retryable:     !durable,
		ErrorClass:    errorClass,
	}
	coordinator.storeEffective(profile, value)
	return value, nil
}

// Update validates and writes one profile's complete effective configuration.
// The host stamps protocol/ranking versions; a caller cannot choose them. On a
// write failure the last known effective value is retained, marked degraded and
// retryable, and never reported as durable.
func (coordinator *Coordinator) Update(ctx context.Context, profileID domain.ProfileID, configuration ProfileConfig) (EffectiveConfiguration, error) {
	if coordinator == nil {
		return EffectiveConfiguration{}, errors.New("lifecycle coordinator is required")
	}
	profile, err := domain.ParseProfileID(string(profileID))
	if err != nil {
		return EffectiveConfiguration{}, ErrInvalidConfiguration
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return EffectiveConfiguration{}, err
	}
	if configuration.ProfileID != profile {
		return EffectiveConfiguration{}, &ValidationError{Field: "profile_id", Err: ErrProfileMismatch}
	}
	if err := configuration.Validate(profile); err != nil {
		return EffectiveConfiguration{}, err
	}
	configuration.ProfileID = profile
	configuration.ProtocolVersion = coordinator.versions.ProtocolVersion
	configuration.RankingModelVersion = coordinator.versions.RankingModelVersion
	if err := configuration.Validate(profile); err != nil {
		return EffectiveConfiguration{}, err
	}

	result := coordinator.store.SaveConfiguration(ctx, storageConfiguration(configuration), coordinator.versions)
	if err := ctx.Err(); err != nil {
		return EffectiveConfiguration{}, err
	}
	persistence, durable := persistenceFromResult(result)
	if persistence == PersistenceReady && durable {
		value := EffectiveConfiguration{
			Configuration: configuration,
			Persistence:   PersistenceReady,
			Durable:       true,
		}
		coordinator.storeEffective(profile, value)
		return value, nil
	}

	previous := coordinator.previous(profile)
	previous.Persistence = persistence
	previous.Durable = false
	previous.Retryable = true
	previous.ErrorClass = string(result.ErrorClass)
	if previous.ErrorClass == "" {
		previous.ErrorClass = string(storage.ErrorStorageFailure)
	}
	coordinator.storeEffective(profile, previous)
	return previous, nil
}

func (coordinator *Coordinator) previous(profile domain.ProfileID) EffectiveConfiguration {
	coordinator.mu.RLock()
	value, ok := coordinator.effective[profile]
	coordinator.mu.RUnlock()
	if ok {
		return value
	}
	return EffectiveConfiguration{
		Configuration: DefaultProfileConfig(profile, coordinator.versions.ProtocolVersion, coordinator.versions.RankingModelVersion),
		Persistence:   PersistenceDegraded,
		Durable:       false,
		Retryable:     true,
		ErrorClass:    string(storage.ErrorStorageUnavailable),
	}
}

func (coordinator *Coordinator) storeEffective(profile domain.ProfileID, value EffectiveConfiguration) {
	coordinator.mu.Lock()
	coordinator.effective[profile] = value
	coordinator.mu.Unlock()
}

// Cached returns the last effective value for one profile without touching the
// store. It is used by the lexical path when persistence is unavailable.
func (coordinator *Coordinator) Cached(profileID domain.ProfileID) (EffectiveConfiguration, bool) {
	if coordinator == nil {
		return EffectiveConfiguration{}, false
	}
	coordinator.mu.RLock()
	defer coordinator.mu.RUnlock()
	value, ok := coordinator.effective[profileID]
	return value, ok
}

func persistenceFromResult(result storage.Result) (PersistenceState, bool) {
	switch result.Status {
	case storage.StatusReady:
		if result.Compatibility == storage.CompatibilityCompatible {
			return PersistenceReady, true
		}
		return PersistenceDegraded, false
	case storage.StatusDegraded, storage.StatusRebuilt, storage.StatusQuarantined, storage.StatusUpgradeRequired:
		return PersistenceDegraded, false
	case storage.StatusDisabled:
		return PersistenceUnavailable, false
	default:
		return PersistenceUnknown, false
	}
}

func profileConfigFromStorage(configuration storage.Configuration) ProfileConfig {
	return ProfileConfig{
		ProfileID:           configuration.ProfileID,
		ResultLimit:         configuration.ResultLimit,
		Density:             Density(configuration.Density),
		Theme:               Theme(configuration.Theme),
		ContextLabels:       configuration.ShowContextLabels,
		RecencyEnabled:      configuration.PersistActivationRecency,
		ProtocolVersion:     configuration.ProtocolVersion,
		RankingModelVersion: configuration.RankingModelVersion,
	}
}

func storageConfiguration(configuration ProfileConfig) storage.Configuration {
	return storage.Configuration{
		ProfileID:                configuration.ProfileID,
		ResultLimit:              configuration.ResultLimit,
		Density:                  string(configuration.Density),
		Theme:                    string(configuration.Theme),
		ShowContextLabels:        configuration.ContextLabels,
		PersistActivationRecency: configuration.RecencyEnabled,
		ProtocolVersion:          configuration.ProtocolVersion,
		RankingModelVersion:      configuration.RankingModelVersion,
	}
}
