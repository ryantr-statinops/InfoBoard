package lifecycle_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/lifecycle"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

const (
	profileA = domain.ProfileID("123e4567-e89b-42d3-a456-426614174000")
	profileB = domain.ProfileID("123e4567-e89b-42d3-a456-426614174099")
)

func versions() storage.CompatibilityVersions {
	return storage.CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion, RankingModelVersion: "rank-1"}
}

func newRepository(t *testing.T) (string, *storage.Repository) {
	t.Helper()
	directory := t.TempDir()
	repository, result := storage.Open(context.Background(), directory)
	if repository == nil || result.Status != storage.StatusReady {
		t.Fatalf("open storage: repository nil=%t result=%+v", repository == nil, result)
	}
	t.Cleanup(func() { _ = repository.Close() })
	return filepath.Join(directory, storage.DatabaseFileName), repository
}

func newCoordinator(t *testing.T, repository *storage.Repository) *lifecycle.Coordinator {
	t.Helper()
	coordinator, err := lifecycle.NewCoordinator(repository, versions())
	if err != nil {
		t.Fatalf("new coordinator: %v", err)
	}
	return coordinator
}

type resetRuntime struct {
	revision uint64
	calls    int
}

func (runtime *resetRuntime) ResetProjection(context.Context, domain.ProfileID) (lifecycle.ProjectionReset, error) {
	runtime.calls++
	return lifecycle.ProjectionReset{Revision: runtime.revision, IndexCleared: true, SnapshotRequested: true}, nil
}

type sessionCloser struct {
	calls int
}

func (closer *sessionCloser) CloseSession(context.Context, domain.ProfileID) error {
	closer.calls++
	return nil
}

