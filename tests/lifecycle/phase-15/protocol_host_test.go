package lifecycle_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/internal/runtime"
	"github.com/ryantr-statinops/InfoBoard/host/lifecycle"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

// recordingChannel is the fake native host's outbound pipe. It keeps the
// complete wire frames so tests assert the protocol result rather than an
// implementation-specific lifecycle value.
type recordingChannel struct {
	mu     sync.Mutex
	frames [][]byte
}

func (channel *recordingChannel) WriteFrame(_ context.Context, payload []byte) error {
	channel.mu.Lock()
	defer channel.mu.Unlock()
	channel.frames = append(channel.frames, append([]byte(nil), payload...))
	return nil
}

func (channel *recordingChannel) reset() {
	channel.mu.Lock()
	defer channel.mu.Unlock()
	channel.frames = nil
}

func (channel *recordingChannel) last(t *testing.T) protocol.Envelope {
	t.Helper()
	channel.mu.Lock()
	defer channel.mu.Unlock()
	if len(channel.frames) != 1 {
		t.Fatalf("frame count = %d, want 1", len(channel.frames))
	}
	envelope, failure := protocol.DecodeEnvelope(channel.frames[0], protocol.DefaultLimits())
	if failure != nil {
		t.Fatalf("decode frame: %+v", failure)
	}
	return envelope
}

// fakeLifecycleAdapter is the composition seam the real host will supply later:
// it maps the runtime hook request onto the lifecycle coordinator and returns
// only the redacted result bytes plus runtime actions.
type fakeLifecycleAdapter struct {
	coordinator *lifecycle.Coordinator
}

func (adapter *fakeLifecycleAdapter) ExecuteLifecycle(
	ctx context.Context,
	request runtime.LifecycleRequest,
) (runtime.LifecycleOutcome, error) {
	switch request.Operation {
	case runtime.LifecycleLoad:
		effective, err := adapter.coordinator.Load(ctx, request.ProfileID)
		if err != nil {
			return runtime.LifecycleOutcome{}, err
		}
		return encodeLifecycleOutcome(resultFromEffective(string(request.Operation), effective), false, false)
	case runtime.LifecycleUpdate:
		configuration, err := lifecycle.DecodeProfileConfig(request.Configuration, request.ProfileID)
		if err != nil {
			return runtime.LifecycleOutcome{}, err
		}
		effective, err := adapter.coordinator.Update(ctx, request.ProfileID, configuration)
		if err != nil {
			return runtime.LifecycleOutcome{}, err
		}
		return encodeLifecycleOutcome(resultFromEffective(string(request.Operation), effective), false, false)
	case runtime.LifecycleReset:
		result, err := adapter.coordinator.Reset(ctx, request.ProfileID)
		if err != nil {
			return runtime.LifecycleOutcome{}, err
		}
		return encodeLifecycleOutcome(result, true, false)
	case runtime.LifecycleUninstall:
		result, err := adapter.coordinator.Uninstall(ctx, request.ProfileID)
		if err != nil {
			return runtime.LifecycleOutcome{}, err
		}
		return encodeLifecycleOutcome(result, false, result.Status == lifecycle.StatusComplete)
	default:
		return runtime.LifecycleOutcome{}, lifecycle.ErrInvalidConfiguration
	}
}

func resultFromEffective(operation string, effective lifecycle.EffectiveConfiguration) lifecycle.LifecycleResult {
	status := lifecycle.StatusComplete
	if effective.Persistence != lifecycle.PersistenceReady || !effective.Durable {
		status = lifecycle.StatusDegraded
	}
	return lifecycle.LifecycleResult{
		Operation:          operation,
		ProfileID:          effective.Configuration.ProfileID,
		Status:             status,
		Retryable:          effective.Retryable,
		Persistence:        effective.Persistence,
		Durable:            effective.Durable,
		ErrorClass:         effective.ErrorClass,
		OwnedStateVerified: effective.Durable,
		Configuration:      effective.Configuration,
	}
}

func encodeLifecycleOutcome(
	result lifecycle.LifecycleResult,
	resync bool,
	shutdown bool,
) (runtime.LifecycleOutcome, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return runtime.LifecycleOutcome{}, err
	}
	return runtime.LifecycleOutcome{Payload: payload, RequireResync: resync, Shutdown: shutdown}, nil
}

type fakeNativeHost struct {
	profile domain.ProfileID
	session *protocol.Session
	channel *recordingChannel
	nextID  int
}

