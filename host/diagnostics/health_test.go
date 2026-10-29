package diagnostics

import (
	"errors"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

func baseSnapshot() Snapshot {
	return Snapshot{
		State:               StateDisconnected,
		Host:                HostUnavailable,
		Protocol:            ProtocolUnknown,
		Projection:          ProjectionUnknown,
		Index:               IndexUnknown,
		Storage:             StorageUnknown,
		ProfileID:           domain.ProfileID("123e4567-e89b-42d3-a456-426614174000"),
		ContextKind:         domain.ContextNormal,
		ProtocolVersion:     1,
		RankingModelVersion: "rank-1",
		SchemaVersion:       1,
	}
}

func readySnapshot() Snapshot {
	snapshot := baseSnapshot()
	snapshot.State = StateReady
	snapshot.Host = HostAvailable
	snapshot.Protocol = ProtocolCompatible
	snapshot.Projection = ProjectionCurrent
	snapshot.Index = IndexReady
	snapshot.Storage = StorageHealthy
	snapshot.ProjectionRevision = 7
	snapshot.IndexedRevision = 7
	snapshot.ProjectionSequence = 3
	snapshot.HandshakeAt = time.Unix(100, 0).UTC()
	return snapshot
}

func TestHealthTransitions(t *testing.T) {
	health, err := NewHealth(baseSnapshot(), func() time.Time { return time.Unix(200, 0).UTC() })
	if err != nil {
		t.Fatal(err)
	}
	var observed []Snapshot
	health.Subscribe(ObserverFunc(func(snapshot Snapshot) { observed = append(observed, snapshot) }))

	path := []Snapshot{
		{State: StateConnecting, Host: HostAvailable, Protocol: ProtocolUnknown, Projection: ProjectionUnknown, Index: IndexUnknown, Storage: StorageUnknown},
		{State: StateHandshaking, Host: HostAvailable, Protocol: ProtocolCompatible, Projection: ProjectionUnknown, Index: IndexUnknown, Storage: StorageUnknown},
		{State: StateSynchronizing, Host: HostAvailable, Protocol: ProtocolCompatible, Projection: ProjectionCurrent, Index: IndexBuilding, Storage: StorageHealthy, ProjectionRevision: 7},
		readySnapshot(),
	}
	for _, next := range path {
		next.ProfileID = baseSnapshot().ProfileID
		next.ContextKind = domain.ContextNormal
		next.ProtocolVersion = 1
		next.RankingModelVersion = "rank-1"
		next.SchemaVersion = 1
		if err := health.Transition(next); err != nil {
			t.Fatalf("transition to %s: %v", next.State, err)
		}
	}
	if got := health.Snapshot(); got.State != StateReady || !got.Ready() {
		t.Fatalf("ready snapshot = %+v", got)
	}
	if len(observed) != len(path) {
		t.Fatalf("observed %d transitions, want %d", len(observed), len(path))
	}
	if observed[len(observed)-1].EventSequence != uint64(len(path)) {
		t.Fatalf("event sequence = %d", observed[len(observed)-1].EventSequence)
	}

	before := health.Snapshot()
	illegal := readySnapshot()
	illegal.State = StateConnecting
	if err := health.Transition(illegal); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("illegal transition error = %v", err)
	}
	if after := health.Snapshot(); after != before {
		t.Fatalf("illegal transition changed snapshot: before=%+v after=%+v", before, after)
	}

	degraded := readySnapshot()
	degraded.State = StateReady
	degraded.Storage = StorageDegraded
	if err := health.Transition(degraded); err != nil {
		t.Fatalf("degraded transition: %v", err)
	}
	if got := health.Snapshot(); got.State != StateDegraded || got.Ready() || !got.LexicalAvailable() {
		t.Fatalf("degraded snapshot = %+v", got)
	}
	if err := health.Transition(readySnapshot()); err != nil {
		t.Fatalf("recovery transition: %v", err)
	}
	if got := health.Snapshot(); got.State != StateReady || !got.Ready() {
		t.Fatalf("recovered snapshot = %+v", got)
	}

	incompatible := readySnapshot()
	incompatible.Protocol = ProtocolMismatch
	if err := health.Transition(incompatible); err != nil {
		t.Fatalf("incompatible transition: %v", err)
	}
	if got := health.Snapshot(); got.State != StateIncompatible || got.LexicalAvailable() {
		t.Fatalf("incompatible snapshot = %+v", got)
	}
	stopping := readySnapshot()
	stopping.State = StateStopping
	if err := health.Transition(stopping); err != nil {
		t.Fatalf("stopping transition: %v", err)
	}
	if !health.Snapshot().State.Terminal() {
		t.Fatal("stopping is not terminal")
	}
}

