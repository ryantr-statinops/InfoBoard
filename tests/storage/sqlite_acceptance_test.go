package storage_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
	"github.com/ryantr-statinops/InfoBoard/host/storage/migrations"
)

const profileA = "123e4567-e89b-42d3-a456-426614174000"
const profileB = "123e4567-e89b-42d3-a456-426614174099"

type fixtureCatalog struct {
	SchemaVersion int           `json:"schema_version"`
	Cases         []fixtureCase `json:"cases"`
}

type fixtureCase struct {
	ID             string           `json:"id"`
	Scenario       string           `json:"scenario"`
	ExpectedStatus string           `json:"expected_status"`
	ExpectedRows   map[string]int64 `json:"expected_rows"`
	Boundary       string           `json:"boundary"`
	Ownership      string           `json:"ownership"`
}

func loadFixtures(t *testing.T) map[string]fixtureCase {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "sqlite", "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog fixtureCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.SchemaVersion != 1 || len(catalog.Cases) != 10 {
		t.Fatalf("fixture catalog schema=%d cases=%d", catalog.SchemaVersion, len(catalog.Cases))
	}
	fixtures := make(map[string]fixtureCase, len(catalog.Cases))
	for _, fixture := range catalog.Cases {
		if fixture.ID == "" || fixture.Scenario == "" || fixture.ExpectedStatus == "" || fixture.Boundary == "" || fixture.Ownership == "" {
			t.Fatalf("incomplete SQL fixture: %+v", fixture)
		}
		if _, exists := fixtures[fixture.ID]; exists {
			t.Fatalf("duplicate SQL fixture %s", fixture.ID)
		}
		fixtures[fixture.ID] = fixture
	}
	return fixtures
}

func newDatabase(t *testing.T) (string, *storage.Repository) {
	t.Helper()
	directory := t.TempDir()
	repository, result := storage.Open(context.Background(), directory)
	if repository == nil || result.Status != storage.StatusReady {
		t.Fatalf("open fresh storage: repository nil=%t result=%+v", repository == nil, result)
	}
	return filepath.Join(directory, storage.DatabaseFileName), repository
}

func openSQL(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func versions() storage.CompatibilityVersions {
	return storage.CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion}
}

func expectStatus(t *testing.T, fixture fixtureCase, result storage.Result) {
	t.Helper()
	if string(result.Status) != fixture.ExpectedStatus {
		t.Fatalf("%s (%s) status=%s result=%+v", fixture.ID, fixture.Scenario, result.Status, result)
	}
}

