package sync

import (
	"context"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

func TestSnapshotContinuesWhenPersistenceDegraded(t *testing.T) {
	orchestrator := newTestOrchestrator(t)
	orchestrator.SetPersistenceDegraded(true)

	outcome, err := orchestrator.ApplySnapshot(context.Background(), protocol.SyncRequest{
		Context:  protocol.RequestContext{ProfileID: "123e4567-e89b-42d3-a456-426614174000", ContextKind: domain.ContextNormal},
		Epoch:    "123e4567-e89b-42d3-a456-426614174001",
		Revision: 1,
		Sequence: 1,
		Snapshot: &protocol.Snapshot{},
	})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if !outcome.Accepted {
		t.Fatalf("degraded persistence blocked sync: %+v", outcome)
	}
	health := orchestrator.PersistenceHealth()
	if !health.Degraded || !health.Queryable || health.Revision != 1 {
		t.Fatalf("unexpected health: %+v", health)
	}
}
