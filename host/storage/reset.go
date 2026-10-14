package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// ResetProfile removes only one profile's durable preferences, activation
// metadata, and profile-scoped diagnostics. Global diagnostics remain available.
func (repository *Repository) ResetProfile(ctx context.Context, profileID string) Result {
	profile, err := domain.ParseProfileID(profileID)
	if err != nil {
		return Result{Status: repository.currentStatus(), ErrorClass: ErrorProfileMismatch}
	}
	return repository.withResetTransaction(ctx, func(_ context.Context, tx *sql.Tx) error {
		for _, table := range []string{tableDiagnostics, tableActivationMetadata, tableConfiguration} {
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE profile_id = ?`, string(profile)); err != nil {
				return err
			}
		}
		return nil
	})
}

// ResetAll clears every user-owned storage row while retaining schema ownership
// and migration history so the database remains safely reopenable. It does not
// clear the in-memory live projection; the caller must request a fresh snapshot.
func (repository *Repository) ResetAll(ctx context.Context) Result {
	return repository.withResetTransaction(ctx, clearUserRows)
}

func clearUserRows(ctx context.Context, tx *sql.Tx) error {
	for _, table := range []string{tableDiagnostics, tableActivationMetadata, tableConfiguration, tableInstallationState} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table); err != nil {
			return err
		}
	}
	return nil
}

func (repository *Repository) withResetTransaction(ctx context.Context, reset func(context.Context, *sql.Tx) error) Result {
	if repository == nil {
		return Result{Status: StatusDegraded, ErrorClass: ErrorStorageUnavailable, Retryable: true}
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
	if err := reset(ctx, tx); err != nil {
		return repository.fail(ctx, err)
	}
	if err := tx.Commit(); err != nil {
		return repository.fail(ctx, err)
	}
	return repository.markReady()
}

// Uninstall removes the fixed InfoBoard database, SQLite sidecars, and only
// quarantine directories bearing InfoBoard's owner marker. It validates both
// database path and ownership before deletion, and can retry after partial file
// cleanup without needing the original repository handle.
func (repository *Repository) Uninstall(ctx context.Context, databasePath string) Result {
	if repository == nil {
		return Result{Status: StatusDegraded, ErrorClass: ErrorStorageUnavailable, Retryable: true}
	}
	if ctx == nil || ctx.Err() != nil {
		return repository.fail(ctx, context.Canceled)
	}
	if !filepath.IsAbs(databasePath) || filepath.Base(databasePath) != DatabaseFileName {
		return Result{Status: StatusQuarantined, ErrorClass: ErrorUnsafeDatabasePath}
	}
	databasePath = filepath.Clean(databasePath)
	parent := filepath.Dir(databasePath)
	parentInfo, err := os.Lstat(parent)
	if err != nil || !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 {
		return Result{Status: StatusQuarantined, ErrorClass: ErrorUnsafeDatabasePath}
	}
	fileInfo, err := os.Lstat(databasePath)
	if errors.Is(err, os.ErrNotExist) {
		if err := removeDatabaseSidecars(databasePath); err != nil {
			return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
		}
		if err := removeOwnedQuarantines(parent); err != nil {
			return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
		}
		repository.markDisabled()
		return Result{Status: StatusDisabled}
	}
	if err != nil || !fileInfo.Mode().IsRegular() {
		return Result{Status: StatusDegraded, ErrorClass: ErrorUnsafeDatabasePath, Retryable: true}
	}

	db := repository.database()
	temporaryHandle := false
	if db == nil {
		db, err = sql.Open("sqlite", databasePath)
		if err != nil {
			return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
		}
		db.SetMaxOpenConns(1)
		temporaryHandle = true
	}
	if !databaseOwnedAt(ctx, db, databasePath) {
		if temporaryHandle {
			_ = db.Close()
		}
		return Result{Status: StatusQuarantined, ErrorClass: ErrorUnsafeDatabasePath}
	}

	if temporaryHandle {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			_ = db.Close()
			return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
		}
		if err := clearUserRows(ctx, tx); err != nil {
			_ = tx.Rollback()
			_ = db.Close()
			return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
		}
		if err := tx.Commit(); err != nil {
			_ = db.Close()
			return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
		}
		if err := db.Close(); err != nil {
			return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
		}
	} else {
		if result := repository.ResetAll(ctx); result.Status != StatusReady {
			return result
		}
		repository.mu.Lock()
		if repository.db != db {
			repository.mu.Unlock()
			return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
		}
		if err := db.Close(); err != nil {
			repository.status = StatusDegraded
			repository.errorClass = ErrorStorageFailure
			repository.mu.Unlock()
			return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
		}
		repository.db = nil
		repository.status = StatusDisabled
		repository.errorClass = ""
		repository.mu.Unlock()
	}

	if err := removeDatabaseFiles(databasePath); err != nil {
		return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
	}
	if err := removeOwnedQuarantines(parent); err != nil {
		return Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
	}
	repository.markDisabled()
	return Result{Status: StatusDisabled}
}

func (repository *Repository) markDisabled() {
	repository.mu.Lock()
	repository.db = nil
	repository.status = StatusDisabled
	repository.errorClass = ""
	repository.mu.Unlock()
}

func databaseOwnedAt(ctx context.Context, db *sql.DB, databasePath string) bool {
	rows, err := db.QueryContext(ctx, `PRAGMA database_list`)
	if err != nil {
		return false
	}
	mainMatches := false
	for rows.Next() {
		var sequence int
		var name, path string
		if err := rows.Scan(&sequence, &name, &path); err != nil {
			_ = rows.Close()
			return false
		}
		if name == "main" && path != "" && filepath.Clean(path) == databasePath {
			mainMatches = true
		}
	}
	rowsErr := rows.Err()
	closeErr := rows.Close()
	if rowsErr != nil || closeErr != nil || !mainMatches {
		return false
	}
	var applicationID int64
	if err := db.QueryRowContext(ctx, `PRAGMA application_id`).Scan(&applicationID); err != nil || applicationID != DatabaseApplicationID {
		return false
	}
	var owner string
	if err := db.QueryRowContext(ctx, `SELECT value FROM schema_meta WHERE key = 'owner_id'`).Scan(&owner); err != nil {
		return false
	}
	return owner == DatabaseOwnerID
}

func removeDatabaseFiles(databasePath string) error {
	for _, path := range []string{databasePath, databasePath + "-journal", databasePath + "-wal", databasePath + "-shm"} {
		if err := removeRegularFileIfPresent(path); err != nil {
			return err
		}
	}
	return nil
}

func removeDatabaseSidecars(databasePath string) error {
	for _, path := range []string{databasePath + "-journal", databasePath + "-wal", databasePath + "-shm"} {
		if err := removeRegularFileIfPresent(path); err != nil {
			return err
		}
	}
	return nil
}

func removeRegularFileIfPresent(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() {
		return os.ErrPermission
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func removeOwnedQuarantines(parent string) error {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || len(entry.Name()) < len(quarantineDirectoryPrefix) || entry.Name()[:len(quarantineDirectoryPrefix)] != quarantineDirectoryPrefix {
			continue
		}
		directory := filepath.Join(parent, entry.Name())
		info, err := os.Lstat(directory)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		marker := filepath.Join(directory, "owner.txt")
		markerInfo, err := os.Lstat(marker)
		if err != nil || !markerInfo.Mode().IsRegular() {
			continue
		}
		owner, err := os.ReadFile(marker)
		if err != nil || string(owner) != DatabaseOwnerID+"\n" {
			continue
		}
		if err := os.RemoveAll(directory); err != nil {
			return err
		}
	}
	return nil
}
