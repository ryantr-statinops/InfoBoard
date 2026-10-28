package lifecycle

import (
	"context"
	"errors"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

type fakeStore struct {
	configs    map[string]storage.Configuration
	result     storage.Result
	saveResult storage.Result
	saved      map[string]storage.Configuration
	saveCalls  int
	resetCalls int
}

func readyResult() storage.Result {
	return storage.Result{Status: storage.StatusReady, Compatibility: storage.CompatibilityCompatible}
}

func (store *fakeStore) LoadConfiguration(_ context.Context, profileID string, versions storage.CompatibilityVersions) (storage.Configuration, storage.Result) {
	configuration, ok := store.configs[profileID]
	if !ok {
		configuration = storage.DefaultConfiguration(domain.ProfileID(profileID))
		configuration.ProtocolVersion = versions.ProtocolVersion
		configuration.RankingModelVersion = versions.RankingModelVersion
	}
	result := store.result
	if result.Status == "" {
		result = readyResult()
	}
	return configuration, result
}

func (store *fakeStore) SaveConfiguration(_ context.Context, configuration storage.Configuration, _ storage.CompatibilityVersions) storage.Result {
	store.saveCalls++
	if store.saved == nil {
		store.saved = make(map[string]storage.Configuration)
	}
	store.saved[string(configuration.ProfileID)] = configuration
	if store.saveResult.Status == "" {
		return readyResult()
	}
	return store.saveResult
}

func (store *fakeStore) ResetProfile(context.Context, string) storage.Result {
	store.resetCalls++
	return readyResult()
}

func testVersions() storage.CompatibilityVersions {
	return storage.CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion, RankingModelVersion: "rank-1"}
}

func newTestCoordinator(t *testing.T, store Store) *Coordinator {
	t.Helper()
	coordinator, err := NewCoordinator(store, testVersions())
	if err != nil {
		t.Fatalf("new coordinator: %v", err)
	}
	return coordinator
}

func TestLoadReturnsSafeDefaultsForMissingProfile(t *testing.T) {
	coordinator := newTestCoordinator(t, &fakeStore{configs: map[string]storage.Configuration{}})
	effective, err := coordinator.Load(context.Background(), profileA)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if effective.Persistence != PersistenceReady || !effective.Durable {
		t.Fatalf("persistence = %+v", effective)
	}
	if effective.Configuration.ResultLimit != 10 ||
		effective.Configuration.Density != DensityComfortable ||
		effective.Configuration.Theme != ThemeSystem ||
		!effective.Configuration.ContextLabels ||
		!effective.Configuration.RecencyEnabled {
		t.Fatalf("defaults = %+v", effective.Configuration)
	}
}

func TestLoadReturnsOnlyRequestedProfile(t *testing.T) {
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	configA := storage.DefaultConfiguration(profileA)
	configA.ResultLimit = 25
	configA.Density = string(DensityCompact)
	configA.Theme = string(ThemeDark)
	configA.ShowContextLabels = false
	configA.PersistActivationRecency = false
	configA.RankingModelVersion = "rank-1"
	store.configs[string(profileA)] = configA
	coordinator := newTestCoordinator(t, store)

	effectiveA, err := coordinator.Load(context.Background(), profileA)
	if err != nil {
		t.Fatalf("load A: %v", err)
	}
	if effectiveA.Configuration.ResultLimit != 25 || effectiveA.Configuration.Density != DensityCompact || effectiveA.Configuration.Theme != ThemeDark || effectiveA.Configuration.ContextLabels || effectiveA.Configuration.RecencyEnabled {
		t.Fatalf("profile A values = %+v", effectiveA.Configuration)
	}
	effectiveB, err := coordinator.Load(context.Background(), profileB)
	if err != nil {
		t.Fatalf("load B: %v", err)
	}
	if effectiveB.Configuration.ResultLimit != 10 || effectiveB.Configuration.Density != DensityComfortable || effectiveB.Configuration.Theme != ThemeSystem || !effectiveB.Configuration.ContextLabels || !effectiveB.Configuration.RecencyEnabled {
		t.Fatalf("profile B leaked values = %+v", effectiveB.Configuration)
	}
}

