// Package migrations applies ordered SQLite migrations as one bounded transaction.
package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DefaultBusyTimeout  = 3 * time.Second
	MaxBusyTimeout      = 5 * time.Second
	maxMigrationIDBytes = 128
	maxOwnerIDBytes     = 128
)

type State string

const (
	StateReady           State = "ready"
	StateDegraded        State = "degraded"
	StateUpgradeRequired State = "upgrade_required"
	StateQuarantined     State = "quarantined"
)

type ErrorClass string

const (
	ErrorNone                ErrorClass = ""
	ErrorInvalidRegistry     ErrorClass = "invalid_migration_registry"
	ErrorStorageUnavailable  ErrorClass = "storage_unavailable"
	ErrorStorageTimeout      ErrorClass = "storage_timeout"
	ErrorMigrationFailed     ErrorClass = "migration_failed"
	ErrorDatabaseNotOwned    ErrorClass = "database_not_owned"
	ErrorSchemaIntegrity     ErrorClass = "schema_integrity_failed"
	ErrorFutureSchemaVersion ErrorClass = "future_schema_version"
)

// Migration is immutable input to the runner. Checksum is derived from Version,
// ID, and every ordered SQL statement; no caller-provided checksum is trusted.
type Migration struct {
	Version    uint64
	ID         string
	Statements []string
}

// Options binds a migration run to one owned SQLite file and a strict lock wait.
type Options struct {
	ApplicationID int64
	OwnerID       string
	BusyTimeout   time.Duration
}

// Result contains only bounded lifecycle metadata; it never carries a driver
// error, SQL statement, or database path.
type Result struct {
	State            State
	ErrorClass       ErrorClass
	Retryable        bool
	CurrentVersion   uint64
	SupportedVersion uint64
	AppliedVersions  []uint64
}

