package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/runtime"
)

type fakeLifecycleHook struct {
	mu       sync.Mutex
	requests []runtime.LifecycleRequest
	outcomes map[runtime.LifecycleOperation]runtime.LifecycleOutcome
	err      error
	delay    time.Duration
}

func (hook *fakeLifecycleHook) ExecuteLifecycle(ctx context.Context, request runtime.LifecycleRequest) (runtime.LifecycleOutcome, error) {
	hook.mu.Lock()
	hook.requests = append(hook.requests, request)
	hook.mu.Unlock()
	if hook.delay > 0 {
		select {
		case <-time.After(hook.delay):
		case <-ctx.Done():
			return runtime.LifecycleOutcome{}, ctx.Err()
		}
	}
	if hook.err != nil {
		return runtime.LifecycleOutcome{}, hook.err
	}
	return hook.outcomes[request.Operation], nil
}

func (hook *fakeLifecycleHook) count() int {
	hook.mu.Lock()
	defer hook.mu.Unlock()
	return len(hook.requests)
}

func lifecycleResult(operation string) LifecycleResult {
	return LifecycleResult{
		Operation:   operation,
		ProfileID:   domain.ProfileID(testProfile),
		Status:      lifecycleStatusComplete,
		Persistence: lifecyclePersistenceReady,
		Durable:     true,
		Configuration: LifecycleConfiguration{
			ProfileID:           testProfile,
			ResultLimit:         10,
			Density:             "comfortable",
			Theme:               "system",
			ContextLabels:       true,
			RecencyEnabled:      true,
			ProtocolVersion:     1,
			RankingModelVersion: "rank-1",
		},
	}
}

func lifecycleOutcome(t *testing.T, result LifecycleResult, resync, shutdown bool) runtime.LifecycleOutcome {
	t.Helper()
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return runtime.LifecycleOutcome{Payload: payload, RequireResync: resync, Shutdown: shutdown}
}

func newLifecycleSession(t *testing.T, hook runtime.LifecycleHook) (*Session, *recorder) {
	t.Helper()
	session, err := NewSession(testConfig(), &fakeSync{}, &fakeQuery{}, nil, hook)
	if err != nil {
		t.Fatal(err)
	}
	out := &recorder{}
	mustBind(t, session, out)
	out.reset()
	return session, out
}

func TestLifecycle_DispatchLoadUpdateResetAndUninstall(t *testing.T) {
	hook := &fakeLifecycleHook{outcomes: map[runtime.LifecycleOperation]runtime.LifecycleOutcome{
		runtime.LifecycleLoad:      lifecycleOutcome(t, lifecycleResult(lifecycleOperationLoad), false, false),
		runtime.LifecycleUpdate:    lifecycleOutcome(t, lifecycleResult(lifecycleOperationUpdate), false, false),
		runtime.LifecycleReset:     lifecycleOutcome(t, lifecycleResult(lifecycleOperationReset), true, false),
		runtime.LifecycleUninstall: lifecycleOutcome(t, lifecycleResult(lifecycleOperationUninstall), false, true),
	}}
	session, out := newLifecycleSession(t, hook)

	configuration := `{"profile_id":"` + testProfile + `","result_limit":25,"density":"compact","theme":"dark","context_labels":false,"recency_enabled":false,"protocol_version":1,"ranking_model_version":"rank-1"}`
	requests := []struct {
		requestID string
		payload   string
		operation runtime.LifecycleOperation
		resync    bool
	}{
		{"lc-load", `{"operation":"load"}`, runtime.LifecycleLoad, false},
		{"lc-update", `{"operation":"update","configuration":` + configuration + `}`, runtime.LifecycleUpdate, false},
		{"lc-reset", `{"operation":"reset"}`, runtime.LifecycleReset, true},
	}
	for _, request := range requests {
		out.reset()
		if err := session.Accept(context.Background(), out, frame(t, TypeLifecycle, request.requestID, 0, request.payload)); err != nil {
			t.Fatalf("%s: %v", request.operation, err)
		}
		frames := out.envelopes(t)
		if len(frames) != 1 || frames[0].Type != TypeLifecycleResult {
			t.Fatalf("%s frames = %+v", request.operation, frames)
		}
		var result LifecycleResult
		if err := json.Unmarshal(frames[0].Payload, &result); err != nil {
			t.Fatal(err)
		}
		if result.Operation != string(request.operation) ||
			result.ProfileID != domain.ProfileID(testProfile) ||
			result.Status != lifecycleStatusComplete ||
			result.ResyncRequired != request.resync {
			t.Fatalf("%s result = %+v", request.operation, result)
		}
	}
	if hook.count() != 3 {
		t.Fatalf("hook calls = %d, want 3", hook.count())
	}
	if !session.isRebuilding() {
		t.Fatal("reset did not mark the session rebuilding")
	}
	if hook.requests[1].Operation != runtime.LifecycleUpdate || len(hook.requests[1].Configuration) == 0 {
		t.Fatalf("update handoff = %+v", hook.requests[1])
	}

	out.reset()
	if err := session.Accept(context.Background(), out, frame(t, TypeLifecycle, "lc-uninstall", 0, `{"operation":"uninstall"}`)); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	frames := out.envelopes(t)
	if len(frames) != 1 || frames[0].Type != TypeLifecycleResult {
		t.Fatalf("uninstall frames = %+v", frames)
	}
	if session.State() != StateShutdown {
		t.Fatalf("state after uninstall = %s", session.State())
	}
}