func TestLoadDegradesWithoutLosingDefaults(t *testing.T) {
	store := &fakeStore{
		configs: map[string]storage.Configuration{},
		result:  storage.Result{Status: storage.StatusDegraded, ErrorClass: storage.ErrorStorageUnavailable, Retryable: true},
	}
	coordinator := newTestCoordinator(t, store)
	effective, err := coordinator.Load(context.Background(), profileA)
	if err != nil {
		t.Fatalf("degraded load must be renderable: %v", err)
	}
	if effective.Persistence != PersistenceDegraded || effective.Durable {
		t.Fatalf("persistence = %+v", effective)
	}
	if effective.Configuration.ResultLimit != 10 || !effective.Configuration.RecencyEnabled {
		t.Fatalf("degraded defaults = %+v", effective.Configuration)
	}
	cached, ok := coordinator.Cached(profileA)
	if !ok || cached.Configuration != effective.Configuration || cached.Persistence != PersistenceDegraded {
		t.Fatalf("cached degraded defaults = %+v ok=%v", cached, ok)
	}
}

func TestLoadReplacesInvalidStoredValuesWithDefaults(t *testing.T) {
	config := storage.DefaultConfiguration(profileA)
	config.Density = "wide"
	config.RankingModelVersion = "rank-1"
	coordinator := newTestCoordinator(t, &fakeStore{configs: map[string]storage.Configuration{string(profileA): config}})
	effective, err := coordinator.Load(context.Background(), profileA)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if effective.Persistence != PersistenceDegraded || effective.Durable {
		t.Fatalf("invalid stored values must degrade: %+v", effective)
	}
	if effective.Configuration.Density != DensityComfortable {
		t.Fatalf("invalid density was not replaced: %+v", effective.Configuration)
	}
}

func TestLoadReplacesCompatibilityMismatchWithDefaults(t *testing.T) {
	config := storage.DefaultConfiguration(profileA)
	config.ResultLimit = 30
	config.RankingModelVersion = "rank-old"
	store := &fakeStore{
		configs: map[string]storage.Configuration{string(profileA): config},
		result:  storage.Result{Status: storage.StatusReady, Compatibility: storage.CompatibilityRankingMismatch},
	}
	coordinator := newTestCoordinator(t, store)
	effective, err := coordinator.Load(context.Background(), profileA)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if effective.Persistence != PersistenceDegraded || effective.Durable {
		t.Fatalf("compatibility mismatch must degrade: %+v", effective)
	}
	if effective.Configuration.ResultLimit != 10 {
		t.Fatalf("mismatched stored limit was used: %+v", effective.Configuration)
	}
}

func TestLoadRejectsMismatchedStoredProfile(t *testing.T) {
	config := storage.DefaultConfiguration(profileB)
	config.RankingModelVersion = "rank-1"
	coordinator := newTestCoordinator(t, &fakeStore{configs: map[string]storage.Configuration{string(profileA): config}})
	effective, err := coordinator.Load(context.Background(), profileA)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if effective.Persistence != PersistenceDegraded || effective.Durable {
		t.Fatalf("cross-profile stored row must degrade: %+v", effective)
	}
	if effective.Configuration.ProfileID != profileA {
		t.Fatalf("cross-profile value escaped: %+v", effective.Configuration)
	}
}

func TestLoadHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	coordinator := newTestCoordinator(t, &fakeStore{configs: map[string]storage.Configuration{}})
	if _, err := coordinator.Load(ctx, profileA); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestNewCoordinatorRejectsInvalidInputs(t *testing.T) {
	if _, err := NewCoordinator(nil, testVersions()); err == nil {
		t.Fatal("nil store must fail")
	}
	if _, err := NewCoordinator(&fakeStore{}, storage.CompatibilityVersions{}); err == nil {
		t.Fatal("invalid protocol version must fail")
	}
}