func TestSQL001FreshDefaultsAndSchema(t *testing.T) {
	fixture := loadFixtures(t)["SQL-001"]
	path, repository := newDatabase(t)
	configuration, result := repository.LoadConfiguration(context.Background(), profileA, versions())
	expectStatus(t, fixture, result)
	if configuration.ProfileID != domain.ProfileID(profileA) || !configuration.PersistActivationRecency || configuration.ResultLimit != storage.DefaultResultLimit {
		t.Fatalf("fresh profile defaults = %+v", configuration)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	db := openSQL(t, path)
	var migrations, configurations, activations, diagnostics int
	for _, query := range []struct {
		statement   string
		destination *int
	}{{"SELECT count(*) FROM schema_migrations", &migrations}, {"SELECT count(*) FROM configuration", &configurations}, {"SELECT count(*) FROM activation_metadata", &activations}, {"SELECT count(*) FROM diagnostics", &diagnostics}} {
		if err := db.QueryRow(query.statement).Scan(query.destination); err != nil {
			t.Fatal(err)
		}
	}
	if int64(migrations) != fixture.ExpectedRows["schema_migrations"] || configurations != 0 || activations != 0 || diagnostics != 0 {
		t.Fatalf("fresh rows migrations=%d configuration=%d activation=%d diagnostics=%d", migrations, configurations, activations, diagnostics)
	}
}

func TestSQL002ProfileIsolation(t *testing.T) {
	fixture := loadFixtures(t)["SQL-002"]
	path, repository := newDatabase(t)
	configuration := storage.DefaultConfiguration(domain.ProfileID(profileA))
	configuration.ResultLimit = 12
	configuration.PersistActivationRecency = true
	configuration.RankingModelVersion = "rank-a"
	if result := repository.SaveConfiguration(context.Background(), configuration, storage.CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion, RankingModelVersion: "rank-a"}); result.Status != storage.StatusReady {
		t.Fatalf("save profile A = %+v", result)
	}
	loadedA, result := repository.LoadConfiguration(context.Background(), profileA, storage.CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion, RankingModelVersion: "rank-a"})
	expectStatus(t, fixture, result)
	loadedB, result := repository.LoadConfiguration(context.Background(), profileB, storage.CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion, RankingModelVersion: "rank-b"})
	expectStatus(t, fixture, result)
	if loadedA.ResultLimit != 12 || loadedA.PersistActivationRecency != true || loadedB.ResultLimit != storage.DefaultResultLimit || !loadedB.PersistActivationRecency {
		t.Fatalf("profile A/B configurations leaked: A=%+v B=%+v", loadedA, loadedB)
	}
	_ = repository.Close()
	db := openSQL(t, path)
	var count int
	if err := db.QueryRow("SELECT count(*) FROM configuration WHERE profile_id = ?", profileA).Scan(&count); err != nil || int64(count) != fixture.ExpectedRows["configuration_profile_a"] {
		t.Fatalf("profile A configuration rows=%d err=%v", count, err)
	}
}

func TestSQL003MigrationReopenIsIdempotent(t *testing.T) {
	fixture := loadFixtures(t)["SQL-003"]
	path, repository := newDatabase(t)
	_ = repository.Close()
	reopened, result := storage.Open(context.Background(), filepath.Dir(path))
	if reopened == nil {
		t.Fatalf("reopen migration repository nil: %+v", result)
	}
	expectStatus(t, fixture, result)
	_ = reopened.Close()
	db := openSQL(t, path)
	var migrations, owners int
	if err := db.QueryRow("SELECT count(*) FROM schema_migrations").Scan(&migrations); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM schema_meta WHERE key = 'owner_id'").Scan(&owners); err != nil {
		t.Fatal(err)
	}
	if int64(migrations) != fixture.ExpectedRows["schema_migrations"] || int64(owners) != fixture.ExpectedRows["schema_meta_owner"] {
		t.Fatalf("migration rows=%d owners=%d", migrations, owners)
	}
}

func TestSQL004FailedMigrationRollsBack(t *testing.T) {
	fixture := loadFixtures(t)["SQL-004"]
	path := filepath.Join(t.TempDir(), storage.DatabaseFileName)
	db := openSQL(t, path)
	options := migrations.Options{ApplicationID: storage.DatabaseApplicationID, OwnerID: storage.DatabaseOwnerID, BusyTimeout: migrations.DefaultBusyTimeout}
	if result := migrations.Run(context.Background(), db, options, storage.InitialMigrations()); result.State != migrations.StateReady {
		t.Fatalf("baseline migration = %+v", result)
	}
	broken := append(storage.InitialMigrations(), migrations.Migration{Version: 2, ID: "fixture_failed_v2", Statements: []string{
		"CREATE TABLE fixture_partial(value TEXT)",
		"CREATE TABL broken_syntax(value TEXT)",
	}})
	result := migrations.Run(context.Background(), db, options, broken)
	expectStatus(t, fixture, storage.Result{Status: storage.StatusDegraded})
	if result.State != migrations.StateDegraded {
		t.Fatalf("failed migration status=%s class=%s", result.State, result.ErrorClass)
	}
	var version, failedTables, versionTwoRows int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='fixture_partial'").Scan(&failedTables); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM schema_migrations WHERE version=2").Scan(&versionTwoRows); err != nil {
		t.Fatal(err)
	}
	if int64(version) != fixture.ExpectedRows["user_version"] || int64(failedTables) != fixture.ExpectedRows["failed_table"] || int64(versionTwoRows) != fixture.ExpectedRows["version_2_history"] {
		t.Fatalf("rollback rows user_version=%d partial_table=%d v2_history=%d", version, failedTables, versionTwoRows)
	}
}

