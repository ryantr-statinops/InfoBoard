package migrations_test

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/storage"
	"github.com/ryantr-statinops/InfoBoard/host/storage/migrations"
	_ "modernc.org/sqlite"
)

func openMigrationDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func initialOptions() migrations.Options {
	return migrations.Options{
		ApplicationID: storage.DatabaseApplicationID,
		OwnerID:       storage.DatabaseOwnerID,
		BusyTimeout:   migrations.DefaultBusyTimeout,
	}
}

func TestRunInitialMigrationAndIdempotentReopen(t *testing.T) {
	db := openMigrationDB(t)
	registry := storage.InitialMigrations()
	first := migrations.Run(context.Background(), db, initialOptions(), registry)
	if first.State != migrations.StateReady || first.CurrentVersion != storage.SupportedSchemaVersion || first.SupportedVersion != storage.SupportedSchemaVersion || !reflect.DeepEqual(first.AppliedVersions, []uint64{1}) {
		t.Fatalf("initial migration result=%+v", first)
	}
	var appID, userVersion int64
	if err := db.QueryRow(`PRAGMA application_id`).Scan(&appID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&userVersion); err != nil {
		t.Fatal(err)
	}
	if appID != storage.DatabaseApplicationID || userVersion != 1 {
		t.Fatalf("database markers app_id=%d user_version=%d", appID, userVersion)
	}
	var owner, version string
	if err := db.QueryRow(`SELECT value FROM schema_meta WHERE key = 'owner_id'`).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT value FROM schema_meta WHERE key = 'schema_version'`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if owner != storage.DatabaseOwnerID || version != "1" {
		t.Fatalf("schema metadata owner=%q version=%q", owner, version)
	}

	second := migrations.Run(context.Background(), db, initialOptions(), registry)
	if second.State != migrations.StateReady || second.CurrentVersion != 1 || len(second.AppliedVersions) != 0 {
		t.Fatalf("idempotent reopen result=%+v", second)
	}
}

func TestRunRollsBackEveryPendingMigrationOnFailure(t *testing.T) {
	db := openMigrationDB(t)
	registry := storage.InitialMigrations()
	first := migrations.Run(context.Background(), db, initialOptions(), registry)
	if first.State != migrations.StateReady {
		t.Fatalf("initial migration result=%+v", first)
	}
	registry = append(registry, migrations.Migration{
		Version: 2,
		ID:      "migration_v2",
		Statements: []string{
			`CREATE TABLE rollback_probe (id INTEGER PRIMARY KEY)`,
			`THIS IS NOT VALID SQL`,
		},
	})
	failed := migrations.Run(context.Background(), db, initialOptions(), registry)
	if failed.State != migrations.StateDegraded || failed.ErrorClass != migrations.ErrorMigrationFailed || !failed.Retryable || failed.CurrentVersion != 1 {
		t.Fatalf("failed migration result=%+v", failed)
	}
	var userVersion int64
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&userVersion); err != nil {
		t.Fatal(err)
	}
	if userVersion != 1 {
		t.Fatalf("failed migration advanced user_version=%d", userVersion)
	}
	var probeCount int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'rollback_probe'`).Scan(&probeCount); err != nil {
		t.Fatal(err)
	}
	var migrationCount int
	if err := db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatal(err)
	}
	if probeCount != 0 || migrationCount != 1 {
		t.Fatalf("failed migration partially committed: probe=%d history=%d", probeCount, migrationCount)
	}
	var schemaVersion string
	if err := db.QueryRow(`SELECT value FROM schema_meta WHERE key = 'schema_version'`).Scan(&schemaVersion); err != nil {
		t.Fatal(err)
	}
	if schemaVersion != "1" {
		t.Fatalf("schema metadata advanced to %q", schemaVersion)
	}
}

func TestRunRejectsChangedMigrationChecksumWithoutMutation(t *testing.T) {
	db := openMigrationDB(t)
	registry := storage.InitialMigrations()
	if result := migrations.Run(context.Background(), db, initialOptions(), registry); result.State != migrations.StateReady {
		t.Fatalf("initial migration result=%+v", result)
	}
	changed := storage.InitialMigrations()
	changed[0].Statements = append(changed[0].Statements, `CREATE TABLE unexpected (id INTEGER)`)
	result := migrations.Run(context.Background(), db, initialOptions(), changed)
	if result.State != migrations.StateQuarantined || result.ErrorClass != migrations.ErrorSchemaIntegrity || result.CurrentVersion != 1 {
		t.Fatalf("changed migration checksum result=%+v", result)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'unexpected'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("checksum mismatch mutated the database")
	}
}

func TestRunPreservesUnknownFutureVersionAndUnownedDatabase(t *testing.T) {
	registry := storage.InitialMigrations()
	futureDB := openMigrationDB(t)
	if _, err := futureDB.Exec(fmt.Sprintf("PRAGMA application_id = %d", storage.DatabaseApplicationID)); err != nil {
		t.Fatal(err)
	}
	if _, err := futureDB.Exec(`PRAGMA user_version = 42`); err != nil {
		t.Fatal(err)
	}
	future := migrations.Run(context.Background(), futureDB, initialOptions(), registry)
	if future.State != migrations.StateUpgradeRequired || future.ErrorClass != migrations.ErrorFutureSchemaVersion || future.CurrentVersion != 42 {
		t.Fatalf("future schema result=%+v", future)
	}
	var futureVersion int64
	if err := futureDB.QueryRow(`PRAGMA user_version`).Scan(&futureVersion); err != nil {
		t.Fatal(err)
	}
	var futureTables int
	if err := futureDB.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&futureTables); err != nil {
		t.Fatal(err)
	}
	if futureVersion != 42 || futureTables != 0 {
		t.Fatalf("future database changed: version=%d tables=%d", futureVersion, futureTables)
	}

	unownedDB := openMigrationDB(t)
	if _, err := unownedDB.Exec(`CREATE TABLE unrelated (id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	unowned := migrations.Run(context.Background(), unownedDB, initialOptions(), registry)
	if unowned.State != migrations.StateQuarantined || unowned.ErrorClass != migrations.ErrorDatabaseNotOwned {
		t.Fatalf("unowned database result=%+v", unowned)
	}
	var unrelated int
	if err := unownedDB.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'unrelated'`).Scan(&unrelated); err != nil {
		t.Fatal(err)
	}
	if unrelated != 1 {
		t.Fatal("unowned database was modified")
	}
}