func newFakeNativeHost(
	t *testing.T,
	coordinator *lifecycle.Coordinator,
	profile domain.ProfileID,
) *fakeNativeHost {
	t.Helper()
	session, err := protocol.NewSession(protocol.Config{
		HostVersion:         "0.1.0",
		RankingModelVersion: "rank-1",
		Limits:              protocol.DefaultLimits(),
		Budgets:             protocol.DefaultBudgets(),
	}, nil, nil, nil, &fakeLifecycleAdapter{coordinator: coordinator})
	if err != nil {
		t.Fatalf("new protocol session: %v", err)
	}
	channel := &recordingChannel{}
	hello := hostFrame(t, profile, protocol.TypeHello, "hello-1", map[string]any{
		"extension_version":           "1.0.0",
		"browser_family":              "chrome",
		"browser_version":             "126.0",
		"context_kind":                "normal",
		"capabilities":                []string{"query"},
		"supported_protocol_versions": []int{protocol.CurrentVersion},
	})
	if _, err := session.Negotiate(context.Background(), channel, hello); err != nil {
		t.Fatalf("negotiate fake host: %v", err)
	}
	channel.reset()
	return &fakeNativeHost{profile: profile, session: session, channel: channel}
}

func hostFrame(
	t *testing.T,
	profile domain.ProfileID,
	messageType protocol.MessageType,
	requestID string,
	payload any,
) []byte {
	t.Helper()
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	frame, failure := protocol.EncodeEnvelope(protocol.Envelope{
		Protocol:  protocol.CurrentVersion,
		Type:      messageType,
		RequestID: requestID,
		ProfileID: string(profile),
		Payload:   encodedPayload,
	}, protocol.DefaultLimits())
	if failure != nil {
		t.Fatalf("encode frame: %+v", failure)
	}
	return frame
}

func (host *fakeNativeHost) request(t *testing.T, operation string, configuration []byte) protocol.LifecycleResult {
	t.Helper()
	payload := map[string]any{"operation": operation}
	if configuration != nil {
		payload["configuration"] = json.RawMessage(configuration)
	}
	host.nextID++
	frame := hostFrame(t, host.profile, protocol.TypeLifecycle, fmt.Sprintf("lifecycle-%s-%d", operation, host.nextID), payload)
	host.channel.reset()
	if err := host.session.Accept(context.Background(), host.channel, frame); err != nil {
		t.Fatalf("%s lifecycle: %v", operation, err)
	}
	envelope := host.channel.last(t)
	if envelope.Type != protocol.TypeLifecycleResult {
		t.Fatalf("%s response type = %s", operation, envelope.Type)
	}
	var result protocol.LifecycleResult
	if err := json.Unmarshal(envelope.Payload, &result); err != nil {
		t.Fatalf("%s decode result: %v", operation, err)
	}
	return result
}

