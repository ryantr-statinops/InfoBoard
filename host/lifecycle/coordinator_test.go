package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

type fakeStore struct {
	configs         map[string]storage.Configuration
	result          storage.Result
	saveResult      storage.Result
	resetResult     storage.Result
	uninstallResult storage.Result
	saved           map[string]storage.Configuration
	saveCalls       int
	resetCalls      int
	uninstallCalls  int
	databasePath    string
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

func (store *fakeStore) ResetProfile(_ context.Context, profileID string) storage.Result {
	store.resetCalls++
	if store.resetResult.Status != "" {
		return store.resetResult
	}
	delete(store.configs, profileID)
	return readyResult()
}

func (store *fakeStore) Uninstall(_ context.Context, databasePath string) storage.Result {
	store.uninstallCalls++
	store.databasePath = databasePath
	if store.uninstallResult.Status != "" {
		return store.uninstallResult
	}
	store.configs = make(map[string]storage.Configuration)
	return storage.Result{Status: storage.StatusDisabled}
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
	if effective.Durable || effective.Persistence != PersistenceDegraded || !effective.Retryable || effective.ErrorClass != ErrorStorageFailure {
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

type fakeResetRuntime struct {
	outcome ProjectionReset
	err     error
	profile domain.ProfileID
	calls   int
}

func (runtime *fakeResetRuntime) ResetProjection(ctx context.Context, profileID domain.ProfileID) (ProjectionReset, error) {
	if err := ctx.Err(); err != nil {
		return ProjectionReset{}, err
	}
	runtime.calls++
	runtime.profile = profileID
	return runtime.outcome, runtime.err
}

func TestResetClearsOnlyActiveProfileAndPublishesDefaults(t *testing.T) {
	configA := storage.DefaultConfiguration(profileA)
	configA.ResultLimit = 25
	configA.RankingModelVersion = "rank-1"
	configB := storage.DefaultConfiguration(profileB)
	configB.ResultLimit = 30
	configB.RankingModelVersion = "rank-1"
	store := &fakeStore{configs: map[string]storage.Configuration{
		string(profileA): configA,
		string(profileB): configB,
	}}
	coordinator := newTestCoordinator(t, store)
	if _, err := coordinator.Load(context.Background(), profileA); err != nil {
		t.Fatalf("seed load: %v", err)
	}
	resetRuntime := &fakeResetRuntime{outcome: ProjectionReset{Revision: 42, IndexCleared: true, SnapshotRequested: true}}
	coordinator.SetResetRuntime(resetRuntime)
	result, err := coordinator.Reset(context.Background(), profileA)
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if result.Status != StatusComplete || result.Retryable || !result.OwnedStateVerified || result.ProjectionRevision != 42 {
		t.Fatalf("reset result = %+v", result)
	}
	if len(result.CompletedSteps) != 3 || result.CompletedSteps[0] != StepStorageReset || result.CompletedSteps[1] != StepIndexCleared || result.CompletedSteps[2] != StepSnapshotRequested {
		t.Fatalf("completed steps = %v", result.CompletedSteps)
	}
	if result.Configuration.ResultLimit != 10 || result.Configuration.Density != DensityComfortable {
		t.Fatalf("reset did not publish defaults: %+v", result.Configuration)
	}
	if _, ok := store.configs[string(profileA)]; ok {
		t.Fatal("profile A rows remain after reset")
	}
	if _, ok := store.configs[string(profileB)]; !ok {
		t.Fatal("profile B rows were deleted")
	}
	if resetRuntime.profile != profileA || resetRuntime.calls != 1 {
		t.Fatalf("runtime handoff = profile:%s calls:%d", resetRuntime.profile, resetRuntime.calls)
	}
	cached, ok := coordinator.Cached(profileA)
	if !ok || !cached.Durable || cached.Configuration.ResultLimit != 10 {
		t.Fatalf("cached reset defaults = %+v ok=%v", cached, ok)
	}
}

func TestResetFailureRetainsPreviousValues(t *testing.T) {
	configA := storage.DefaultConfiguration(profileA)
	configA.ResultLimit = 25
	configA.RankingModelVersion = "rank-1"
	store := &fakeStore{
		configs:     map[string]storage.Configuration{string(profileA): configA},
		resetResult: storage.Result{Status: storage.StatusDegraded, ErrorClass: storage.ErrorStorageFailure, Retryable: true},
	}
	coordinator := newTestCoordinator(t, store)
	if _, err := coordinator.Load(context.Background(), profileA); err != nil {
		t.Fatalf("seed load: %v", err)
	}
	resetRuntime := &fakeResetRuntime{outcome: ProjectionReset{Revision: 7, IndexCleared: true, SnapshotRequested: true}}
	coordinator.SetResetRuntime(resetRuntime)
	result, err := coordinator.Reset(context.Background(), profileA)
	if err != nil {
		t.Fatalf("reset failure must be renderable: %v", err)
	}
	if result.Status != StatusRetryable || !result.Retryable || result.Durable || result.ErrorClass != ErrorStorageFailure {
		t.Fatalf("reset failure status = %+v", result)
	}
	if result.Configuration.ResultLimit != 25 {
		t.Fatalf("reset failure lost previous values: %+v", result.Configuration)
	}
	if resetRuntime.calls != 0 {
		t.Fatalf("runtime ran after storage failure %d times", resetRuntime.calls)
	}
}

func TestResetRuntimeFailureIsIncomplete(t *testing.T) {
	configA := storage.DefaultConfiguration(profileA)
	configA.ResultLimit = 25
	configA.RankingModelVersion = "rank-1"
	store := &fakeStore{configs: map[string]storage.Configuration{string(profileA): configA}}
	coordinator := newTestCoordinator(t, store)
	if _, err := coordinator.Load(context.Background(), profileA); err != nil {
		t.Fatalf("seed load: %v", err)
	}
	resetRuntime := &fakeResetRuntime{err: errors.New("index reset failed")}
	coordinator.SetResetRuntime(resetRuntime)
	result, err := coordinator.Reset(context.Background(), profileA)
	if err != nil {
		t.Fatalf("runtime failure must be renderable: %v", err)
	}
	if result.Status != StatusIncomplete || !result.Retryable || !result.Durable || result.OwnedStateVerified || result.ErrorClass != ErrorProjectionResetFailed {
		t.Fatalf("runtime failure result = %+v", result)
	}
	if len(result.CompletedSteps) != 1 || result.CompletedSteps[0] != StepStorageReset {
		t.Fatalf("runtime failure steps = %v", result.CompletedSteps)
	}
	if result.Configuration.ResultLimit != 10 {
		t.Fatalf("runtime failure did not publish defaults: %+v", result.Configuration)
	}
	cached, ok := coordinator.Cached(profileA)
	if !ok || !cached.Retryable || cached.Configuration.ResultLimit != 10 {
		t.Fatalf("cached incomplete reset = %+v ok=%v", cached, ok)
	}
}

func TestResetRequiresRuntimeBeforeDeleting(t *testing.T) {
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	result, err := coordinator.Reset(context.Background(), profileA)
	if err != nil {
		t.Fatalf("missing runtime must be renderable: %v", err)
	}
	if result.Status != StatusRetryable || result.ErrorClass != ErrorResetRuntimeUnavailable {
		t.Fatalf("missing runtime result = %+v", result)
	}
	if store.resetCalls != 0 {
		t.Fatalf("storage reset ran without runtime %d times", store.resetCalls)
	}
}

func TestResetHonorsCancellation(t *testing.T) {
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	coordinator.SetResetRuntime(&fakeResetRuntime{outcome: ProjectionReset{Revision: 1, IndexCleared: true, SnapshotRequested: true}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := coordinator.Reset(ctx, profileA); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if store.resetCalls != 0 {
		t.Fatalf("cancelled reset reached storage %d times", store.resetCalls)
	}
}

func TestResetIsIdempotent(t *testing.T) {
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	resetRuntime := &fakeResetRuntime{outcome: ProjectionReset{Revision: 3, IndexCleared: true, SnapshotRequested: true}}
	coordinator.SetResetRuntime(resetRuntime)
	for attempt := 0; attempt < 2; attempt++ {
		result, err := coordinator.Reset(context.Background(), profileA)
		if err != nil {
			t.Fatalf("reset attempt %d: %v", attempt, err)
		}
		if result.Status != StatusComplete || !result.OwnedStateVerified {
			t.Fatalf("reset attempt %d result = %+v", attempt, result)
		}
	}
	if store.resetCalls != 2 || resetRuntime.calls != 2 {
		t.Fatalf("idempotent reset calls storage=%d runtime=%d", store.resetCalls, resetRuntime.calls)
	}
}

type fakeCloser struct {
	err     error
	profile domain.ProfileID
	calls   int
}

func (closer *fakeCloser) CloseSession(ctx context.Context, profileID domain.ProfileID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	closer.calls++
	closer.profile = profileID
	return closer.err
}

func testUninstallPlan(t *testing.T, root string, entries []Entry, databasePath string) UninstallPlan {
	t.Helper()
	manifest, err := NewManifest(root, entries)
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	plan, err := NewUninstallPlan(manifest, databasePath)
	if err != nil {
		t.Fatalf("uninstall plan: %v", err)
	}
	return plan
}

func TestUninstallRemovesManifestAndClearsState(t *testing.T) {
	root := t.TempDir()
	owned := filepath.Join(root, "owned.bin")
	unrelated := filepath.Join(root, "unrelated.txt")
	databasePath := filepath.Join(root, "infoboard.sqlite")
	for path, contents := range map[string]string{owned: "owned", unrelated: "keep", databasePath: "db"} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan := testUninstallPlan(t, root, []Entry{{ID: "owned", RelativePath: "owned.bin", Kind: KindFile}}, databasePath)
	store := &fakeStore{configs: map[string]storage.Configuration{
		string(profileA): storage.DefaultConfiguration(profileA),
		string(profileB): storage.DefaultConfiguration(profileB),
	}}
	coordinator := newTestCoordinator(t, store)
	coordinator.SetUninstallPlan(plan)
	closer := &fakeCloser{}
	coordinator.SetSessionCloser(closer)
	if _, err := coordinator.Load(context.Background(), profileA); err != nil {
		t.Fatalf("seed load: %v", err)
	}
	result, err := coordinator.Uninstall(context.Background(), profileA)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if result.Status != StatusComplete || result.Retryable || !result.OwnedStateVerified || result.Persistence != PersistenceUnavailable || result.Durable {
		t.Fatalf("uninstall result = %+v", result)
	}
	if len(result.CompletedSteps) != 3 || result.CompletedSteps[0] != StepSessionClosed || result.CompletedSteps[1] != StepStorageUninstall || result.CompletedSteps[2] != StepPathsRemoved {
		t.Fatalf("uninstall steps = %v", result.CompletedSteps)
	}
	if closer.profile != profileA || closer.calls != 1 || store.uninstallCalls != 1 || store.databasePath != databasePath {
		t.Fatalf("uninstall calls closer=%+v storage=%d path=%s", closer, store.uninstallCalls, store.databasePath)
	}
	if _, err := os.Lstat(owned); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned artifact remains: %v", err)
	}
	if contents, err := os.ReadFile(unrelated); err != nil || string(contents) != "keep" {
		t.Fatalf("unrelated artifact changed: %q err=%v", contents, err)
	}
	if _, ok := coordinator.Cached(profileA); ok {
		t.Fatal("effective cache survived uninstall")
	}
}

func TestUninstallMissingArtifactsIsComplete(t *testing.T) {
	root := t.TempDir()
	databasePath := filepath.Join(root, "infoboard.sqlite")
	plan := testUninstallPlan(t, root, []Entry{{ID: "missing", RelativePath: "missing.bin", Kind: KindFile}}, databasePath)
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	coordinator.SetUninstallPlan(plan)
	coordinator.SetSessionCloser(&fakeCloser{})
	result, err := coordinator.Uninstall(context.Background(), profileA)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if result.Status != StatusComplete || !result.OwnedStateVerified || result.Retryable {
		t.Fatalf("missing artifacts result = %+v", result)
	}
}

func TestUninstallRequiresConfiguration(t *testing.T) {
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	result, err := coordinator.Uninstall(context.Background(), profileA)
	if err != nil {
		t.Fatalf("missing configuration must be renderable: %v", err)
	}
	if result.Status != StatusRetryable || result.ErrorClass != ErrorUninstallNotConfigured {
		t.Fatalf("missing configuration result = %+v", result)
	}
	if store.uninstallCalls != 0 {
		t.Fatalf("storage uninstall ran without configuration %d times", store.uninstallCalls)
	}
}

func TestUninstallSessionCloseFailureStopsCleanup(t *testing.T) {
	root := t.TempDir()
	owned := filepath.Join(root, "owned.bin")
	if err := os.WriteFile(owned, []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(root, "infoboard.sqlite")
	plan := testUninstallPlan(t, root, []Entry{{ID: "owned", RelativePath: "owned.bin", Kind: KindFile}}, databasePath)
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	coordinator.SetUninstallPlan(plan)
	coordinator.SetSessionCloser(&fakeCloser{err: errors.New("close failed")})
	result, err := coordinator.Uninstall(context.Background(), profileA)
	if err != nil {
		t.Fatalf("close failure must be renderable: %v", err)
	}
	if result.Status != StatusRetryable || result.ErrorClass != ErrorSessionCloseFailed || store.uninstallCalls != 0 {
		t.Fatalf("close failure result = %+v storage=%d", result, store.uninstallCalls)
	}
	if _, err := os.Lstat(owned); err != nil {
		t.Fatalf("owned artifact was removed before session close: %v", err)
	}
}

func TestUninstallStorageFailureIsRetryable(t *testing.T) {
	root := t.TempDir()
	owned := filepath.Join(root, "owned.bin")
	if err := os.WriteFile(owned, []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(root, "infoboard.sqlite")
	plan := testUninstallPlan(t, root, []Entry{{ID: "owned", RelativePath: "owned.bin", Kind: KindFile}}, databasePath)
	store := &fakeStore{
		configs:         map[string]storage.Configuration{},
		uninstallResult: storage.Result{Status: storage.StatusDegraded, ErrorClass: storage.ErrorStorageFailure, Retryable: true},
	}
	coordinator := newTestCoordinator(t, store)
	coordinator.SetUninstallPlan(plan)
	coordinator.SetSessionCloser(&fakeCloser{})
	result, err := coordinator.Uninstall(context.Background(), profileA)
	if err != nil {
		t.Fatalf("storage failure must be renderable: %v", err)
	}
	if result.Status != StatusRetryable || !result.Retryable || result.ErrorClass != ErrorStorageFailure {
		t.Fatalf("storage failure result = %+v", result)
	}
	if len(result.CompletedSteps) != 1 || result.CompletedSteps[0] != StepSessionClosed {
		t.Fatalf("storage failure steps = %v", result.CompletedSteps)
	}
	if _, err := os.Lstat(owned); err != nil {
		t.Fatalf("owned artifact was removed after storage failure: %v", err)
	}
}

func TestUninstallCleanupFailureIsIncomplete(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "owned-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	databasePath := filepath.Join(root, "infoboard.sqlite")
	plan := testUninstallPlan(t, root, []Entry{{ID: "link", RelativePath: "owned-link", Kind: KindFile}}, databasePath)
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	coordinator.SetUninstallPlan(plan)
	coordinator.SetSessionCloser(&fakeCloser{})
	result, err := coordinator.Uninstall(context.Background(), profileA)
	if err != nil {
		t.Fatalf("cleanup failure must be renderable: %v", err)
	}
	if result.Status != StatusIncomplete || !result.Retryable || result.OwnedStateVerified || result.ErrorClass != ErrorPathSafety {
		t.Fatalf("cleanup failure result = %+v", result)
	}
	if len(result.CompletedSteps) != 2 || result.CompletedSteps[0] != StepSessionClosed || result.CompletedSteps[1] != StepStorageUninstall {
		t.Fatalf("cleanup failure steps = %v", result.CompletedSteps)
	}
	if contents, err := os.ReadFile(outside); err != nil || string(contents) != "outside" {
		t.Fatalf("outside target changed: %q err=%v", contents, err)
	}
}

func TestUninstallHonorsCancellation(t *testing.T) {
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := coordinator.Uninstall(ctx, profileA); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if store.uninstallCalls != 0 {
		t.Fatalf("cancelled uninstall reached storage %d times", store.uninstallCalls)
	}
}

func TestUninstallIsIdempotent(t *testing.T) {
	root := t.TempDir()
	databasePath := filepath.Join(root, "infoboard.sqlite")
	plan := testUninstallPlan(t, root, []Entry{{ID: "missing", RelativePath: "missing.bin", Kind: KindFile}}, databasePath)
	store := &fakeStore{configs: map[string]storage.Configuration{}}
	coordinator := newTestCoordinator(t, store)
	coordinator.SetUninstallPlan(plan)
	closer := &fakeCloser{}
	coordinator.SetSessionCloser(closer)
	for attempt := 0; attempt < 2; attempt++ {
		result, err := coordinator.Uninstall(context.Background(), profileA)
		if err != nil {
			t.Fatalf("uninstall attempt %d: %v", attempt, err)
		}
		if result.Status != StatusComplete || !result.OwnedStateVerified {
			t.Fatalf("uninstall attempt %d result = %+v", attempt, result)
		}
	}
	if store.uninstallCalls != 2 || closer.calls != 2 {
		t.Fatalf("idempotent uninstall calls storage=%d closer=%d", store.uninstallCalls, closer.calls)
	}
}

func TestNewUninstallPlanRejectsUnsafeInputs(t *testing.T) {
	root := t.TempDir()
	manifest, err := NewManifest(root, DefaultEntries())
	if err != nil {
		t.Fatal(err)
	}
	for _, databasePath := range []string{"", "infoboard.sqlite", filepath.Join(root, "other.sqlite")} {
		if _, err := NewUninstallPlan(manifest, databasePath); !errors.Is(err, ErrInvalidConfiguration) {
			t.Fatalf("database path %q must fail: %v", databasePath, err)
		}
	}
	if _, err := NewUninstallPlan(Manifest{}, filepath.Join(root, "infoboard.sqlite")); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("zero manifest must fail: %v", err)
	}
}
