package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/storage/migrations"
	_ "modernc.org/sqlite"
	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const quarantineDirectoryPrefix = ".infoboard-quarantine-"

var requiredSchemaColumns = []struct {
	table   string
	columns []string
}{
	{tableSchemaMeta, []string{"key", "value"}},
	{tableSchemaMigrations, []string{"version", "migration_id", "checksum", "applied_at"}},
	{tableConfiguration, []string{"profile_id", "result_limit", "density", "theme", "show_context_labels", "persist_activation_recency", "protocol_version", "ranking_model_version", "updated_at"}},
	{tableInstallationState, []string{"installation_key", "extension_version", "host_version", "browser_family", "protocol_version", "ranking_model_version", "host_registration_state", "last_successful_migration", "repair_state", "updated_at"}},
	{tableActivationMetadata, []string{"activation_id", "profile_id", "context_kind", "tab_id", "domain", "activated_at", "source"}},
	{tableDiagnostics, []string{"diagnostic_id", "occurred_at", "profile_id", "event_kind", "error_class", "retryable", "duration_ms", "projection_revision", "protocol_version", "ranking_model_version", "schema_version", "payload_bytes", "redacted_payload"}},
}

// RecoveryResult reports quarantine and rebuild status without returning a path,
// SQLite error string, or any data derived from a stored record.
type RecoveryResult struct {
	Status             Status
	Quarantined        bool
	ErrorClass         ErrorClass
	Retryable          bool
	QuarantineRetained bool
}

// CheckIntegrity verifies SQLite's page integrity and the required version-1
// schema shape. A future schema must be rejected by migrations.Run before this
// function is called, so it is never mistaken for corruption.
func CheckIntegrity(ctx context.Context, db *sql.DB) Result {
	if ctx == nil || db == nil {
		return Result{Status: StatusDegraded, ErrorClass: ErrorStorageUnavailable, Retryable: true}
	}
	rows, err := db.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return integrityFailure(ctx, err)
	}
	checked := false
	for rows.Next() {
		var outcome string
		if err := rows.Scan(&outcome); err != nil {
			rows.Close()
			return integrityFailure(ctx, err)
		}
		checked = true
		if outcome != "ok" {
			rows.Close()
			return Result{Status: StatusQuarantined, ErrorClass: ErrorDatabaseCorrupt}
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return integrityFailure(ctx, err)
	}
	if err := rows.Close(); err != nil {
		return integrityFailure(ctx, err)
	}
	if !checked {
		return Result{Status: StatusQuarantined, ErrorClass: ErrorDatabaseCorrupt}
	}
	if err := verifyRequiredSchema(ctx, db); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Result{Status: StatusQuarantined, ErrorClass: ErrorSchemaIntegrity}
		}
		return integrityFailure(ctx, err)
	}
	return Result{Status: StatusReady}
}

func verifyRequiredSchema(ctx context.Context, db *sql.DB) error {
	actualTables := make(map[string]struct{}, len(ownedTables))
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			rows.Close()
			return err
		}
		actualTables[table] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(actualTables) != len(ownedTables) {
		return sql.ErrNoRows
	}
	for _, required := range requiredSchemaColumns {
		if _, exists := actualTables[required.table]; !exists {
			return sql.ErrNoRows
		}
		columns := make(map[string]struct{}, len(required.columns))
		columnRows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info('%s')", required.table))
		if err != nil {
			return err
		}
		for columnRows.Next() {
			var cid, notNull, primaryKey int
			var name, columnType string
			var defaultValue sql.NullString
			if err := columnRows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				columnRows.Close()
				return err
			}
			columns[name] = struct{}{}
		}
		if err := columnRows.Err(); err != nil {
			columnRows.Close()
			return err
		}
		if err := columnRows.Close(); err != nil {
			return err
		}
		for _, name := range required.columns {
			if _, exists := columns[name]; !exists {
				return sql.ErrNoRows
			}
		}
	}
	return nil
}

func integrityFailure(ctx context.Context, err error) Result {
	if ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return Result{Status: StatusDegraded, ErrorClass: ErrorStorageTimeout, Retryable: true}
	}
	var sqliteError *sqlite.Error
	if errors.As(err, &sqliteError) {
		code := sqliteError.Code() & 0xff
		if code == sqlite3.SQLITE_CORRUPT || code == sqlite3.SQLITE_NOTADB {
			return Result{Status: StatusQuarantined, ErrorClass: ErrorDatabaseCorrupt}
		}
	}
	return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
}