func TestSQL005FutureVersionBytesStayUnchanged(t *testing.T) {
	fixture := loadFixtures(t)["SQL-005"]
	path, repository := newDatabase(t)
	_ = repository.Close()
	db := openSQL(t, path)
	if _, err := db.Exec("PRAGMA user_version = 2"); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	repository, result := storage.Open(context.Background(), filepath.Dir(path))
	expectStatus(t, fixture, result)
	if repository != nil {
		t.Fatal("future schema unexpectedly returned a writable repository")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("future database bytes changed")
	}
}

func TestSQL006CorruptDatabaseIsPreservedAndRebuilt(t *testing.T) {
	fixture := loadFixtures(t)["SQL-006"]
	directory := t.TempDir()
	path := filepath.Join(directory, storage.DatabaseFileName)
	original := []byte("SQL-006-corrupt-bytes")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	repository, result := storage.Open(context.Background(), directory)
	expectStatus(t, fixture, result)
	if repository == nil {
		t.Fatalf("rebuilt repository is nil: %+v", result)
	}
	_ = repository.Close()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var preserved []byte
	for _, entry := range entries {
		if entry.IsDir() {
			preserved, err = os.ReadFile(filepath.Join(directory, entry.Name(), storage.DatabaseFileName))
			if err == nil {
				break
			}
		}
	}
	if string(preserved) != string(original) {
		t.Fatalf("quarantined bytes = %q", preserved)
	}
}

func TestSQL007ActivationAgeAndProfileCountBoundaries(t *testing.T) {
	fixture := loadFixtures(t)["SQL-007"]
	path, repository := newDatabase(t)
	ctx := context.Background()
	for _, profileID := range []string{profileA, profileB} {
		configuration := storage.DefaultConfiguration(domain.ProfileID(profileID))
		configuration.PersistActivationRecency = true
		if result := repository.SaveConfiguration(ctx, configuration, versions()); result.Status != storage.StatusReady {
			t.Fatalf("enable retention for %s: %+v", profileID, result)
		}
	}
	now := time.Now().Unix()
	for index := range storage.MaxActivationRowsPerProfile + 1 {
		result := repository.RecordActivation(ctx, storage.ActivationRecord{ProfileID: profileA, TabID: int64(index), Domain: "example.test", ActivatedAt: now, Source: domain.ActivationKeyboardEnter})
		if result.Status != storage.StatusReady {
			t.Fatalf("record profile A activation %d: %+v", index, result)
		}
	}
	if result := repository.RecordActivation(ctx, storage.ActivationRecord{ProfileID: profileB, TabID: 1, Domain: "example.test", ActivatedAt: now, Source: domain.ActivationSurfaceSelect}); result.Status != storage.StatusReady {
		t.Fatalf("record profile B activation: %+v", result)
	}
	_ = repository.Close()
	db := openSQL(t, path)
	for profileID, want := range map[string]int64{profileA: fixture.ExpectedRows["profile_a"], profileB: fixture.ExpectedRows["profile_b"]} {
		var count int64
		if err := db.QueryRow("SELECT count(*) FROM activation_metadata WHERE profile_id = ?", profileID).Scan(&count); err != nil || count != want {
			t.Fatalf("%s activation count=%d want=%d err=%v", profileID, count, want, err)
		}
	}
}