// Checksum returns the deterministic SHA-256 identity of one migration.
func (migration Migration) Checksum() string {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%d\x00%s\x00", migration.Version, migration.ID)
	for _, statement := range migration.Statements {
		_, _ = hash.Write([]byte(statement))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// Run verifies ownership and migration history, then applies every pending
// migration in order under BEGIN IMMEDIATE. The schema version and application
// ID change only inside the transaction that also records every applied checksum.
func Run(ctx context.Context, db *sql.DB, options Options, registry []Migration) Result {
	supported, valid := validateRegistry(registry)
	if !valid || options.ApplicationID <= 0 || !bounded(options.OwnerID, maxOwnerIDBytes) {
		return Result{State: StateDegraded, ErrorClass: ErrorInvalidRegistry, SupportedVersion: supported}
	}
	if options.BusyTimeout == 0 {
		options.BusyTimeout = DefaultBusyTimeout
	}
	if options.BusyTimeout < 0 || options.BusyTimeout > MaxBusyTimeout {
		return Result{State: StateDegraded, ErrorClass: ErrorInvalidRegistry, SupportedVersion: supported}
	}
	if ctx == nil || db == nil {
		return Result{State: StateDegraded, ErrorClass: ErrorStorageUnavailable, Retryable: true, SupportedVersion: supported}
	}

	connection, err := db.Conn(ctx)
	if err != nil {
		return failure(ctx, err, supported, 0)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d", options.BusyTimeout.Milliseconds())); err != nil {
		return failure(ctx, err, supported, 0)
	}
	if _, err = connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return failure(ctx, err, supported, 0)
	}
	transactionOpen := true
	rollback := func() {
		if transactionOpen {
			_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
			transactionOpen = false
		}
	}
	defer rollback()

	applicationID, err := pragmaInt64(ctx, connection, "application_id")
	if err != nil {
		return failure(ctx, err, supported, 0)
	}
	version, err := pragmaInt64(ctx, connection, "user_version")
	if err != nil || version < 0 {
		if err != nil {
			return failure(ctx, err, supported, 0)
		}
		return quarantined(ErrorSchemaIntegrity, supported, 0)
	}
	if applicationID != 0 && applicationID != options.ApplicationID {
		return quarantined(ErrorDatabaseNotOwned, supported, uint64(version))
	}
	if applicationID == 0 && version == 0 {
		count, countErr := userTableCount(ctx, connection)
		if countErr != nil {
			return failure(ctx, countErr, supported, 0)
		}
		if count != 0 {
			return quarantined(ErrorDatabaseNotOwned, supported, 0)
		}
	} else if applicationID != options.ApplicationID {
		return quarantined(ErrorDatabaseNotOwned, supported, uint64(version))
	}
	if uint64(version) > supported {
		return Result{State: StateUpgradeRequired, ErrorClass: ErrorFutureSchemaVersion, CurrentVersion: uint64(version), SupportedVersion: supported}
	}
	if applicationID == options.ApplicationID && version == 0 {
		return quarantined(ErrorSchemaIntegrity, supported, 0)
	}
	if version > 0 {
		if err := verifyHistory(ctx, connection, options.OwnerID, uint64(version), registry); err != nil {
			return quarantined(ErrorSchemaIntegrity, supported, uint64(version))
		}
	}
	if uint64(version) == supported {
		return Result{State: StateReady, CurrentVersion: uint64(version), SupportedVersion: supported}
	}

	applied := make([]uint64, 0, supported-uint64(version))
	for _, migration := range registry {
		if migration.Version <= uint64(version) {
			continue
		}
		for _, statement := range migration.Statements {
			if _, err := connection.ExecContext(ctx, statement); err != nil {
				return failure(ctx, err, supported, uint64(version))
			}
		}
		if _, err := connection.ExecContext(ctx,
			`INSERT INTO schema_migrations(version, migration_id, checksum, applied_at) VALUES (?, ?, ?, ?)`,
			migration.Version, migration.ID, migration.Checksum(), time.Now().Unix()); err != nil {
			return failure(ctx, err, supported, uint64(version))
		}
		if _, err := connection.ExecContext(ctx,
			`INSERT INTO schema_meta(key, value) VALUES ('owner_id', ?), ('schema_version', ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			options.OwnerID, fmt.Sprint(migration.Version)); err != nil {
			return failure(ctx, err, supported, uint64(version))
		}
		applied = append(applied, migration.Version)
	}
	if _, err := connection.ExecContext(ctx, fmt.Sprintf("PRAGMA application_id = %d", options.ApplicationID)); err != nil {
		return failure(ctx, err, supported, uint64(version))
	}
	if _, err := connection.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", supported)); err != nil {
		return failure(ctx, err, supported, uint64(version))
	}
	committedApplicationID, err := pragmaInt64(ctx, connection, "application_id")
	if err != nil || committedApplicationID != options.ApplicationID {
		return quarantined(ErrorSchemaIntegrity, supported, uint64(version))
	}
	committedVersion, err := pragmaInt64(ctx, connection, "user_version")
	if err != nil || committedVersion != int64(supported) {
		return quarantined(ErrorSchemaIntegrity, supported, uint64(version))
	}
	if err := verifyHistory(ctx, connection, options.OwnerID, supported, registry); err != nil {
		return quarantined(ErrorSchemaIntegrity, supported, uint64(version))
	}
	if _, err := connection.ExecContext(ctx, "COMMIT"); err != nil {
		return failure(ctx, err, supported, uint64(version))
	}
	transactionOpen = false
	return Result{State: StateReady, CurrentVersion: supported, SupportedVersion: supported, AppliedVersions: applied}
}

func validateRegistry(registry []Migration) (uint64, bool) {
	if len(registry) == 0 {
		return 0, false
	}
	seenIDs := make(map[string]struct{}, len(registry))
	for index, migration := range registry {
		if migration.Version != uint64(index+1) || !bounded(migration.ID, maxMigrationIDBytes) || len(migration.Statements) == 0 {
			return uint64(len(registry)), false
		}
		if _, exists := seenIDs[migration.ID]; exists {
			return uint64(len(registry)), false
		}
		seenIDs[migration.ID] = struct{}{}
		for _, statement := range migration.Statements {
			if strings.TrimSpace(statement) == "" {
				return uint64(len(registry)), false
			}
		}
	}
	return uint64(len(registry)), true
}

func verifyHistory(ctx context.Context, connection *sql.Conn, ownerID string, current uint64, registry []Migration) error {
	var owner, schemaVersion string
	if err := connection.QueryRowContext(ctx, `SELECT value FROM schema_meta WHERE key = 'owner_id'`).Scan(&owner); err != nil {
		return err
	}
	if owner != ownerID {
		return sql.ErrNoRows
	}
	if err := connection.QueryRowContext(ctx, `SELECT value FROM schema_meta WHERE key = 'schema_version'`).Scan(&schemaVersion); err != nil {
		return err
	}
	if schemaVersion != fmt.Sprint(current) {
		return sql.ErrNoRows
	}
	rows, err := connection.QueryContext(ctx, `SELECT version, migration_id, checksum FROM schema_migrations ORDER BY version`)
	if err != nil {
		return err
	}
	defer rows.Close()
	count := uint64(0)
	for rows.Next() {
		var version int64
		var id, checksum string
		if err := rows.Scan(&version, &id, &checksum); err != nil {
			return err
		}
		if version <= 0 || uint64(version) != count+1 || uint64(version) > current {
			return sql.ErrNoRows
		}
		migration := registry[version-1]
		if id != migration.ID || checksum != migration.Checksum() {
			return sql.ErrNoRows
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count != current {
		return sql.ErrNoRows
	}
	return nil
}

func userTableCount(ctx context.Context, connection *sql.Conn) (int, error) {
	var count int
	err := connection.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&count)
	return count, err
}

func pragmaInt64(ctx context.Context, connection *sql.Conn, name string) (int64, error) {
	var value int64
	err := connection.QueryRowContext(ctx, "PRAGMA "+name).Scan(&value)
	return value, err
}

func bounded(value string, maxBytes int) bool {
	return value != "" && utf8.ValidString(value) && len(value) <= maxBytes
}

func failure(ctx context.Context, err error, supported, current uint64) Result {
	class := ErrorMigrationFailed
	if ctx != nil && ctx.Err() != nil {
		class = ErrorStorageTimeout
	}
	return Result{State: StateDegraded, ErrorClass: class, Retryable: class != ErrorInvalidRegistry, CurrentVersion: current, SupportedVersion: supported}
}

func quarantined(class ErrorClass, supported, current uint64) Result {
	return Result{State: StateQuarantined, ErrorClass: class, CurrentVersion: current, SupportedVersion: supported}
}