func TestUpdatePersistsStampedConfiguration(t *testing.T) {
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	configuration := validConfig(profileA)
	configuration.ProtocolVersion = 99
	configuration.RankingModelVersion = "caller-version"
	effective, err := coordinator.Update(context.Background(), profileA, configuration)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !effective.Durable || effective.Persistence != PersistenceReady || effective.Retryable {
		t.Fatalf("successful update status = %+v", effective)
	}
	if effective.Configuration.ProtocolVersion != 1 || effective.Configuration.RankingModelVersion != "rank-1" {
		t.Fatalf("host versions were not stamped: %+v", effective.Configuration)
	}
	saved, ok := store.saved[string(profileA)]
	if !ok {
		t.Fatal("update did not reach the store")
	}
	if saved.ProtocolVersion != 1 || saved.RankingModelVersion != "rank-1" || saved.ResultLimit != 12 || saved.Density != "compact" {
		t.Fatalf("saved configuration = %+v", saved)
	}
	cached, ok := coordinator.Cached(profileA)
	if !ok || cached.Configuration != effective.Configuration || !cached.Durable {
		t.Fatalf("cached successful update = %+v ok=%v", cached, ok)
	}
}

func TestUpdateDoesNotAffectAnotherProfile(t *testing.T) {
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	configuration := validConfig(profileA)
	configuration.ResultLimit = 25
	if _, err := coordinator.Update(context.Background(), profileA, configuration); err != nil {
		t.Fatalf("update A: %v", err)
	}
	effectiveB, err := coordinator.Load(context.Background(), profileB)
	if err != nil {
		t.Fatalf("load B: %v", err)
	}
	if effectiveB.Configuration.ResultLimit != 10 || effectiveB.Configuration.ProfileID != profileB {
		t.Fatalf("profile B received profile A values: %+v", effectiveB.Configuration)
	}
}

func TestUpdateRetainsLastEffectiveValueOnWriteFailure(t *testing.T) {
	configA := storage.DefaultConfiguration(profileA)
	configA.ResultLimit = 25
	configA.Density = string(DensityCompact)
	configA.RankingModelVersion = "rank-1"
	store := &fakeStore{configs: map[string]storage.Configuration{string(profileA): configA}}
	coordinator := newTestCoordinator(t, store)
	if _, err := coordinator.Load(context.Background(), profileA); err != nil {
		t.Fatalf("seed load: %v", err)
	}
	store.saveResult = storage.Result{Status: storage.StatusDegraded, ErrorClass: storage.ErrorStorageFailure, Retryable: true}
	configuration := validConfig(profileA)
	configuration.ResultLimit = 30
	effective, err := coordinator.Update(context.Background(), profileA, configuration)
	if err != nil {
		t.Fatalf("degraded update must be renderable: %v", err)
	}
	if effective.Durable || effective.Persistence != PersistenceDegraded || !effective.Retryable || effective.ErrorClass != string(storage.ErrorStorageFailure) {
		t.Fatalf("degraded update status = %+v", effective)
	}
	if effective.Configuration.ResultLimit != 25 || effective.Configuration.Density != DensityCompact {
		t.Fatalf("write failure lost last effective value: %+v", effective.Configuration)
	}
	cached, ok := coordinator.Cached(profileA)
	if !ok || cached.Configuration.ResultLimit != 25 || cached.Durable || !cached.Retryable {
		t.Fatalf("cached failed update = %+v ok=%v", cached, ok)
	}
}

func TestUpdateWithoutPriorLoadReturnsDefaultsOnFailure(t *testing.T) {
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	store.saveResult = storage.Result{Status: storage.StatusDegraded, ErrorClass: storage.ErrorStorageUnavailable, Retryable: true}
	coordinator := newTestCoordinator(t, store)
	configuration := validConfig(profileA)
	effective, err := coordinator.Update(context.Background(), profileA, configuration)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if effective.Durable || effective.Configuration.ResultLimit != 10 || effective.Configuration.Density != DensityComfortable {
		t.Fatalf("failure without cache must use defaults: %+v", effective)
	}
}

func TestUpdateRejectsInvalidInputsBeforeWriting(t *testing.T) {
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	invalid := validConfig(profileA)
	invalid.Density = "wide"
	if _, err := coordinator.Update(context.Background(), profileA, invalid); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("invalid config must fail: %v", err)
	}
	mismatched := validConfig(profileB)
	if _, err := coordinator.Update(context.Background(), profileA, mismatched); !errors.Is(err, ErrProfileMismatch) {
		t.Fatalf("mismatched profile must fail: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := coordinator.Update(ctx, profileA, validConfig(profileA)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled update must fail: %v", err)
	}
	if store.saveCalls != 0 {
		t.Fatalf("invalid updates reached the store %d times", store.saveCalls)
	}
}
