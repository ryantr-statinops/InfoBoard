package storage_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/storage"
	"github.com/ryantr-statinops/InfoBoard/host/storage/migrations"
)

func newInMemoryRepository(t *testing.T) (*sql.DB, *storage.Repository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	migrationResult := migrations.Run(context.Background(), db, migrations.Options{
		ApplicationID: storage.DatabaseApplicationID,
		OwnerID:       storage.DatabaseOwnerID,
		BusyTimeout:   migrations.DefaultBusyTimeout,
	}, storage.InitialMigrations())
	if migrationResult.State != migrations.StateReady {
		_ = db.Close()
		t.Fatalf("in-memory fixture migration = %+v", migrationResult)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, storage.NewRepository(db)
}
