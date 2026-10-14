package storage

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

var errStorageUnavailable = errors.New("storage unavailable")

// Status is the bounded persistence state exposed to host/runtime adapters.
type Status string

const (
	StatusReady           Status = "ready"
	StatusDegraded        Status = "degraded"
	StatusUpgradeRequired Status = "upgrade_required"
	StatusQuarantined     Status = "quarantined"
	StatusRebuilt         Status = "rebuilt"
	StatusDisabled        Status = "disabled"
)

// CompatibilityStatus distinguishes stored protocol/ranking versions from
// persistence health; a mismatch never degrades the in-memory lexical path.
type CompatibilityStatus string

const (
	CompatibilityUnknown                    CompatibilityStatus = "unknown"
	CompatibilityCompatible                 CompatibilityStatus = "compatible"
	CompatibilityProtocolMismatch           CompatibilityStatus = "protocol_mismatch"
	CompatibilityRankingMismatch            CompatibilityStatus = "ranking_mismatch"
	CompatibilityProtocolAndRankingMismatch CompatibilityStatus = "protocol_and_ranking_mismatch"
)

// CompatibilityVersions are supplied by the running host, not loaded from the
// database being checked.
type CompatibilityVersions struct {
	ProtocolVersion     int
	RankingModelVersion string
}

func (versions CompatibilityVersions) valid() bool {
	return versions.ProtocolVersion > 0 && boundedString(versions.RankingModelVersion, MaxStoredVersionBytes, true)
}

// ErrorClass contains only closed storage failure categories. It never carries
// SQLite text, file paths, SQL, browser data, or caller-provided strings.
type ErrorClass string

const (
	ErrorProfileMismatch       ErrorClass = "profile_mismatch"
	ErrorInvalidConfiguration  ErrorClass = "invalid_configuration"
	ErrorStorageUnavailable    ErrorClass = "storage_unavailable"
	ErrorStorageTimeout        ErrorClass = "storage_timeout"
	ErrorStorageFailure        ErrorClass = "storage_failure"
	ErrorSchemaUpgradeRequired ErrorClass = "schema_upgrade_required"
	ErrorDatabaseQuarantined   ErrorClass = "database_quarantined"
	ErrorDatabaseCorrupt       ErrorClass = "database_corrupt"
	ErrorSchemaIntegrity       ErrorClass = "schema_integrity_failed"
	ErrorQuarantineFailed      ErrorClass = "quarantine_failed"
	ErrorRecoveryFailed        ErrorClass = "recovery_failed"
	ErrorUnsafeDatabasePath    ErrorClass = "unsafe_database_path"
)

// Result is the bounded outcome of one repository operation.
type Result struct {
	Status        Status
	ErrorClass    ErrorClass
	Retryable     bool
	Compatibility CompatibilityStatus
}

// Configuration contains durable profile-scoped user preferences and the
// versions that were used to interpret them. Live projections are not fields.
type Configuration struct {
	ProfileID                domain.ProfileID
	ResultLimit              int
	Density                  string
	Theme                    string
	ShowContextLabels        bool
	PersistActivationRecency bool
	ProtocolVersion          int
	RankingModelVersion      string
	UpdatedAt                int64
}

// DefaultConfiguration returns a privacy-first configuration for one validated
// profile. Activation recency stays session-only until the user opts in.
func DefaultConfiguration(profileID domain.ProfileID) Configuration {
	return Configuration{
		ProfileID: profileID, ResultLimit: DefaultResultLimit, Density: DefaultDensity,
		Theme: DefaultTheme, ShowContextLabels: DefaultShowContextLabels != 0,
		PersistActivationRecency: DefaultPersistActivationRecency != 0,
		ProtocolVersion:          protocol.CurrentVersion,
	}
}
func defaultsForVersions(profile domain.ProfileID, versions CompatibilityVersions) Configuration {
	configuration := DefaultConfiguration(profile)
	configuration.ProtocolVersion = versions.ProtocolVersion
	configuration.RankingModelVersion = versions.RankingModelVersion
	return configuration
}

func compatibilityOf(configuration Configuration, versions CompatibilityVersions) CompatibilityStatus {
	protocolMismatch := configuration.ProtocolVersion != versions.ProtocolVersion
	rankingMismatch := configuration.RankingModelVersion != versions.RankingModelVersion
	switch {
	case protocolMismatch && rankingMismatch:
		return CompatibilityProtocolAndRankingMismatch
	case protocolMismatch:
		return CompatibilityProtocolMismatch
	case rankingMismatch:
		return CompatibilityRankingMismatch
	default:
		return CompatibilityCompatible
	}
}

