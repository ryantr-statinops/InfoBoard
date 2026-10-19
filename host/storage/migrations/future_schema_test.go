package migrations_test

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/storage"
	"github.com/ryantr-statinops/InfoBoard/host/storage/migrations"
)

func TestFutureSchemaVersionPreservesOwnedDatabaseByteForByte(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if result := migrations.Run(context.Background(), db, initialOptions(), storage.InitialMigrations()); result.State != migrations.StateReady {
		t.Fatalf("initialize database result=%+v", result)
	}
	if _, err := db.Exec(`INSERT INTO configuration(profile_id, result_limit, ranking_model_version) VALUES (?, 9, 'rank-current')`, "123e4567-e89b-42d3-a456-426614174000"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 42`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	futureDB, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	futureDB.SetMaxOpenConns(1)
	result := migrations.Run(context.Background(), futureDB, initialOptions(), storage.InitialMigrations())
	if result.State != migrations.StateUpgradeRequired || result.ErrorClass != migrations.ErrorFutureSchemaVersion || result.CurrentVersion != 42 {
		t.Fatalf("future schema outcome=%+v", result)
	}
	if err := futureDB.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("future-schema inspection rewrote the database file")
	}
}
