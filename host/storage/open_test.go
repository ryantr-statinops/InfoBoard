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

func TestOpenCreatesRestrictedDatabaseAndReopensIdempotently(t *testing.T) {
	directory := t.TempDir()
	repository, result := Open(context.Background(), directory)
	if repository == nil || result.Status != StatusReady || result.ErrorClass != "" {
		t.Fatalf("first open repository nil:%t result:%+v", repository == nil, result)
	}
	path := filepath.Join(directory, DatabaseFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("database permissions = %04o, want 0600", info.Mode().Perm())
	}
	if result := CheckIntegrity(context.Background(), repository.database()); result.Status != StatusReady {
		t.Fatalf("opened database integrity = %+v", result)
	}

	reopened, reopenResult := Open(context.Background(), directory)
	if reopened == nil || reopenResult.Status != StatusReady || reopenResult.ErrorClass != "" {
		t.Fatalf("reopen repository nil:%t result:%+v", reopened == nil, reopenResult)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenQuarantinesCorruptFileAndRebuildsSchema(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, DatabaseFileName)
	original := []byte("corrupt owned sqlite bytes")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	repository, result := Open(context.Background(), directory)
	if repository == nil || result.Status != StatusRebuilt || result.ErrorClass != "" {
		t.Fatalf("recovery repository nil:%t result:%+v", repository == nil, result)
	}
	defer repository.Close()
	if check := CheckIntegrity(context.Background(), repository.database()); check.Status != StatusReady {
		t.Fatalf("rebuilt database integrity = %+v", check)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var preserved []byte
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) >= len(quarantineDirectoryPrefix) && entry.Name()[:len(quarantineDirectoryPrefix)] == quarantineDirectoryPrefix {
			preserved, err = os.ReadFile(filepath.Join(directory, entry.Name(), DatabaseFileName))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if string(preserved) != string(original) {
		t.Fatalf("preserved quarantine bytes = %q", preserved)
	}
}

func TestOpenDoesNotRewriteFutureOrUnownedDatabase(t *testing.T) {
	ctx := context.Background()
	futureDirectory := t.TempDir()
	futurePath := filepath.Join(futureDirectory, DatabaseFileName)
	futureDB, err := sql.Open("sqlite", futurePath)
	if err != nil {
		t.Fatal(err)
	}
	futureDB.SetMaxOpenConns(1)
	migrationResult := migrations.Run(ctx, futureDB, migrations.Options{
		ApplicationID: DatabaseApplicationID,
		OwnerID:       DatabaseOwnerID,
		BusyTimeout:   migrations.DefaultBusyTimeout,
	}, InitialMigrations())
	if migrationResult.State != migrations.StateReady {
		t.Fatalf("setup migration = %+v", migrationResult)
	}
	if _, err := futureDB.Exec("PRAGMA user_version = 2"); err != nil {
		t.Fatal(err)
	}
	if err := futureDB.Close(); err != nil {
		t.Fatal(err)
	}
	futureBytes, err := os.ReadFile(futurePath)
	if err != nil {
		t.Fatal(err)
	}
	if futureRepository, result := Open(ctx, futureDirectory); futureRepository != nil || result.Status != StatusUpgradeRequired {
		t.Fatalf("future schema open repository nil:%t result:%+v", futureRepository == nil, result)
	}
	afterFutureBytes, err := os.ReadFile(futurePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterFutureBytes) != string(futureBytes) {
		t.Fatal("future schema file changed during open")
	}

	unownedDirectory := t.TempDir()
	unownedPath := filepath.Join(unownedDirectory, DatabaseFileName)
	unownedDB, err := sql.Open("sqlite", unownedPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unownedDB.Exec("CREATE TABLE unrelated_sentinel(value TEXT)"); err != nil {
		t.Fatal(err)
	}
	if err := unownedDB.Close(); err != nil {
		t.Fatal(err)
	}
	unownedBytes, err := os.ReadFile(unownedPath)
	if err != nil {
		t.Fatal(err)
	}
	if repository, result := Open(ctx, unownedDirectory); repository != nil || result.Status != StatusQuarantined || result.ErrorClass != ErrorSchemaIntegrity {
		t.Fatalf("unowned database open repository nil:%t result:%+v", repository == nil, result)
	}
	afterUnownedBytes, err := os.ReadFile(unownedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterUnownedBytes) != string(unownedBytes) {
		t.Fatal("unowned database changed during open")
	}
}