// Repository stores profile-scoped preferences. A database failure returns
// defaults to the caller and changes only persistence status; query/index and
// browser activation remain owned by their in-memory/runtime paths.
type Repository struct {
	mu         sync.RWMutex
	db         *sql.DB
	status     Status
	errorClass ErrorClass
}

// NewRepository wraps an already opened database. Opening, schema compatibility,
// and migrations are owned by the IP-08 open/migration path; a nil handle starts
// degraded rather than manufacturing an in-memory SQLite database.
func NewRepository(db *sql.DB) *Repository {
	status, class := StatusReady, ErrorClass("")
	if db == nil {
		status, class = StatusDegraded, ErrorStorageUnavailable
	}
	return &Repository{db: db, status: status, errorClass: class}
}

// Close releases the SQLite handle. It is safe to call repeatedly.
func (repository *Repository) Close() error {
	if repository == nil {
		return nil
	}
	repository.mu.Lock()
	db := repository.db
	repository.db = nil
	if repository.status != StatusDisabled {
		repository.status = StatusDegraded
		repository.errorClass = ErrorStorageUnavailable
	}
	repository.mu.Unlock()
	if db == nil {
		return nil
	}
	return db.Close()
}

// CurrentResult returns the repository's safe state without exposing database
// error text, paths, or SQL.
func (repository *Repository) CurrentResult() Result {
	if repository == nil {
		return Result{Status: StatusDegraded, ErrorClass: ErrorStorageUnavailable, Retryable: true, Compatibility: CompatibilityUnknown}
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	return repository.resultLocked()
}

func (repository *Repository) resultLocked() Result {
	class := repository.errorClass
	if class == "" {
		switch repository.status {
		case StatusDegraded:
			class = ErrorStorageUnavailable
		case StatusUpgradeRequired:
			class = ErrorSchemaUpgradeRequired
		case StatusQuarantined:
			class = ErrorDatabaseQuarantined
		}
	}
	return Result{Status: repository.status, ErrorClass: class, Retryable: repository.status == StatusDegraded || repository.status == StatusQuarantined, Compatibility: CompatibilityUnknown}
}

func (repository *Repository) markReady() Result {
	repository.mu.Lock()
	if repository.status == StatusDegraded || repository.status == StatusRebuilt {
		repository.status = StatusReady
		repository.errorClass = ""
	}
	result := repository.resultLocked()
	repository.mu.Unlock()
	return result
}

// LoadConfiguration reads only the requested profile. Missing rows return safe
// defaults; storage failures also return defaults but mark persistence degraded.
// Version mismatches are reported independently from storage health.
func (repository *Repository) LoadConfiguration(ctx context.Context, profileID string, versions CompatibilityVersions) (Configuration, Result) {
	profile, err := domain.ParseProfileID(profileID)
	if err != nil {
		return Configuration{}, Result{Status: repository.currentStatus(), ErrorClass: ErrorProfileMismatch, Compatibility: CompatibilityUnknown}
	}
	if !versions.valid() {
		return Configuration{}, Result{Status: repository.currentStatus(), ErrorClass: ErrorInvalidConfiguration, Compatibility: CompatibilityUnknown}
	}
	defaults := defaultsForVersions(profile, versions)
	if repository == nil || repository.db == nil {
		return defaults, repository.fail(ctx, errStorageUnavailable)
	}
	current := repository.CurrentResult()
	if current.Status == StatusUpgradeRequired || current.Status == StatusQuarantined {
		return defaults, current
	}

	var configuration Configuration
	var showContextLabels, persistRecency int
	query := "SELECT profile_id, result_limit, density, theme, show_context_labels, " +
		"persist_activation_recency, protocol_version, ranking_model_version, updated_at " +
		"FROM configuration WHERE profile_id = ?"
	err = repository.db.QueryRowContext(ctx, query, profileID).Scan(
		&configuration.ProfileID, &configuration.ResultLimit, &configuration.Density,
		&configuration.Theme, &showContextLabels, &persistRecency,
		&configuration.ProtocolVersion, &configuration.RankingModelVersion,
		&configuration.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		result := repository.markReady()
		result.Compatibility = CompatibilityCompatible
		return defaults, result
	}
	if err != nil {
		return defaults, repository.fail(ctx, err)
	}
	configuration.ShowContextLabels = showContextLabels == 1
	configuration.PersistActivationRecency = persistRecency == 1
	if configuration.ProfileID != profile || !configuration.valid() {
		return defaults, repository.fail(ctx, errors.New("invalid stored configuration"))
	}
	result := repository.markReady()
	result.Compatibility = compatibilityOf(configuration, versions)
	if result.Compatibility != CompatibilityCompatible {
		return defaults, result
	}
	return configuration, result
}

// SaveConfiguration validates every field and upserts exactly one profile row.
// Errors are reduced to a closed class; callers must not log the raw SQL error.
func (repository *Repository) SaveConfiguration(ctx context.Context, configuration Configuration, versions CompatibilityVersions) Result {
	if _, err := domain.ParseProfileID(string(configuration.ProfileID)); err != nil {
		return Result{Status: repository.currentStatus(), ErrorClass: ErrorProfileMismatch, Compatibility: CompatibilityUnknown}
	}
	if !configuration.valid() || !versions.valid() {
		return Result{Status: repository.currentStatus(), ErrorClass: ErrorInvalidConfiguration, Compatibility: CompatibilityUnknown}
	}
	compatibility := compatibilityOf(configuration, versions)
	if compatibility != CompatibilityCompatible {
		result := repository.CurrentResult()
		result.Compatibility = compatibility
		return result
	}
	current := repository.CurrentResult()
	if current.Status == StatusUpgradeRequired || current.Status == StatusQuarantined {
		return current
	}
	if repository == nil || repository.db == nil {
		return repository.fail(ctx, errStorageUnavailable)
	}
	updatedAt := configuration.UpdatedAt
	if updatedAt <= 0 {
		updatedAt = time.Now().Unix()
	}
	_, err := repository.db.ExecContext(ctx, `
		INSERT INTO configuration (
			profile_id, result_limit, density, theme, show_context_labels,
			persist_activation_recency, protocol_version, ranking_model_version, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(profile_id) DO UPDATE SET
			result_limit = excluded.result_limit,
			density = excluded.density,
			theme = excluded.theme,
			show_context_labels = excluded.show_context_labels,
			persist_activation_recency = excluded.persist_activation_recency,
			protocol_version = excluded.protocol_version,
			ranking_model_version = excluded.ranking_model_version,
			updated_at = excluded.updated_at`,
		string(configuration.ProfileID), configuration.ResultLimit, configuration.Density,
		configuration.Theme, configuration.ShowContextLabels, configuration.PersistActivationRecency,
		configuration.ProtocolVersion, configuration.RankingModelVersion, updatedAt,
	)
	if err != nil {
		return repository.fail(ctx, err)
	}
	result := repository.markReady()
	result.Compatibility = CompatibilityCompatible
	return result
}

func (configuration Configuration) valid() bool {
	return configuration.ResultLimit >= 1 && configuration.ResultLimit <= protocol.MaxResults &&
		(configuration.Density == "compact" || configuration.Density == "comfortable") &&
		boundedString(configuration.Theme, MaxStoredVersionBytes, false) &&
		configuration.ProtocolVersion > 0 &&
		boundedString(configuration.RankingModelVersion, MaxStoredVersionBytes, true) &&
		configuration.UpdatedAt >= 0
}

func boundedString(value string, maxBytes int, allowEmpty bool) bool {
	return (allowEmpty || value != "") && utf8.ValidString(value) && len(value) <= maxBytes
}

func (repository *Repository) currentStatus() Status {
	return repository.CurrentResult().Status
}

func (repository *Repository) fail(ctx context.Context, err error) Result {
	class := ErrorStorageFailure
	if ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		class = ErrorStorageTimeout
	}
	if repository == nil {
		return Result{Status: StatusDegraded, ErrorClass: ErrorStorageUnavailable, Retryable: true, Compatibility: CompatibilityUnknown}
	}
	repository.mu.Lock()
	if repository.status == StatusDisabled {
		repository.mu.Unlock()
		return Result{Status: StatusDisabled, ErrorClass: ErrorStorageUnavailable, Compatibility: CompatibilityUnknown}
	}
	if repository.status != StatusUpgradeRequired && repository.status != StatusQuarantined {
		repository.status = StatusDegraded
		repository.errorClass = class
	}
	result := repository.resultLocked()
	repository.mu.Unlock()
	return result
}
