package sync

import (
	"context"
	"fmt"
	"sync"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/projection"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

// Orchestrator coordinates hello->snapshot->index->ready and revision-safe handoff.
// It implements protocol.SyncOwner.
type Orchestrator struct {
	mu          sync.Mutex
	reconciler  *projection.Reconciler
	publisher   IndexPublisher
	degraded    bool
}

func NewOrchestrator(partition projection.Partition, publisher IndexPublisher) (*Orchestrator, error) {
	reconciler, rejection := projection.New(partition)
	if rejection != nil {
		return nil, rejection
	}
	if publisher == nil {
		publisher = NewNoopPublisher()
	}
	return &Orchestrator{reconciler: reconciler, publisher: publisher}, nil
}

// State reports the committed projection status for the protocol layer.
func (o *Orchestrator) State() protocol.SyncState {
	committed := o.reconciler.State().Committed()
	return protocol.SyncState{
		Known:      committed.Revision != 0,
		Epoch:      committed.Epoch,
		Revision:   committed.Revision,
		Sequence:   committed.Sequence,
		Count:      uint64(committed.Count()),
		Rebuilding: o.publisher.Rebuilding(),
	}
}

// ApplySnapshot stages and atomically publishes one authoritative snapshot.
func (o *Orchestrator) ApplySnapshot(ctx context.Context, request protocol.SyncRequest) (protocol.SyncOutcome, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if request.Snapshot == nil {
		return protocol.SyncOutcome{Accepted: false, Reason: protocol.ResyncRevisionMismatch}, nil
	}
	input := projection.SnapshotRequest{
		ProfileID:   string(request.Context.ProfileID),
		ContextKind: request.Context.ContextKind,
		Epoch:       request.Epoch,
		Revision:    request.Revision,
		Sequence:    request.Sequence,
		Records:     request.Snapshot.Records,
	}
	_, handoff, rejection := o.reconciler.ApplySnapshot(input)
	if rejection != nil {
		return protocol.SyncOutcome{
			Accepted: false,
			Revision: rejection.ExpectedRevision,
			Sequence: rejection.ExpectedSequence,
			Reason:   rejection.Resync,
		}, nil
	}
	if err := o.publisher.Publish(ctx, string(handoff.ProfileID), handoff.ContextKind, handoff.Epoch, handoff.Revision, o.records()); err != nil {
		return protocol.SyncOutcome{Accepted: false, Reason: protocol.ResyncRevisionMismatch}, err
	}
	return protocol.SyncOutcome{
		Accepted: true,
		Revision: handoff.Revision,
		Sequence: handoff.Sequence,
		Count:    uint64(handoff.Count),
		Epoch:    handoff.Epoch,
	}, nil
}

func (o *Orchestrator) records() []domain.EligibleTabRecord {
	return o.reconciler.State().Committed().Records()
}

// ApplyDelta is implemented in the ordered-delta task.
func (o *Orchestrator) ApplyDelta(ctx context.Context, request protocol.SyncRequest) (protocol.SyncOutcome, error) {
	return protocol.SyncOutcome{}, fmt.Errorf("delta not implemented")
}

// SetPersistenceDegraded records IP-08 storage health without blocking sync.
func (o *Orchestrator) SetPersistenceDegraded(degraded bool) { o.degraded = degraded }
func (o *Orchestrator) IsPersistenceDegraded() bool         { return o.degraded }
