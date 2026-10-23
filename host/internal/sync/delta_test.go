package sync

import (
	"context"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

func tabRecord(id int64, epoch string, revision uint64) protocol.TabRecord {
	return protocol.TabRecord{
		ProfileID:          "123e4567-e89b-42d3-a456-426614174000",
		ContextKind:        domain.ContextNormal,
		TabID:              id,
		WindowID:           1,
		TitleDisplay:       "Example",
		TitleSearch:        "example",
		URLDisplay:         "example.test",
		URLSearch:          "example.test/",
		DomainDisplay:      "example.test",
		DomainSearch:       "example.test",
		Eligible:           true,
		ObservedAt:         1,
		ProjectionEpoch:    epoch,
		ProjectionRevision: revision,
	}
}

func TestDeltaAppliesAfterSnapshot(t *testing.T) {
	orchestrator := newTestOrchestrator(t)
	ctx := context.Background()
	const epoch = "123e4567-e89b-42d3-a456-426614174001"

	if _, err := orchestrator.ApplySnapshot(ctx, protocol.SyncRequest{
		Context:  protocol.RequestContext{ProfileID: "123e4567-e89b-42d3-a456-426614174000", ContextKind: domain.ContextNormal},
		Epoch:    epoch,
		Revision: 1,
		Sequence: 1,
		Snapshot: &protocol.Snapshot{Tabs: []protocol.TabRecord{tabRecord(1, epoch, 1)}},
	}); err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	delta := &protocol.Delta{
		BaseRevision:  1,
		SequenceStart: 2,
		SequenceEnd:   2,
		Operations: []protocol.Operation{{
			Sequence: 2,
			Kind:     protocol.OperationCreate,
			TabID:    2,
			Record:   ptrRecord(tabRecord(2, epoch, 2)),
		}},
	}
	outcome, err := orchestrator.ApplyDelta(ctx, protocol.SyncRequest{
		Context:  protocol.RequestContext{ProfileID: "123e4567-e89b-42d3-a456-426614174000", ContextKind: domain.ContextNormal},
		Epoch:    epoch,
		Revision: 2,
		Delta:    delta,
	})
	if err != nil {
		t.Fatalf("delta: %v", err)
	}
	if !outcome.Accepted || outcome.Revision != 2 || outcome.Count != 2 {
		t.Fatalf("unexpected outcome: %+v", outcome)
	}
}

func TestGapDeltaRequestsResync(t *testing.T) {
	orchestrator := newTestOrchestrator(t)
	ctx := context.Background()
	const epoch = "123e4567-e89b-42d3-a456-426614174001"

	if _, err := orchestrator.ApplySnapshot(ctx, protocol.SyncRequest{
		Context:  protocol.RequestContext{ProfileID: "123e4567-e89b-42d3-a456-426614174000", ContextKind: domain.ContextNormal},
		Epoch:    epoch,
		Revision: 1,
		Sequence: 1,
		Snapshot: &protocol.Snapshot{},
	}); err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	outcome, err := orchestrator.ApplyDelta(ctx, protocol.SyncRequest{
		Context:  protocol.RequestContext{ProfileID: "123e4567-e89b-42d3-a456-426614174000", ContextKind: domain.ContextNormal},
		Epoch:    epoch,
		Revision: 2,
		Delta: &protocol.Delta{
			BaseRevision:  1,
			SequenceStart: 5,
			SequenceEnd:   5,
			Operations: []protocol.Operation{{
				Sequence: 5,
				Kind:     protocol.OperationCreate,
				TabID:    2,
				Record:   ptrRecord(tabRecord(2, epoch, 2)),
			}},
		},
	})
	if err != nil {
		t.Fatalf("delta: %v", err)
	}
	if outcome.Accepted || !outcome.Rebuild {
		t.Fatalf("gap accepted: %+v", outcome)
	}
	if state := orchestrator.State(); state.Revision != 1 {
		t.Fatalf("revision advanced on gap: %d", state.Revision)
	}
}

func ptrRecord(record protocol.TabRecord) *protocol.TabRecord { return &record }
