package sync

import (
	"context"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

func TestQueryGateRefusesUnknownRevision(t *testing.T) {
	orchestrator := newTestOrchestrator(t)
	allowed, code := orchestrator.CheckQuery(1)
	if allowed || code != protocol.CodeRevisionMismatch {
		t.Fatalf("unknown revision allowed: allowed=%v code=%s", allowed, code)
	}
}

func TestQueryGateAllowsCurrentRevision(t *testing.T) {
	orchestrator := newTestOrchestrator(t)
	if _, err := orchestrator.ApplySnapshot(context.Background(), protocol.SyncRequest{
		Context:  protocol.RequestContext{ProfileID: "123e4567-e89b-42d3-a456-426614174000", ContextKind: domain.ContextNormal},
		Epoch:    "123e4567-e89b-42d3-a456-426614174001",
		Revision: 1,
		Sequence: 1,
		Snapshot: &protocol.Snapshot{},
	}); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	allowed, code := orchestrator.CheckQuery(1)
	if !allowed || code != "" {
		t.Fatalf("current revision refused: allowed=%v code=%s", allowed, code)
	}
	if allowed, _ := orchestrator.CheckQuery(2); allowed {
		t.Fatalf("future revision allowed")
	}
}
