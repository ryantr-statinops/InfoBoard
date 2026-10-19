package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/storage/migrations"
	_ "modernc.org/sqlite"
)

func TestQuarantineAndRebuildPreservesCorruptDatabase(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	path := filepath.Join(directory, DatabaseFileName)
	original := []byte("not a sqlite database")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)

	integrity := CheckIntegrity(ctx, db)
	if integrity.Status != StatusQuarantined || integrity.ErrorClass != ErrorDatabaseCorrupt {
		t.Fatalf("corrupt file status = %+v", integrity)
	}

	rebuilt, result := QuarantineAndRebuild(ctx, path, db)
	if rebuilt == nil || result.Status != StatusRebuilt || !result.Quarantined || !result.QuarantineRetained {
		t.Fatalf("recovery result = %+v, database nil = %t", result, rebuilt == nil)
	}
	defer rebuilt.Close()
	if result.ErrorClass != "" {
		t.Fatalf("successful recovery has error class %q", result.ErrorClass)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var quarantine string
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) >= len(quarantineDirectoryPrefix) && entry.Name()[:len(quarantineDirectoryPrefix)] == quarantineDirectoryPrefix {
			quarantine = filepath.Join(directory, entry.Name())
		}
	}
	if quarantine == "" {
		t.Fatal("quarantine directory was not retained")
	}
	preserved, err := os.ReadFile(filepath.Join(quarantine, DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(preserved) != string(original) {
		t.Fatalf("quarantined bytes = %q, want original bytes", preserved)
	}
	owner, err := os.ReadFile(filepath.Join(quarantine, "owner.txt"))
	if err != nil || string(owner) != DatabaseOwnerID+"\n" {
		t.Fatalf("quarantine owner marker = %q, err = %v", owner, err)
	}

	var appID int64
	if err := rebuilt.QueryRowContext(ctx, "PRAGMA application_id").Scan(&appID); err != nil {
		t.Fatal(err)
	}
	if appID != DatabaseApplicationID {
		t.Fatalf("rebuilt application_id = %d, want %d", appID, DatabaseApplicationID)
	}
	if check := CheckIntegrity(ctx, rebuilt); check.Status != StatusReady {
		t.Fatalf("rebuilt database integrity = %+v", check)
	}
	var tables int
	if err := rebuilt.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'").Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != len(OwnedTableNames()) {
		t.Fatalf("rebuilt owned tables = %d, want %d", tables, len(OwnedTableNames()))
	}
}

func TestQuarantineAndRebuildRefusesHealthyDatabase(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), DatabaseFileName)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	migrationResult := migrations.Run(ctx, db, migrations.Options{
		ApplicationID: DatabaseApplicationID,
		OwnerID:       DatabaseOwnerID,
		BusyTimeout:   migrations.DefaultBusyTimeout,
	}, InitialMigrations())
	if migrationResult.State != migrations.StateReady {
		t.Fatalf("migration state = %+v", migrationResult)
	}

	rebuilt, result := QuarantineAndRebuild(ctx, path, db)
	if rebuilt != nil || result.Status != StatusReady || result.Quarantined {
		t.Fatalf("healthy database recovery result = %+v, rebuilt nil = %t", result, rebuilt == nil)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("healthy database was moved: %v", err)
	}
}

func TestQuarantineAndRebuildRejectsUnsafePath(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	path := filepath.Join(directory, DatabaseFileName)
	if err := os.WriteFile(path, []byte("not sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	resultDB, result := QuarantineAndRebuild(ctx, filepath.Join(directory, "unrelated.sqlite"), db)
	if resultDB != nil || result.ErrorClass != ErrorUnsafeDatabasePath {
		t.Fatalf("unsafe path result = %+v, rebuilt nil = %t", result, resultDB == nil)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("refused recovery changed original path: %v", err)
	}
}