func TestLifecycle_RejectsMalformedRequestsBeforeHook(t *testing.T) {
	hook := &fakeLifecycleHook{outcomes: map[runtime.LifecycleOperation]runtime.LifecycleOutcome{}}
	session, out := newLifecycleSession(t, hook)
	configuration := `{"profile_id":"` + testProfile + `","result_limit":10,"density":"comfortable","theme":"system","context_labels":true,"recency_enabled":true,"protocol_version":1,"ranking_model_version":"rank-1"}`
	otherProfile := `{"profile_id":"123e4567-e89b-42d3-a456-426614174999","result_limit":10,"density":"comfortable","theme":"system","context_labels":true,"recency_enabled":true,"protocol_version":1,"ranking_model_version":"rank-1"}`
	cases := []struct {
		name    string
		payload string
		want    ErrorCode
	}{
		{"missing_operation", `{}`, CodeInvalidFrame},
		{"unknown_operation", `{"operation":"teleport"}`, CodeInvalidFrame},
		{"missing_update_configuration", `{"operation":"update"}`, CodeInvalidFrame},
		{"configuration_on_load", `{"operation":"load","configuration":` + configuration + `}`, CodeInvalidFrame},
		{"invalid_limit", `{"operation":"update","configuration":{"profile_id":"` + testProfile + `","result_limit":0,"density":"comfortable","theme":"system","context_labels":true,"recency_enabled":true,"protocol_version":1,"ranking_model_version":"rank-1"}}`, CodeInvalidFrame},
		{"invalid_theme", `{"operation":"update","configuration":{"profile_id":"` + testProfile + `","result_limit":10,"density":"comfortable","theme":"neon","context_labels":true,"recency_enabled":true,"protocol_version":1,"ranking_model_version":"rank-1"}}`, CodeInvalidFrame},
		{"profile_mismatch", `{"operation":"update","configuration":` + otherProfile + `}`, CodeProfileMismatch},
		{"unknown_key", `{"operation":"load","extra":true}`, CodeInvalidFrame},
		{"duplicate_operation", `{"operation":"load","operation":"load"}`, CodeInvalidFrame},
	}
	for index, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			out.reset()
			_ = session.Accept(context.Background(), out, frame(t, TypeLifecycle, "lc-invalid-"+string(rune('a'+index)), 0, testCase.payload))
			if code := lastErrorCode(t, out); code != testCase.want {
				t.Fatalf("code = %s, want %s", code, testCase.want)
			}
		})
	}
	if hook.count() != 0 {
		t.Fatalf("malformed requests reached the hook: %d", hook.count())
	}
}

func TestLifecycle_HookFailureAndBudgetAreTyped(t *testing.T) {
	hook := &fakeLifecycleHook{err: errors.New("storage unavailable")}
	session, out := newLifecycleSession(t, hook)
	_ = session.Accept(context.Background(), out, frame(t, TypeLifecycle, "lc-failure", 0, `{"operation":"load"}`))
	if code := lastErrorCode(t, out); code != CodeInternalFailure {
		t.Fatalf("failure code = %s", code)
	}

	hook = &fakeLifecycleHook{delay: 50 * time.Millisecond, outcomes: map[runtime.LifecycleOperation]runtime.LifecycleOutcome{}}
	session, out = newLifecycleSession(t, hook)
	session.config.Budgets.Lifecycle = time.Millisecond
	_ = session.Accept(context.Background(), out, frame(t, TypeLifecycle, "lc-timeout", 0, `{"operation":"load"}`))
	if code := lastErrorCode(t, out); code != CodeQueryTimeout {
		t.Fatalf("timeout code = %s", code)
	}
}

func TestLifecycle_UnboundAndMissingHookFailClosed(t *testing.T) {
	session, err := NewSession(testConfig(), &fakeSync{}, &fakeQuery{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := &recorder{}
	if err := session.Accept(context.Background(), out, frame(t, TypeLifecycle, "lc-unbound", 0, `{"operation":"load"}`)); err == nil {
		t.Fatal("an unbound lifecycle request was accepted")
	}

	out = &recorder{}
	mustBind(t, session, out)
	out.reset()
	_ = session.Accept(context.Background(), out, frame(t, TypeLifecycle, "lc-no-hook", 0, `{"operation":"load"}`))
	if code := lastErrorCode(t, out); code != CodeInternalFailure {
		t.Fatalf("missing hook code = %s", code)
	}
}
