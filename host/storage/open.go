package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/storage/migrations"
	_ "modernc.org/sqlite"
)

// Open creates or opens the fixed InfoBoard database inside an absolute runtime
// data directory, validates ownership/schema, migrates transactionally, and
// prunes retained data before returning a repository. Failure never exposes the
// path or driver error; callers can continue with in-memory query state.
func Open(ctx context.Context, dataDirectory string) (*Repository, Result) {
	if ctx == nil || ctx.Err() != nil {
		return nil, Result{Status: StatusDegraded, ErrorClass: ErrorStorageTimeout, Retryable: true}
	}
	if !filepath.IsAbs(dataDirectory) {
		return nil, Result{Status: StatusDegraded, ErrorClass: ErrorUnsafeDatabasePath}
	}
	dataDirectory = filepath.Clean(dataDirectory)
	if err := os.MkdirAll(dataDirectory, 0o700); err != nil {
		return nil, Result{Status: StatusDegraded, ErrorClass: ErrorStorageUnavailable, Retryable: true}
	}
	directoryInfo, err := os.Lstat(dataDirectory)
	if err != nil || !directoryInfo.IsDir() || directoryInfo.Mode()&os.ModeSymlink != 0 {
		return nil, Result{Status: StatusDegraded, ErrorClass: ErrorUnsafeDatabasePath}
	}
	databasePath := filepath.Join(dataDirectory, DatabaseFileName)
	created := false
	fileInfo, err := os.Lstat(databasePath)
	if errors.Is(err, os.ErrNotExist) {
		file, createErr := os.OpenFile(databasePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if createErr != nil {
			return nil, Result{Status: StatusDegraded, ErrorClass: ErrorStorageUnavailable, Retryable: true}
		}
		if closeErr := file.Close(); closeErr != nil {
			return nil, Result{Status: StatusDegraded, ErrorClass: ErrorStorageUnavailable, Retryable: true}
		}
		created = true
	} else if err != nil || !fileInfo.Mode().IsRegular() {
		return nil, Result{Status: StatusDegraded, ErrorClass: ErrorUnsafeDatabasePath}
	}

	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		if created {
			_ = os.Remove(databasePath)
		}
		return nil, Result{Status: StatusDegraded, ErrorClass: ErrorStorageUnavailable, Retryable: true}
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if !created {
		var currentVersion int64
		if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&currentVersion); err != nil {
			integrity := CheckIntegrity(ctx, db)
			if isQuarantinableIntegrity(ctx, integrity, db) {
				return rebuildOwnedDatabase(ctx, databasePath, db)
			}
			_ = db.Close()
			return nil, integrity
		}
		if currentVersion > int64(SupportedSchemaVersion) {
			_ = db.Close()
			return nil, Result{Status: StatusUpgradeRequired, ErrorClass: ErrorSchemaUpgradeRequired}
		}
		integrity := CheckIntegrity(ctx, db)
		if isQuarantinableIntegrity(ctx, integrity, db) {
			return rebuildOwnedDatabase(ctx, databasePath, db)
		}
		if integrity.Status != StatusReady {
			_ = db.Close()
			return nil, integrity
		}
	}

	options := migrations.Options{ApplicationID: DatabaseApplicationID, OwnerID: DatabaseOwnerID, BusyTimeout: migrations.DefaultBusyTimeout}
	migrationResult := migrations.Run(ctx, db, options, InitialMigrations())
	if migrationResult.State != migrations.StateReady {
		_ = db.Close()
		if created {
			_ = os.Remove(databasePath)
		}
		if migrationResult.State == migrations.StateUpgradeRequired {
			return nil, Result{Status: StatusUpgradeRequired, ErrorClass: ErrorSchemaUpgradeRequired}
		}
		if migrationResult.State == migrations.StateQuarantined {
			return nil, Result{Status: StatusQuarantined, ErrorClass: ErrorUnsafeDatabasePath}
		}
		class := ErrorStorageFailure
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
			class = ErrorStorageTimeout
		}
		return nil, Result{Status: StatusDegraded, ErrorClass: class, Retryable: true}
	}

	integrity := CheckIntegrity(ctx, db)
	if isQuarantinableIntegrity(ctx, integrity, db) {
		return rebuildOwnedDatabase(ctx, databasePath, db)
	}
	if integrity.Status != StatusReady {
		_ = db.Close()
		return nil, integrity
	}
	if err := os.Chmod(databasePath, 0o600); err != nil {
		_ = db.Close()
		return nil, Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
	}

	repository := NewRepository(db)
	pruneResult := repository.PruneRetention(ctx, time.Now())
	if pruneResult.Status != StatusReady {
		return repository, pruneResult
	}
	return repository, Result{Status: StatusReady}
}

func isQuarantinableIntegrity(ctx context.Context, result Result, db *sql.DB) bool {
	if result.Status != StatusQuarantined {
		return false
	}
	if result.ErrorClass == ErrorDatabaseCorrupt {
		return true
	}
	return result.ErrorClass == ErrorSchemaIntegrity && databaseClaimsInfoBoard(ctx, db)
}

func databaseClaimsInfoBoard(ctx context.Context, db *sql.DB) bool {
	var applicationID int64
	if err := db.QueryRowContext(ctx, "PRAGMA application_id").Scan(&applicationID); err != nil || applicationID != DatabaseApplicationID {
		return false
	}
	var owner string
	if err := db.QueryRowContext(ctx, "SELECT value FROM schema_meta WHERE key = 'owner_id'").Scan(&owner); err != nil {
		return false
	}
	return owner == DatabaseOwnerID
}

func rebuildOwnedDatabase(ctx context.Context, databasePath string, db *sql.DB) (*Repository, Result) {
	rebuiltDB, recovery := QuarantineAndRebuild(ctx, databasePath, db)
	if rebuiltDB == nil || recovery.Status != StatusRebuilt {
		return nil, Result{Status: recovery.Status, ErrorClass: recovery.ErrorClass, Retryable: recovery.Retryable}
	}
	if err := os.Chmod(databasePath, 0o600); err != nil {
		_ = rebuiltDB.Close()
		return nil, Result{Status: StatusDegraded, ErrorClass: ErrorStorageFailure, Retryable: true}
	}
	repository := NewRepository(rebuiltDB)
	pruneResult := repository.PruneRetention(ctx, time.Now())
	if pruneResult.Status != StatusReady {
		return repository, pruneResult
	}
	return repository, Result{Status: StatusRebuilt}
}
