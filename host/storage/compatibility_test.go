package storage

import (
	"bytes"
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

func TestCompatibilityMismatchesStaySeparateFromStorageHealth(t *testing.T) {
	_, repository := newSchemaRepository(t)
	profile := domain.ProfileID("123e4567-e89b-42d3-a456-426614174000")
	storedVersions := CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion, RankingModelVersion: "rank-1"}
	stored := defaultsForVersions(profile, storedVersions)
	stored.ResultLimit = 17
	stored.UpdatedAt = 1_800_000_000
	if result := repository.SaveConfiguration(context.Background(), stored, storedVersions); result.Status != StatusReady || result.Compatibility != CompatibilityCompatible {
		t.Fatalf("save compatible configuration result=%+v", result)
	}

	protocolVersions := CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion + 1, RankingModelVersion: "rank-1"}
	config, result := repository.LoadConfiguration(context.Background(), string(profile), protocolVersions)
	if result.Status != StatusReady || result.ErrorClass != "" || result.Compatibility != CompatibilityProtocolMismatch || config != defaultsForVersions(profile, protocolVersions) {
		t.Fatalf("protocol mismatch result config=%+v outcome=%+v", config, result)
	}

	rankingVersions := CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion, RankingModelVersion: "rank-2"}
	config, result = repository.LoadConfiguration(context.Background(), string(profile), rankingVersions)
	if result.Status != StatusReady || result.ErrorClass != "" || result.Compatibility != CompatibilityRankingMismatch || config != defaultsForVersions(profile, rankingVersions) {
		t.Fatalf("ranking mismatch result config=%+v outcome=%+v", config, result)
	}
	if result := repository.SaveConfiguration(context.Background(), stored, rankingVersions); result.Status != StatusReady || result.ErrorClass != "" || result.Compatibility != CompatibilityRankingMismatch {
		t.Fatalf("incompatible write result=%+v", result)
	}

	bothVersions := CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion + 1, RankingModelVersion: "rank-2"}
	config, result = repository.LoadConfiguration(context.Background(), string(profile), bothVersions)
	if result.Status != StatusReady || result.Compatibility != CompatibilityProtocolAndRankingMismatch || config != defaultsForVersions(profile, bothVersions) {
		t.Fatalf("combined mismatch result config=%+v outcome=%+v", config, result)
	}

	config, result = repository.LoadConfiguration(context.Background(), string(profile), storedVersions)
	if result.Status != StatusReady || result.Compatibility != CompatibilityCompatible || config != stored {
		t.Fatalf("version mismatch rewrote stored preferences: config=%+v outcome=%+v", config, result)
	}
	if repository.CurrentResult().Status != StatusReady {
		t.Fatalf("compatibility mismatch degraded persistence: %+v", repository.CurrentResult())
	}
}

func TestUnknownFutureSchemaPreservesDatabaseFileAndRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future-schema.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	options := migrations.Options{ApplicationID: DatabaseApplicationID, OwnerID: DatabaseOwnerID, BusyTimeout: migrations.DefaultBusyTimeout}
	if result := migrations.Run(context.Background(), db, options, InitialMigrations()); result.State != migrations.StateReady {
		t.Fatalf("initialize database result=%+v", result)
	}
	const profileID = "123e4567-e89b-42d3-a456-426614174000"
	if _, err := db.Exec(`INSERT INTO configuration(profile_id, result_limit, theme) VALUES (?, 17, 'dark')`, profileID); err != nil {
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
	result := migrations.Run(context.Background(), futureDB, options, InitialMigrations())
	if result.State != migrations.StateUpgradeRequired || result.ErrorClass != migrations.ErrorFutureSchemaVersion || result.CurrentVersion != 42 {
		t.Fatalf("future schema result=%+v", result)
	}
	var resultLimit int
	var theme string
	if err := futureDB.QueryRow(`SELECT result_limit, theme FROM configuration WHERE profile_id = ?`, profileID).Scan(&resultLimit, &theme); err != nil {
		t.Fatal(err)
	}
	if resultLimit != 17 || theme != "dark" {
		t.Fatalf("future version row changed: result_limit=%d theme=%q", resultLimit, theme)
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
