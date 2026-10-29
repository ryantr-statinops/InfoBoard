package diagnostics

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestEventValidationAndRecorder(t *testing.T) {
	now := func() time.Time { return time.Unix(500, 0).UTC() }
	recorder, err := NewRecorder("session-1", 2, now)
	if err != nil {
		t.Fatal(err)
	}
	event, err := recorder.Record(Event{
		Component:         ComponentHost,
		ReasonCode:        ReasonHostStart,
		RecommendedAction: ActionNone,
		ProtocolVersion:   1,
		Timing:            EventTiming{Latency: LatencyFast},
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.SchemaVersion != EventSchemaVersion || event.EventID != "evt-session-1-1" || event.SessionID != "session-1" {
		t.Fatalf("event metadata = %+v", event)
	}
	if !event.EventTime.Equal(time.Unix(500, 0).UTC()) {
		t.Fatalf("event time = %s", event.EventTime)
	}
	if _, err := recorder.Record(Event{
		Component: ComponentHost, ReasonCode: ReasonHostStart, RecommendedAction: ActionNone,
		Timing: EventTiming{Latency: LatencyFast},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := recorder.Record(Event{
		Component: ComponentHost, ReasonCode: ReasonHostExit, RecommendedAction: ActionNone,
		Timing: EventTiming{Latency: LatencyFast},
	}); err != nil {
		t.Fatal(err)
	}
	if got := len(recorder.Events()); got != 2 {
		t.Fatalf("ring length = %d", got)
	}
	if recorder.Dropped() != 1 {
		t.Fatalf("dropped = %d", recorder.Dropped())
	}
	if recorder.PersistenceDegraded() {
		t.Fatal("recorder without sink reported degraded")
	}
}

func TestEventSinkFailureKeepsBoundedRing(t *testing.T) {
	recorder, err := NewRecorder("session-2", 4, func() time.Time { return time.Unix(600, 0).UTC() })
	if err != nil {
		t.Fatal(err)
	}
	recorder.SetSink(func(Event) error { return errors.New("sqlite unavailable") })
	event, err := recorder.Record(Event{
		Component: ComponentStorage, ReasonCode: ReasonStorageDegradation,
		ErrorClass: ErrorPersistenceDegraded, RecommendedAction: ActionRepairPersistence,
		Retryable: true, Timing: EventTiming{Latency: LatencySlow},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(recorder.Events()) != 1 || recorder.Events()[0].EventID != event.EventID {
		t.Fatalf("event was not retained: %+v", recorder.Events())
	}
	if !recorder.PersistenceDegraded() || recorder.PersistenceFailed() != 1 {
		t.Fatalf("persistence counters = %d/%t", recorder.PersistenceFailed(), recorder.PersistenceDegraded())
	}
	if recorder.LastError() == nil || !strings.Contains(recorder.LastError().Error(), "sqlite") {
		t.Fatalf("last error = %v", recorder.LastError())
	}
}

func TestRecorderAttachEmitsOneTransitionPerHealthChange(t *testing.T) {
	health, err := NewHealth(baseSnapshot(), func() time.Time { return time.Unix(700, 0).UTC() })
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := NewRecorder("session-3", 16, func() time.Time { return time.Unix(700, 0).UTC() })
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Attach(health); err != nil {
		t.Fatal(err)
	}
	next := readySnapshot()
	next.State = StateConnecting
	next.Host = HostAvailable
	next.Protocol = ProtocolUnknown
	next.Projection = ProjectionUnknown
	next.Index = IndexUnknown
	next.Storage = StorageUnknown
	if err := health.Transition(next); err != nil {
		t.Fatal(err)
	}
	next.State = StateHandshaking
	next.Protocol = ProtocolCompatible
	if err := health.Transition(next); err != nil {
		t.Fatal(err)
	}
	events := recorder.Events()
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2: %+v", len(events), events)
	}
	if events[0].FromState != StateDisconnected || events[0].ToState != StateConnecting {
		t.Fatalf("first transition = %s -> %s", events[0].FromState, events[0].ToState)
	}
	if events[1].FromState != StateConnecting || events[1].ToState != StateHandshaking {
		t.Fatalf("second transition = %s -> %s", events[1].FromState, events[1].ToState)
	}
	if events[1].EventSequence == 0 {
		t.Fatal("event sequence missing")
	}
	if health.Transitions() != 2 {
		t.Fatalf("health transitions = %d", health.Transitions())
	}
}

func TestEventValidationRejectsUnsafeValues(t *testing.T) {
	base := Event{
		SchemaVersion:     EventSchemaVersion,
		EventID:           "evt-1",
		EventTime:         time.Unix(1, 0).UTC(),
		SessionID:         "session-1",
		Component:         ComponentHost,
		ReasonCode:        ReasonHostStart,
		RecommendedAction: ActionNone,
		Timing:            EventTiming{Latency: LatencyUnknown},
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("base event invalid: %v", err)
	}
	invalid := base
	invalid.ReasonCode = ReasonCode("free text")
	if !errors.Is(invalid.Validate(), ErrInvalidEvent) {
		t.Fatal("unknown reason accepted")
	}
	invalid = base
	invalid.EventTime = time.Unix(1, 0)
	if !errors.Is(invalid.Validate(), ErrInvalidEvent) {
		t.Fatal("non-UTC event time accepted")
	}
	invalid = base
	invalid.SessionID = "https://example.test/?token=secret"
	if !errors.Is(invalid.Validate(), ErrInvalidEvent) {
		t.Fatal("unsafe identifier accepted")
	}
	invalid = base
	invalid.Timing.DurationMillis = -1
	if !errors.Is(invalid.Validate(), ErrInvalidEvent) {
		t.Fatal("negative duration accepted")
	}
	invalid = base
	invalid.Timing.Latency = LatencyBucket("slow-ish")
	if !errors.Is(invalid.Validate(), ErrInvalidEvent) {
		t.Fatal("unknown latency accepted")
	}
}

func TestAllReasonCodesAndActionsAreBounded(t *testing.T) {
	reasons := []ReasonCode{
		ReasonStateTransition, ReasonHostStart, ReasonHostExit, ReasonHandshakeSuccess,
		ReasonHandshakeFailure, ReasonProtocolRejection, ReasonSnapshotAccepted,
		ReasonDeltaGap, ReasonIndexBuildStart, ReasonIndexBuildFinish,
		ReasonIndexBuildFailure, ReasonQueryTimeout, ReasonStorageMigration,
		ReasonStorageDegradation, ReasonStorageRecovery, ReasonRepairStart,
		ReasonRepairFinish, ReasonRepairExhausted, ReasonReset,
		ReasonRetentionEviction, ReasonPersistenceDegraded, ReasonInternalFailure,
	}
	for _, reason := range reasons {
		if !reason.Valid() {
			t.Fatalf("reason %q invalid", reason)
		}
	}
	actions := []RecommendedAction{
		ActionNone, ActionRetry, ActionReconnect, ActionRefreshProfile,
		ActionRefreshProjection, ActionRebuildIndex, ActionRepairPersistence,
		ActionUpdateHost, ActionShortenQuery, ActionWait, ActionOpenDiagnostics,
		ActionRestartHost,
	}
	for _, action := range actions {
		if !action.Valid() {
			t.Fatalf("action %q invalid", action)
		}
	}
	for _, state := range []State{StateReady, StateDegraded, StateIncompatible, StateStopping, StateDisconnected, StateRecovering, StateConnecting} {
		if !ActionForState(state).Valid() {
			t.Fatalf("action for %s invalid", state)
		}
	}
}