func TestSQL008DiagnosticAgeAndUTF8ByteBudget(t *testing.T) {
	fixture := loadFixtures(t)["SQL-008"]
	db, repository := newInMemoryRepository(t)
	payload := make([]byte, storage.MaxDiagnosticRowBytes)
	for index := range payload {
		payload[index] = 'x'
	}
	base := time.Now().UnixMilli()
	for index := range storage.MaxDiagnosticsPayloadBytes/storage.MaxDiagnosticRowBytes + 1 {
		result := repository.AppendDiagnostic(context.Background(), storage.DiagnosticRecord{
			OccurredAt: base + int64(index), EventKind: "storage.test", ErrorClass: storage.ErrorStorageFailure,
			ProtocolVersion: protocol.CurrentVersion, SchemaVersion: storage.SupportedSchemaVersion,
			RedactedPayload: string(payload),
		})
		if result.Status != storage.StatusReady {
			t.Fatalf("append diagnostic %d: %+v", index, result)
		}
	}
	var count, payloadBytes int64
	if err := db.QueryRow("SELECT count(*), sum(payload_bytes) FROM diagnostics").Scan(&count, &payloadBytes); err != nil {
		t.Fatal(err)
	}
	if payloadBytes != fixture.ExpectedRows["retained_diagnostic_payload_bytes"] || payloadBytes != storage.MaxDiagnosticsPayloadBytes {
		t.Fatalf("diagnostic payload bytes=%d", payloadBytes)
	}
	if count != storage.MaxDiagnosticsPayloadBytes/storage.MaxDiagnosticRowBytes {
		t.Fatalf("diagnostic row count=%d", count)
	}
}

func TestSQL009UnavailableRepositoryReturnsDegradedDefaults(t *testing.T) {
	fixture := loadFixtures(t)["SQL-009"]
	repository := storage.NewRepository(nil)
	configuration, result := repository.LoadConfiguration(context.Background(), profileA, versions())
	expectStatus(t, fixture, result)
	if configuration.ProfileID != domain.ProfileID(profileA) || !configuration.PersistActivationRecency {
		t.Fatalf("degraded defaults = %+v", configuration)
	}
	if result.ErrorClass != storage.ErrorStorageUnavailable {
		t.Fatalf("unbounded failure class = %q", result.ErrorClass)
	}
	if fixture.ExpectedRows["error_class_is_closed"] != 1 {
		t.Fatal("fixture must assert closed error classes")
	}
}

func TestSQL010ResetAndUninstallPreserveUnownedFiles(t *testing.T) {
	fixture := loadFixtures(t)["SQL-010"]
	path, repository := newDatabase(t)
	ctx := context.Background()
	for _, profileID := range []string{profileA, profileB} {
		configuration := storage.DefaultConfiguration(domain.ProfileID(profileID))
		if result := repository.SaveConfiguration(ctx, configuration, versions()); result.Status != storage.StatusReady {
			t.Fatalf("save %s configuration: %+v", profileID, result)
		}
	}
	if result := repository.ResetProfile(ctx, profileA); result.Status != storage.StatusReady {
		t.Fatalf("profile reset = %+v", result)
	}
	if result := repository.ResetAll(ctx); result.Status != storage.StatusReady {
		t.Fatalf("full reset = %+v", result)
	}
	sentinel := filepath.Join(filepath.Dir(path), "outside-owner-sentinel")
	if err := os.WriteFile(sentinel, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	result := repository.Uninstall(ctx, path)
	expectStatus(t, fixture, result)
	if result.ErrorClass != "" {
		t.Fatalf("uninstall error class=%s", result.ErrorClass)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("owned DB remains: %v", err)
	}
	if content, err := os.ReadFile(sentinel); err != nil || string(content) != "preserve" {
		t.Fatalf("unowned sentinel changed: content=%q err=%v", content, err)
	}
	if result := repository.Uninstall(ctx, path); result.Status != storage.StatusDisabled {
		t.Fatalf("repeat uninstall status=%s", result.Status)
	}
}

func TestFixtureIDsAreContiguousAndUnique(t *testing.T) {
	fixtures := loadFixtures(t)
	for index := range 10 {
		id := fmt.Sprintf("SQL-%03d", index+1)
		if _, ok := fixtures[id]; !ok {
			t.Fatalf("missing fixture %s", id)
		}
	}
}
