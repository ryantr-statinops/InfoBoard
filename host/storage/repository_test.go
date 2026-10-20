package storage

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	_ "modernc.org/sqlite"
	"time"
)

func newSchemaRepository(t *testing.T) (*sql.DB, *Repository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	for _, statement := range SchemaStatements() {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			t.Fatalf("apply schema: %v", err)
		}
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, NewRepository(db)
}

func TestRepositoryConfigurationIsProfileScopedAndDefaultsWhenMissing(t *testing.T) {
	_, repository := newSchemaRepository(t)
	profileA := domain.ProfileID("123e4567-e89b-42d3-a456-426614174000")
	profileB := domain.ProfileID("123e4567-e89b-42d3-a456-426614174099")
	versions := CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion, RankingModelVersion: "rank-current"}

	defaults, result := repository.LoadConfiguration(context.Background(), string(profileA), versions)
	if result.Status != StatusReady || result.ErrorClass != "" || result.Compatibility != CompatibilityCompatible || defaults != defaultsForVersions(profileA, versions) {
		t.Fatalf("missing profile configuration=%+v result=%+v", defaults, result)
	}

	configuration := defaults
	configuration.ResultLimit = 12
	configuration.Density = "compact"
	configuration.Theme = "dark"
	configuration.ShowContextLabels = false
	configuration.PersistActivationRecency = true
	configuration.RankingModelVersion = "rank-current"
	configuration.UpdatedAt = 1_800_000_000
	if result := repository.SaveConfiguration(context.Background(), configuration, versions); result.Status != StatusReady || result.ErrorClass != "" {
		t.Fatalf("save configuration result=%+v", result)
	}
	loaded, result := repository.LoadConfiguration(context.Background(), string(profileA), versions)
	if result.Status != StatusReady || loaded != configuration {
		t.Fatalf("loaded profile A=%+v result=%+v want=%+v", loaded, result, configuration)
	}
	profileBDefaults, result := repository.LoadConfiguration(context.Background(), string(profileB), versions)
	if result.Status != StatusReady || profileBDefaults != defaultsForVersions(profileB, versions) {
		t.Fatalf("profile B read leaked profile A configuration: config=%+v result=%+v", profileBDefaults, result)
	}
}

func TestRepositoryRejectsInvalidProfileAndConfigurationWithoutMutation(t *testing.T) {
	_, repository := newSchemaRepository(t)
	profile := domain.ProfileID("123e4567-e89b-42d3-a456-426614174000")
	versions := CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion}
	defaults, result := repository.LoadConfiguration(context.Background(), string(profile), versions)
	if result.Status != StatusReady {
		t.Fatalf("initial result=%+v", result)
	}

	if _, result := repository.LoadConfiguration(context.Background(), "not-a-profile", versions); result.ErrorClass != ErrorProfileMismatch || result.Status != StatusReady {
		t.Fatalf("malformed profile result=%+v", result)
	}
	invalid := defaults
	invalid.ResultLimit = protocol.MaxResults + 1
	if result := repository.SaveConfiguration(context.Background(), invalid, versions); result.ErrorClass != ErrorInvalidConfiguration || result.Status != StatusReady || result.Retryable {
		t.Fatalf("invalid configuration result=%+v", result)
	}
	stillDefault, result := repository.LoadConfiguration(context.Background(), string(profile), versions)
	if result.Status != StatusReady || stillDefault != defaults {
		t.Fatalf("invalid write mutated configuration: value=%+v result=%+v", stillDefault, result)
	}
}

func TestRepositoryStorageFailureReturnsDefaultsAndDegradesOnlyPersistence(t *testing.T) {
	db, repository := newSchemaRepository(t)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	profile := domain.ProfileID("123e4567-e89b-42d3-a456-426614174000")
	versions := CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion}
	configuration, result := repository.LoadConfiguration(context.Background(), string(profile), versions)
	if configuration != defaultsForVersions(profile, versions) || result.Status != StatusDegraded || !result.Retryable || result.ErrorClass != ErrorStorageFailure {
		t.Fatalf("storage failure configuration=%+v result=%+v", configuration, result)
	}
	writeResult := repository.SaveConfiguration(context.Background(), configuration, versions)
	if writeResult.Status != StatusDegraded || !writeResult.Retryable || writeResult.ErrorClass != ErrorStorageFailure {
		t.Fatalf("failed write result=%+v", writeResult)
	}
	if current := repository.CurrentResult(); current.Status != StatusDegraded || !current.Retryable || current.ErrorClass != ErrorStorageFailure {
		t.Fatalf("repository did not retain bounded degraded state: %+v", current)
	}
}

func TestRepositoryDeadlineReturnsOnlyBoundedTimeoutClass(t *testing.T) {
	_, repository := newSchemaRepository(t)
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer cancel()
	profile := domain.ProfileID("123e4567-e89b-42d3-a456-426614174000")
	versions := CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion}
	configuration, result := repository.LoadConfiguration(ctx, string(profile), versions)
	if configuration != defaultsForVersions(profile, versions) || result.Status != StatusDegraded || result.ErrorClass != ErrorStorageTimeout || !result.Retryable {
		t.Fatalf("deadline result configuration=%+v result=%+v", configuration, result)
	}
}