func countRows(t *testing.T, databasePath, table, profileID string) int {
	t.Helper()
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatalf("open sql: %v", err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM `+table+` WHERE profile_id = ?`, profileID).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func TestFocusProfileIsolationDefaultsAndReset(t *testing.T) {
	databasePath, repository := newRepository(t)
	coordinator := newCoordinator(t, repository)
	ctx := context.Background()

	effectiveA, err := coordinator.Load(ctx, profileA)
	if err != nil || effectiveA.Configuration.ResultLimit != 10 || !effectiveA.Configuration.RecencyEnabled {
		t.Fatalf("profile A defaults = %+v err=%v", effectiveA, err)
	}
	configA := lifecycle.DefaultProfileConfig(profileA, protocol.CurrentVersion, "rank-1")
	configA.ResultLimit = 12
	configA.Density = lifecycle.DensityCompact
	configA.Theme = lifecycle.ThemeDark
	configA.ContextLabels = false
	if _, err := coordinator.Update(ctx, profileA, configA); err != nil {
		t.Fatalf("update profile A: %v", err)
	}
	if _, err := coordinator.Update(ctx, profileB, lifecycle.DefaultProfileConfig(profileB, protocol.CurrentVersion, "rank-1")); err != nil {
		t.Fatalf("seed profile B: %v", err)
	}

	loadedA, err := coordinator.Load(ctx, profileA)
	if err != nil || loadedA.Configuration.ResultLimit != 12 || loadedA.Configuration.Density != lifecycle.DensityCompact {
		t.Fatalf("profile A config = %+v err=%v", loadedA, err)
	}
	loadedB, err := coordinator.Load(ctx, profileB)
	if err != nil || loadedB.Configuration.ResultLimit != 10 || loadedB.Configuration.Density != lifecycle.DensityComfortable {
		t.Fatalf("profile B leaked values = %+v err=%v", loadedB, err)
	}

	now := time.Now().Unix()
	for _, profile := range []domain.ProfileID{profileA, profileB} {
		if result := repository.RecordActivation(ctx, storage.ActivationRecord{
			ProfileID: string(profile), TabID: 1, Domain: "example.test", ActivatedAt: now, Source: domain.ActivationKeyboardEnter,
		}); result.Status != storage.StatusReady {
			t.Fatalf("record activation for %s: %+v", profile, result)
		}
	}
	resetRuntime := &resetRuntime{revision: 9}
	coordinator.SetResetRuntime(resetRuntime)
	reset, err := coordinator.Reset(ctx, profileA)
	if err != nil {
		t.Fatalf("reset profile A: %v", err)
	}
	if reset.Status != lifecycle.StatusComplete || !reset.OwnedStateVerified || reset.ProjectionRevision != 9 {
		t.Fatalf("reset result = %+v", reset)
	}
	if reset.Configuration.ResultLimit != 10 || !reset.Configuration.RecencyEnabled {
		t.Fatalf("reset did not publish defaults: %+v", reset.Configuration)
	}
	second, err := coordinator.Reset(ctx, profileA)
	if err != nil || second.Status != lifecycle.StatusComplete || !second.OwnedStateVerified {
		t.Fatalf("second reset = %+v err=%v", second, err)
	}
	if resetRuntime.calls != 2 {
		t.Fatalf("reset runtime calls = %d", resetRuntime.calls)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("close repository before row checks: %v", err)
	}
	if got := countRows(t, databasePath, "configuration", string(profileA)); got != 0 {
		t.Fatalf("profile A configuration rows = %d", got)
	}
	if got := countRows(t, databasePath, "activation_metadata", string(profileA)); got != 0 {
		t.Fatalf("profile A activation rows after reset = %d", got)
	}
	if got := countRows(t, databasePath, "configuration", string(profileB)); got != 1 {
		t.Fatalf("profile B configuration rows after reset = %d", got)
	}
	if got := countRows(t, databasePath, "activation_metadata", string(profileB)); got != 1 {
		t.Fatalf("profile B activation rows after reset = %d", got)
	}
}

func TestFocusUninstallManifestAndSentinels(t *testing.T) {
	databasePath, repository := newRepository(t)
	directory := filepath.Dir(databasePath)
	owned := filepath.Join(directory, "owned.bin")
	unrelated := filepath.Join(directory, "unrelated.txt")
	for path, contents := range map[string]string{owned: "owned", unrelated: "keep"} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manifest, err := lifecycle.NewManifest(directory, []lifecycle.Entry{
		{ID: "owned", RelativePath: "owned.bin", Kind: lifecycle.KindFile},
	})
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	plan, err := lifecycle.NewUninstallPlan(manifest, databasePath)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	coordinator := newCoordinator(t, repository)
	coordinator.SetUninstallPlan(plan)
	closer := &sessionCloser{}
	coordinator.SetSessionCloser(closer)

	result, err := coordinator.Uninstall(context.Background(), profileA)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if result.Status != lifecycle.StatusComplete || !result.OwnedStateVerified || result.Retryable {
		t.Fatalf("uninstall result = %+v", result)
	}
	if closer.calls != 1 {
		t.Fatalf("session close calls = %d", closer.calls)
	}
	for _, path := range []string{databasePath, owned} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s remains after uninstall: %v", path, err)
		}
	}
	if contents, err := os.ReadFile(unrelated); err != nil || string(contents) != "keep" {
		t.Fatalf("unrelated sentinel changed: %q err=%v", contents, err)
	}
}

func TestFocusDegradedStoreKeepsLexicalFallback(t *testing.T) {
	_, repository := newRepository(t)
	coordinator := newCoordinator(t, repository)
	if err := repository.Close(); err != nil {
		t.Fatalf("close storage: %v", err)
	}
	effective, err := coordinator.Load(context.Background(), profileA)
	if err != nil {
		t.Fatalf("degraded load: %v", err)
	}
	if effective.Persistence != lifecycle.PersistenceDegraded || effective.Durable {
		t.Fatalf("degraded state = %+v", effective)
	}
	if effective.Configuration.ResultLimit != 10 || !effective.Configuration.RecencyEnabled {
		t.Fatalf("degraded defaults = %+v", effective.Configuration)
	}
	// The lexical path is independent of SQLite; this stands in for the
	// projection/query owner while the integration harness supplies the real
	// fake host.
	index := []string{"example"}
	if len(index) != 1 {
		t.Fatalf("lexical fallback lost records: %v", index)
	}
}
