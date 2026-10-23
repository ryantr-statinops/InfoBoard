package sync

import (
	"context"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/projection"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

func newTestOrchestrator(t *testing.T) *Orchestrator {
	t.Helper()
	profile := domain.ProfileID("123e4567-e89b-42d3-a456-426614174000")
	partition := projection.Partition{ProfileID: profile, ContextKind: domain.ContextNormal}
	orchestrator, err := NewOrchestrator(partition, NewNoopPublisher())
	if err != nil {
		t.Fatalf("new orchestrator: %v", err)
	}
	return orchestrator
}

func TestSnapshotCommitsOneGeneration(t *testing.T) {
	orchestrator := newTestOrchestrator(t)
	outcome, err := orchestrator.ApplySnapshot(context.Background(), protocol.SyncRequest{
		Context:  protocol.RequestContext{ProfileID: "123e4567-e89b-42d3-a456-426614174000", ContextKind: domain.ContextNormal},
		Epoch:    "123e4567-e89b-42d3-a456-426614174001",
		Revision: 1,
		Sequence: 1,
		Snapshot: &protocol.Snapshot{},
	})
	if err != nil {
		t.Fatalf("apply snapshot: %v", err)
	}
	if !outcome.Accepted || outcome.Revision != 1 {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
}

func TestInvalidSnapshotKeepsRevision(t *testing.T) {
	orchestrator := newTestOrchestrator(t)
	outcome, err := orchestrator.ApplySnapshot(context.Background(), protocol.SyncRequest{
		Context:  protocol.RequestContext{ProfileID: "123e4567-e89b-42d3-a456-426614174000", ContextKind: domain.ContextNormal},
		Epoch:    "123e4567-e89b-42d3-a456-426614174001",
		Revision: 2,
		Sequence: 1,
		Snapshot: &protocol.Snapshot{},
	})
	if err != nil {
		t.Fatalf("apply snapshot: %v", err)
	}
	if outcome.Accepted {
		t.Fatalf("invalid snapshot was accepted")
	}
	if state := orchestrator.State(); state.Revision != 0 {
		t.Fatalf("revision advanced on rejection: %d", state.Revision)
	}
}