func TestHealthInvariants(t *testing.T) {
	ready := readySnapshot()
	if !ready.Valid() || !ready.Ready() || !ready.LexicalAvailable() {
		t.Fatalf("ready invariants failed: %+v", ready)
	}
	if ready.Normalize() != ready {
		t.Fatalf("ready snapshot changed during normalization: %+v", ready.Normalize())
	}

	degraded := ready
	degraded.Storage = StorageDegraded
	degraded = degraded.Normalize()
	if degraded.State != StateDegraded || degraded.Ready() || !degraded.LexicalAvailable() {
		t.Fatalf("degraded invariants failed: %+v", degraded)
	}

	stale := ready
	stale.IndexedRevision = 6
	stale = stale.Normalize()
	if stale.State != StateRecovering || stale.Ready() || stale.LexicalAvailable() {
		t.Fatalf("stale index invariants failed: %+v", stale)
	}

	mismatch := ready
	mismatch.Protocol = ProtocolMismatch
	mismatch = mismatch.Normalize()
	if mismatch.State != StateIncompatible || mismatch.LexicalAvailable() {
		t.Fatalf("protocol mismatch invariants failed: %+v", mismatch)
	}

	missingHost := ready
	missingHost.Host = HostUnavailable
	missingHost = missingHost.Normalize()
	if missingHost.State != StateDisconnected || missingHost.LexicalAvailable() {
		t.Fatalf("host unavailable invariants failed: %+v", missingHost)
	}

	invalid := ready
	invalid.Index = IndexState("invented")
	if invalid.Valid() {
		t.Fatal("invalid index state accepted")
	}
	if _, err := NewHealth(invalid, nil); !errors.Is(err, ErrInvalidSnapshot) {
		t.Fatalf("invalid snapshot error = %v", err)
	}
}

func TestHealthTransitionTable(t *testing.T) {
	all := []State{
		StateDisconnected, StateConnecting, StateHandshaking, StateSynchronizing,
		StateReady, StateDegraded, StateRecovering, StateIncompatible, StateStopping,
	}
	for _, from := range all {
		for _, to := range all {
			if from == to {
				if !transitionAllowed(from, to) {
					t.Fatalf("%s -> %s should be idempotent", from, to)
				}
				continue
			}
			want := false
			switch from {
			case StateDisconnected:
				want = to == StateConnecting || to == StateIncompatible || to == StateStopping
			case StateConnecting:
				want = to == StateHandshaking || to == StateRecovering || to == StateDisconnected || to == StateIncompatible || to == StateStopping
			case StateHandshaking:
				want = to == StateSynchronizing || to == StateRecovering || to == StateDisconnected || to == StateIncompatible || to == StateStopping
			case StateSynchronizing:
				want = to == StateReady || to == StateRecovering || to == StateDisconnected || to == StateIncompatible || to == StateStopping
			case StateReady:
				want = to == StateDegraded || to == StateRecovering || to == StateDisconnected || to == StateIncompatible || to == StateStopping
			case StateDegraded:
				want = to == StateReady || to == StateRecovering || to == StateDisconnected || to == StateIncompatible || to == StateStopping
			case StateRecovering:
				want = to == StateConnecting || to == StateHandshaking || to == StateSynchronizing || to == StateDisconnected || to == StateIncompatible || to == StateStopping
			case StateIncompatible:
				want = to == StateStopping
			case StateStopping:
				want = false
			}
			if got := transitionAllowed(from, to); got != want {
				t.Fatalf("%s -> %s allowed=%t, want %t", from, to, got, want)
			}
		}
	}
}
