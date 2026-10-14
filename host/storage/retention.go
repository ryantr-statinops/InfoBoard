package storage

import (
	"context"
	"database/sql"
	"time"
	"unicode/utf8"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// ActivationRecord is the durable subset of a successful normal-context
// activation. It deliberately excludes titles, URLs, queries, and result content.
type ActivationRecord struct {
	ProfileID   string
	TabID       int64
	Domain      string
	ActivatedAt int64 // Unix seconds.
	Source      domain.ActivationSource
}

// DiagnosticRecord contains bounded event metadata and an already-redacted
// payload. PayloadBytes is derived from UTF-8 bytes at insertion time.
type DiagnosticRecord struct {
	OccurredAt          int64 // Unix milliseconds.
	ProfileID           string
	EventKind           string
	ErrorClass          ErrorClass
	Retryable           bool
	DurationMillis      int64
	ProjectionRevision  uint64
	ProtocolVersion     int
	RankingModelVersion string
	SchemaVersion       uint64
	RedactedPayload     string
}

// RecordActivation stores only valid normal-context metadata and only when the
// profile has explicitly enabled activation-recency persistence. It enforces
// time and per-profile count retention in the same transaction as the insert.
func (repository *Repository) RecordActivation(ctx context.Context, record ActivationRecord) Result {
	profile, err := domain.ParseProfileID(record.ProfileID)
	if err != nil {
		return Result{Status: repository.currentStatus(), ErrorClass: ErrorProfileMismatch}
	}
	if record.TabID < 0 || uint64(record.TabID) > domain.MaxProjectionRevision ||
		!boundedString(record.Domain, 255, false) || record.ActivatedAt < 0 ||
		(record.Source != domain.ActivationKeyboardEnter && record.Source != domain.ActivationSurfaceSelect) {
		return Result{Status: repository.currentStatus(), ErrorClass: ErrorInvalidConfiguration}
	}
	if ctx == nil || ctx.Err() != nil {
		return repository.fail(ctx, context.Canceled)
	}
	db := repository.database()
	if db == nil {
		return repository.fail(ctx, errStorageUnavailable)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return repository.fail(ctx, err)
	}
	defer tx.Rollback()
	var enabled bool
	err = tx.QueryRowContext(ctx, `SELECT persist_activation_recency FROM configuration WHERE profile_id = ?`, string(profile)).Scan(&enabled)
	if err != nil && err != sql.ErrNoRows {
		return repository.fail(ctx, err)
	}
	if !enabled {
		return repository.markReady()
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO activation_metadata(profile_id, context_kind, tab_id, domain, activated_at, source) VALUES (?, 'normal', ?, ?, ?, ?)`,
		string(profile), record.TabID, record.Domain, record.ActivatedAt, record.Source); err != nil {
		return repository.fail(ctx, err)
	}
	if err := pruneRetentionTx(ctx, tx, time.Now()); err != nil {
		return repository.fail(ctx, err)
	}
	if err := tx.Commit(); err != nil {
		return repository.fail(ctx, err)
	}
	return repository.markReady()
}

// AppendDiagnostic stores one bounded diagnostic event and prunes expired or
// excess payload rows atomically. The caller must supply a redacted payload;
// IP-08 T10 owns the redaction policy and allowed event vocabulary.
func (repository *Repository) AppendDiagnostic(ctx context.Context, record DiagnosticRecord) Result {
	profileID := any(nil)
	if record.ProfileID != "" {
		profile, err := domain.ParseProfileID(record.ProfileID)
		if err != nil {
			return Result{Status: repository.currentStatus(), ErrorClass: ErrorProfileMismatch}
		}
		profileID = string(profile)
	}
	if !boundedString(record.EventKind, MaxStoredVersionBytes, false) ||
		!boundedString(string(record.ErrorClass), MaxStoredVersionBytes, false) ||
		record.OccurredAt < 0 || record.DurationMillis < 0 ||
		record.ProjectionRevision > domain.MaxProjectionRevision ||
		record.ProtocolVersion <= 0 ||
		!boundedString(record.RankingModelVersion, MaxStoredVersionBytes, true) ||
		record.SchemaVersion == 0 || record.SchemaVersion > SupportedSchemaVersion ||
		!utf8.ValidString(record.RedactedPayload) || len(record.RedactedPayload) > MaxDiagnosticRowBytes {
		return Result{Status: repository.currentStatus(), ErrorClass: ErrorInvalidConfiguration}
	}
	if ctx == nil || ctx.Err() != nil {
		return repository.fail(ctx, context.Canceled)
	}
	db := repository.database()
	if db == nil {
		return repository.fail(ctx, errStorageUnavailable)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return repository.fail(ctx, err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO diagnostics(occurred_at, profile_id, event_kind, error_class, retryable, duration_ms, projection_revision, protocol_version, ranking_model_version, schema_version, payload_bytes, redacted_payload) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.OccurredAt, profileID, record.EventKind, string(record.ErrorClass), record.Retryable, record.DurationMillis,
		record.ProjectionRevision, record.ProtocolVersion, record.RankingModelVersion, record.SchemaVersion,
		len(record.RedactedPayload), record.RedactedPayload); err != nil {
		return repository.fail(ctx, err)
	}
	if err := pruneRetentionTx(ctx, tx, time.Now()); err != nil {
		return repository.fail(ctx, err)
	}
	if err := tx.Commit(); err != nil {
		return repository.fail(ctx, err)
	}
	return repository.markReady()
}

// PruneRetention enforces the fixed retention windows and budgets during startup
// and shutdown, in addition to append paths. Older records are removed first;
// timestamp ties are ordered by monotonically increasing SQLite row ID.
func (repository *Repository) PruneRetention(ctx context.Context, now time.Time) Result {
	if ctx == nil || ctx.Err() != nil {
		return repository.fail(ctx, context.Canceled)
	}
	db := repository.database()
	if db == nil {
		return repository.fail(ctx, errStorageUnavailable)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return repository.fail(ctx, err)
	}
	defer tx.Rollback()
	if err := pruneRetentionTx(ctx, tx, now); err != nil {
		return repository.fail(ctx, err)
	}
	if err := tx.Commit(); err != nil {
		return repository.fail(ctx, err)
	}
	return repository.markReady()
}

func (repository *Repository) database() *sql.DB {
	if repository == nil {
		return nil
	}
	repository.mu.RLock()
	db := repository.db
	repository.mu.RUnlock()
	return db
}

func pruneRetentionTx(ctx context.Context, tx *sql.Tx, now time.Time) error {
	activationCutoff := now.Unix() - ActivationRetentionSeconds
	if _, err := tx.ExecContext(ctx, `DELETE FROM activation_metadata WHERE activated_at < ?`, activationCutoff); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM activation_metadata WHERE activation_id IN (
		SELECT activation_id FROM (
			SELECT activation_id, ROW_NUMBER() OVER (
				PARTITION BY profile_id ORDER BY activated_at DESC, activation_id DESC
			) AS retention_rank FROM activation_metadata
		) WHERE retention_rank > ?
	)`, MaxActivationRowsPerProfile); err != nil {
		return err
	}
	diagnosticCutoff := now.UnixMilli() - DiagnosticsRetentionMillis
	if _, err := tx.ExecContext(ctx, `DELETE FROM diagnostics WHERE occurred_at < ?`, diagnosticCutoff); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM diagnostics WHERE diagnostic_id IN (
		SELECT diagnostic_id FROM (
			SELECT diagnostic_id,
				SUM(payload_bytes) OVER (ORDER BY occurred_at DESC, diagnostic_id DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS retained_bytes
			FROM diagnostics
		) WHERE retained_bytes > ?
	)`, MaxDiagnosticsPayloadBytes)
	return err
}
