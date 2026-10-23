package sync

import (
	"context"
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

// ApplyDelta stages one ordered batch and acknowledges it only after the
// projection and index publish the same revision. Duplicates are idempotent.
func (o *Orchestrator) ApplyDelta(ctx context.Context, request protocol.SyncRequest) (protocol.SyncOutcome, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if request.Delta == nil {
		return protocol.SyncOutcome{Accepted: false, Reason: protocol.ResyncRevisionMismatch}, nil
	}
	input, rejection := o.reconciler.BindDelta(projection.DeltaRequest{
		ProfileID:   string(request.Context.ProfileID),
		ContextKind: request.Context.ContextKind,
		Epoch:       request.Epoch,
		Revision:    request.Revision,
		Delta:       *request.Delta,
	})
	if rejection != nil {
		return protocol.SyncOutcome{
			Accepted: false,
			Revision: rejection.ExpectedRevision,
			Sequence: rejection.ExpectedSequence,
			Reason:   rejection.Resync,
			Rebuild:  rejection.ResyncRequired(),
		}, nil
	}
	outcome, rejection := o.reconciler.State().ApplyDelta(input)
	if rejection != nil {
		return protocol.SyncOutcome{
			Accepted: false,
			Revision: rejection.ExpectedRevision,
			Sequence: rejection.ExpectedSequence,
			Reason:   rejection.Resync,
			Rebuild:  rejection.ResyncRequired(),
		}, nil
	}
	handoff := o.reconciler.State().Handoff()
	if outcome == projection.DeltaApplied {
		if err := o.publisher.Publish(ctx, string(handoff.ProfileID), handoff.ContextKind, handoff.Epoch, handoff.Revision, o.records()); err != nil {
			return protocol.SyncOutcome{Accepted: false, Reason: protocol.ResyncRevisionMismatch}, err
		}
	}
	return protocol.SyncOutcome{
		Accepted: true,
		Revision: handoff.Revision,
		Sequence: handoff.Sequence,
		Count:    uint64(handoff.Count),
		Epoch:    handoff.Epoch,
	}, nil
}

// SetPersistenceDegraded records IP-08 storage health without blocking sync.
func (o *Orchestrator) SetPersistenceDegraded(degraded bool) { o.degraded = degraded }
func (o *Orchestrator) IsPersistenceDegraded() bool         { return o.degraded }

// CheckQuery enforces the revision fence for a query handoff. It never falls back
// to an older ready revision: unknown/old revisions are REVISION_MISMATCH and a
// staging index is INDEX_REBUILDING, both without a result payload.
func (o *Orchestrator) CheckQuery(revision uint64) (bool, protocol.ErrorCode) {
	state := o.State()
	if state.Rebuilding {
		return false, protocol.CodeIndexRebuilding
	}
	if !state.Known || revision != state.Revision {
		return false, protocol.CodeRevisionMismatch
	}
	return true, ""
}

// PersistenceHealth is the bounded, redacted health view exposed to IP-12/IP-16.
type PersistenceHealth struct {
	Degraded  bool
	Revision  uint64
	Queryable bool
}

// PersistenceHealth reports storage posture without ever exposing tab fields.
func (o *Orchestrator) PersistenceHealth() PersistenceHealth {
	state := o.State()
	return PersistenceHealth{
		Degraded:  o.degraded,
		Revision:  state.Revision,
		Queryable: state.Known && !state.Rebuilding,
	}
}
