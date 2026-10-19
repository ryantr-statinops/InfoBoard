package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/storage/migrations"
	_ "modernc.org/sqlite"
)

func openMigratedRepository(t *testing.T, path string) (*sql.DB, *Repository) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	result := migrations.Run(context.Background(), db, migrations.Options{
		ApplicationID: DatabaseApplicationID,
		OwnerID:       DatabaseOwnerID,
		BusyTimeout:   migrations.DefaultBusyTimeout,
	}, InitialMigrations())
	if result.State != migrations.StateReady {
		_ = db.Close()
		t.Fatalf("migration result = %+v", result)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, NewRepository(db)
}

func TestResetProfileDeletesOnlySelectedProfileRows(t *testing.T) {
	db, repository := newSchemaRepository(t)
	ctx := context.Background()
	versions := CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion, RankingModelVersion: "rank-current"}
	for _, profileID := range []string{retentionProfileA, retentionProfileB} {
		configuration := DefaultConfiguration(domain.ProfileID(profileID))
		configuration.PersistActivationRecency = true
		configuration.RankingModelVersion = versions.RankingModelVersion
		if result := repository.SaveConfiguration(ctx, configuration, versions); result.Status != StatusReady {
			t.Fatalf("save %s configuration: %+v", profileID, result)
		}
		if result := repository.RecordActivation(ctx, ActivationRecord{
			ProfileID: profileID, TabID: 1, Domain: "example.test", ActivatedAt: 1_800_000_000, Source: domain.ActivationKeyboardEnter,
		}); result.Status != StatusReady {
			t.Fatalf("record %s activation: %+v", profileID, result)
		}
		if _, err := db.Exec(`INSERT INTO diagnostics(occurred_at, profile_id, event_kind, error_class, retryable, duration_ms, projection_revision, protocol_version, ranking_model_version, schema_version, payload_bytes, redacted_payload) VALUES (1, ?, 'storage.test', 'storage_failure', 0, 0, 0, ?, '', 1, 0, '')`, profileID, protocol.CurrentVersion); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO diagnostics(occurred_at, event_kind, error_class, retryable, duration_ms, projection_revision, protocol_version, ranking_model_version, schema_version, payload_bytes, redacted_payload) VALUES (1, 'storage.global', 'storage_failure', 0, 0, 0, ?, '', 1, 0, '')`, protocol.CurrentVersion); err != nil {
		t.Fatal(err)
	}

	result := repository.ResetProfile(ctx, retentionProfileA)
	if result.Status != StatusReady || result.ErrorClass != "" {
		t.Fatalf("profile reset result = %+v", result)
	}
	for _, table := range []string{"configuration", "activation_metadata", "diagnostics"} {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM `+table+` WHERE profile_id = ?`, retentionProfileA).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("profile A rows in %s after reset = %d", table, count)
		}
		if err := db.QueryRow(`SELECT count(*) FROM `+table+` WHERE profile_id = ?`, retentionProfileB).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("profile B rows in %s after profile A reset = %d", table, count)
		}
	}
	var globalDiagnostics int
	if err := db.QueryRow(`SELECT count(*) FROM diagnostics WHERE profile_id IS NULL`).Scan(&globalDiagnostics); err != nil {
		t.Fatal(err)
	}
	if globalDiagnostics != 1 {
		t.Fatalf("global diagnostics after profile reset = %d", globalDiagnostics)
	}
}

func TestResetAllIsIdempotentAndKeepsSchemaOwnership(t *testing.T) {
	db, repository := openMigratedRepository(t, filepath.Join(t.TempDir(), DatabaseFileName))
	for _, statement := range []string{
		`INSERT INTO configuration(profile_id) VALUES ('123e4567-e89b-42d3-a456-426614174000')`,
		`INSERT INTO installation_state(installation_key) VALUES ('infoboard')`,
		`INSERT INTO diagnostics(occurred_at, event_kind, error_class, retryable, duration_ms, schema_version, payload_bytes, redacted_payload) VALUES (1, 'storage.test', 'storage_failure', 0, 0, 1, 0, '')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	for attempt := range 2 {
		result := repository.ResetAll(context.Background())
		if result.Status != StatusReady || result.ErrorClass != "" {
			t.Fatalf("full reset attempt %d = %+v", attempt, result)
		}
	}
	for _, table := range []string{"configuration", "installation_state", "activation_metadata", "diagnostics"} {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("InfoBoard rows in %s after reset = %d", table, count)
		}
	}
	var ownerEntries, migrationEntries int
	if err := db.QueryRow(`SELECT count(*) FROM schema_meta WHERE key = 'owner_id'`).Scan(&ownerEntries); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&migrationEntries); err != nil {
		t.Fatal(err)
	}
	if ownerEntries != 1 || migrationEntries != 1 {
		t.Fatalf("reset removed reopen metadata: owner=%d migrations=%d", ownerEntries, migrationEntries)
	}
}

func TestUninstallRemovesOnlyOwnedDatabaseAndQuarantineArtifacts(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	databasePath := filepath.Join(directory, DatabaseFileName)
	db, repository := openMigratedRepository(t, databasePath)
	if _, err := db.Exec(`INSERT INTO installation_state(installation_key) VALUES ('infoboard')`); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(databasePath+"-wal", []byte("owned sidecar"), 0o600); err != nil {
		t.Fatal(err)
	}
	ownedQuarantine := filepath.Join(directory, quarantineDirectoryPrefix+"test")
	if err := os.Mkdir(ownedQuarantine, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ownedQuarantine, "owner.txt"), []byte(DatabaseOwnerID+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ownedQuarantine, DatabaseFileName), []byte("quarantined"), 0o600); err != nil {
		t.Fatal(err)
	}
	unownedQuarantine := filepath.Join(directory, quarantineDirectoryPrefix+"unowned")
	if err := os.Mkdir(unownedQuarantine, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unownedQuarantine, "sentinel"), []byte("leave me"), 0o600); err != nil {
		t.Fatal(err)
	}
	unrelated := filepath.Join(directory, "unrelated-sentinel.txt")
	if err := os.WriteFile(unrelated, []byte("leave me"), 0o600); err != nil {
		t.Fatal(err)
	}

	result := repository.Uninstall(ctx, databasePath)
	if result.Status != StatusDisabled || result.ErrorClass != "" {
		t.Fatalf("uninstall result = %+v", result)
	}
	for _, path := range []string{databasePath, databasePath + "-wal", ownedQuarantine} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("owned uninstall artifact remains at %q: %v", path, err)
		}
	}
	for _, path := range []string{unrelated, filepath.Join(unownedQuarantine, "sentinel")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("unrelated artifact was removed: %q: %v", path, err)
		}
	}

	if result := repository.Uninstall(ctx, databasePath); result.Status != StatusDisabled || result.ErrorClass != "" {
		t.Fatalf("repeated uninstall result = %+v", result)
	}
}

func TestUninstallRefusesForeignDatabaseAtReservedFilename(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), DatabaseFileName)
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE unrelated_sentinel(value TEXT)"); err != nil {
		t.Fatal(err)
	}
	repository := NewRepository(db)
	result := repository.Uninstall(context.Background(), databasePath)
	if result.Status != StatusQuarantined || result.ErrorClass != ErrorUnsafeDatabasePath {
		t.Fatalf("foreign database uninstall result = %+v", result)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(databasePath); err != nil {
		t.Fatalf("foreign database was removed: %v", err)
	}
}