func TestProtocolFakeHostLifecycleIntegration(t *testing.T) {
	databasePath, repository := newRepository(t)
	coordinator := newCoordinator(t, repository)
	coordinator.SetResetRuntime(&resetRuntime{revision: 9})
	directory := filepath.Dir(databasePath)
	owned := filepath.Join(directory, "owned.bin")
	unrelated := filepath.Join(directory, "unrelated.txt")
	if err := os.WriteFile(owned, []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unrelated, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := lifecycle.NewManifest(directory, []lifecycle.Entry{
		{ID: "owned", RelativePath: "owned.bin", Kind: lifecycle.KindFile},
	})
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	plan, err := lifecycle.NewUninstallPlan(manifest, databasePath)
	if err != nil {
		t.Fatalf("uninstall plan: %v", err)
	}
	coordinator.SetUninstallPlan(plan)
	coordinator.SetSessionCloser(&sessionCloser{})

	hostA := newFakeNativeHost(t, coordinator, profileA)
	hostB := newFakeNativeHost(t, coordinator, profileB)

	loadedA := hostA.request(t, "load", nil)
	if loadedA.ProfileID != profileA || loadedA.Configuration.ResultLimit != 10 || !loadedA.Configuration.RecencyEnabled {
		t.Fatalf("profile A defaults = %+v", loadedA)
	}
	if loadedA.CompletedSteps == nil {
		t.Fatal("lifecycle result encoded completed_steps as null instead of an array")
	}
	configurationA := lifecycle.DefaultProfileConfig(profileA, protocol.CurrentVersion, "rank-1")
	configurationA.ResultLimit = 25
	configurationA.Density = lifecycle.DensityCompact
	configurationA.Theme = lifecycle.ThemeDark
	configurationA.ContextLabels = false
	configurationA.RecencyEnabled = false
	rawA, err := json.Marshal(configurationA)
	if err != nil {
		t.Fatal(err)
	}
	updatedA := hostA.request(t, "update", rawA)
	if updatedA.Status != "complete" || updatedA.Configuration.ResultLimit != 25 || updatedA.Configuration.Theme != "dark" {
		t.Fatalf("profile A update = %+v", updatedA)
	}

	loadedB := hostB.request(t, "load", nil)
	if loadedB.ProfileID != profileB || loadedB.Configuration.ResultLimit != 10 || !loadedB.Configuration.RecencyEnabled {
		t.Fatalf("profile B defaults = %+v", loadedB)
	}
	configurationB := lifecycle.DefaultProfileConfig(profileB, protocol.CurrentVersion, "rank-1")
	configurationB.ResultLimit = 33
	rawB, err := json.Marshal(configurationB)
	if err != nil {
		t.Fatal(err)
	}
	updatedB := hostB.request(t, "update", rawB)
	if updatedB.Configuration.ResultLimit != 33 {
		t.Fatalf("profile B update = %+v", updatedB)
	}

	resetA := hostA.request(t, "reset", nil)
	if resetA.Status != "complete" || !resetA.ResyncRequired || resetA.Configuration.ResultLimit != 10 {
		t.Fatalf("profile A reset = %+v", resetA)
	}
	secondResetA := hostA.request(t, "reset", nil)
	if secondResetA.Status != "complete" || secondResetA.Configuration.ResultLimit != 10 {
		t.Fatalf("profile A second reset = %+v", secondResetA)
	}
	loadedBAfterReset := hostB.request(t, "load", nil)
	if loadedBAfterReset.Configuration.ResultLimit != 33 {
		t.Fatalf("profile B changed after profile A reset = %+v", loadedBAfterReset)
	}

	uninstalledA := hostA.request(t, "uninstall", nil)
	if uninstalledA.Status != "complete" || !uninstalledA.OwnedStateVerified || !containsStep(uninstalledA.CompletedSteps, "storage_uninstall") {
		t.Fatalf("profile A uninstall = %+v", uninstalledA)
	}
	if hostA.session.State() != protocol.StateShutdown {
		t.Fatalf("session state after uninstall = %s", hostA.session.State())
	}
	for _, path := range []string{databasePath, owned} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s remains after uninstall: %v", path, err)
		}
	}
	if contents, err := os.ReadFile(unrelated); err != nil || string(contents) != "keep" {
		t.Fatalf("unrelated sentinel changed: %q err=%v", contents, err)
	}

	secondHostA := newFakeNativeHost(t, coordinator, profileA)
	secondUninstallA := secondHostA.request(t, "uninstall", nil)
	if secondUninstallA.Status != "complete" {
		t.Fatalf("idempotent uninstall = %+v", secondUninstallA)
	}
}

func containsStep(steps []string, want string) bool {
	for _, step := range steps {
		if step == want {
			return true
		}
	}
	return false
}

func TestProtocolFakeHostDegradedPersistenceKeepsDefaults(t *testing.T) {
	_, repository := newRepository(t)
	coordinator := newCoordinator(t, repository)
	if err := repository.Close(); err != nil {
		t.Fatalf("close repository: %v", err)
	}
	host := newFakeNativeHost(t, coordinator, profileA)
	result := host.request(t, "load", nil)
	if result.Status != "degraded" || result.Persistence != "degraded" || result.Durable {
		t.Fatalf("degraded result = %+v", result)
	}
	if result.Configuration.ResultLimit != 10 || !result.Configuration.RecencyEnabled {
		t.Fatalf("degraded defaults = %+v", result.Configuration)
	}
}

func TestProtocolFakeHostRejectsUnsafeUninstallPath(t *testing.T) {
	databasePath, repository := newRepository(t)
	_ = repository.Close()
	manifest, err := lifecycle.NewManifest(filepath.Dir(databasePath), []lifecycle.Entry{
		{ID: "database", RelativePath: storage.DatabaseFileName, Kind: lifecycle.KindFile},
	})
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if _, err := lifecycle.NewUninstallPlan(manifest, storage.DatabaseFileName); !errors.Is(err, lifecycle.ErrInvalidConfiguration) {
		t.Fatalf("relative database path accepted: %v", err)
	}
}