// QuarantineAndRebuild may be called only after integrity/schema validation
// reports a quarantinable failure. It moves the owned database and SQLite sidecar
// files into a unique directory under the same parent, creates a mode-0600 database,
// and reapplies the current schema. The quarantine artifact is retained.
func QuarantineAndRebuild(ctx context.Context, databasePath string, db *sql.DB) (*sql.DB, RecoveryResult) {
	if ctx == nil || ctx.Err() != nil {
		return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorStorageTimeout, Retryable: true}
	}
	if db == nil {
		return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorStorageUnavailable, Retryable: true}
	}
	check := CheckIntegrity(ctx, db)
	if check.Status != StatusQuarantined || (check.ErrorClass != ErrorDatabaseCorrupt && check.ErrorClass != ErrorSchemaIntegrity) {
		return nil, RecoveryResult{Status: check.Status, ErrorClass: check.ErrorClass, Retryable: check.Retryable}
	}
	if check.ErrorClass == ErrorSchemaIntegrity && !databaseClaimsInfoBoard(ctx, db) {
		return nil, RecoveryResult{Status: StatusQuarantined, ErrorClass: ErrorUnsafeDatabasePath}
	}
	if !filepath.IsAbs(databasePath) || filepath.Base(databasePath) != DatabaseFileName {
		return nil, RecoveryResult{Status: StatusQuarantined, ErrorClass: ErrorUnsafeDatabasePath}
	}
	databasePath = filepath.Clean(databasePath)
	parent := filepath.Dir(databasePath)
	parentInfo, err := os.Lstat(parent)
	if err != nil || !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 {
		return nil, RecoveryResult{Status: StatusQuarantined, ErrorClass: ErrorUnsafeDatabasePath}
	}
	fileInfo, err := os.Lstat(databasePath)
	if err != nil || !fileInfo.Mode().IsRegular() {
		return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorQuarantineFailed, Retryable: true}
	}
	if db != nil {
		if err := db.Close(); err != nil {
			return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorQuarantineFailed, Retryable: true}
		}
	}

	stamp := time.Now().UTC().Format("20060102T150405Z")
	quarantineDirectory, err := os.MkdirTemp(parent, quarantineDirectoryPrefix+stamp+"-")
	if err != nil {
		return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorQuarantineFailed, Retryable: true}
	}
	moved := make([][2]string, 0, 4)
	rollbackMoves := func() bool {
		for index := len(moved) - 1; index >= 0; index-- {
			if err := os.Rename(moved[index][1], moved[index][0]); err != nil {
				return false
			}
		}
		return true
	}
	for _, source := range []string{databasePath, databasePath + "-journal", databasePath + "-wal", databasePath + "-shm"} {
		info, err := os.Lstat(source)
		if errors.Is(err, os.ErrNotExist) && source != databasePath {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			if rollbackMoves() {
				_ = os.Remove(quarantineDirectory)
			}
			return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorQuarantineFailed, Retryable: true}
		}
		target := filepath.Join(quarantineDirectory, filepath.Base(source))
		if err := os.Rename(source, target); err != nil {
			if rollbackMoves() {
				_ = os.Remove(quarantineDirectory)
			}
			return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorQuarantineFailed, Retryable: true}
		}
		moved = append(moved, [2]string{source, target})
	}

	marker := []byte(DatabaseOwnerID + "\n")
	if err := os.WriteFile(filepath.Join(quarantineDirectory, "owner.txt"), marker, 0o600); err != nil {
		if rollbackMoves() {
			_ = os.RemoveAll(quarantineDirectory)
		}
		return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorQuarantineFailed, Retryable: true}
	}
	freshFile, err := os.OpenFile(databasePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		if rollbackMoves() {
			_ = os.RemoveAll(quarantineDirectory)
		}
		return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorRecoveryFailed, Retryable: true}
	}
	if err := freshFile.Close(); err != nil {
		return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorRecoveryFailed, Retryable: true, QuarantineRetained: true}
	}

	freshDB, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorRecoveryFailed, Retryable: true, QuarantineRetained: true}
	}
	freshDB.SetMaxOpenConns(1)
	options := migrations.Options{ApplicationID: DatabaseApplicationID, OwnerID: DatabaseOwnerID, BusyTimeout: migrations.DefaultBusyTimeout}
	migrationResult := migrations.Run(ctx, freshDB, options, InitialMigrations())
	if migrationResult.State != migrations.StateReady {
		_ = freshDB.Close()
		return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorRecoveryFailed, Retryable: migrationResult.Retryable, QuarantineRetained: true}
	}
	integrity := CheckIntegrity(ctx, freshDB)
	if integrity.Status != StatusReady {
		_ = freshDB.Close()
		return nil, RecoveryResult{Status: StatusDegraded, ErrorClass: ErrorRecoveryFailed, Retryable: integrity.Retryable, QuarantineRetained: true}
	}
	return freshDB, RecoveryResult{Status: StatusRebuilt, Quarantined: true, QuarantineRetained: true}
}

// IsSQLiteCorruptionError classifies only the SQLite primary CORRUPT/NOTADB
// result codes; callers never need to inspect driver error strings.
func IsSQLiteCorruptionError(err error) bool {
	var sqliteError *sqlite.Error
	if !errors.As(err, &sqliteError) {
		return false
	}
	code := sqliteError.Code() & 0xff
	return code == sqlite3.SQLITE_CORRUPT || code == sqlite3.SQLITE_NOTADB
}

func (recovery RecoveryResult) String() string {
	return strings.Join([]string{string(recovery.Status), string(recovery.ErrorClass)}, ":")
}
