package projection

import (
	"sort"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

// ApplyDelta stages a complete validated batch against one committed projection.
// Readers observe the previous immutable value until every operation succeeds,
// then see the new canonical state through one pointer replacement.
func (state *State) ApplyDelta(input DeltaInput) *Rejection {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.committed == nil {
		return reject(protocol.CodeSnapshotRequired, CauseMissingProjectionRevision, protocol.ResyncRevisionMismatch, 0, input.SequenceStart, -1)
	}
	next, rejection := reduceDelta(*state.committed, input)
	if rejection != nil {
		return rejection
	}
	state.committed = &next
	return nil
}

func reduceDelta(current Committed, input DeltaInput) (Committed, *Rejection) {
	if input.Partition != current.Partition {
		return Committed{}, reject(protocol.CodeProfileMismatch, CauseMismatchedProfileID, protocol.ResyncProfileChanged, current.Revision, input.SequenceStart, -1)
	}
	if current.Status != StatusSnapshotReady && current.Status != StatusReady {
		return Committed{}, reject(protocol.CodeSnapshotRequired, CauseMissingProjectionRevision, protocol.ResyncRevisionMismatch, current.Revision, input.SequenceStart, -1)
	}
	if input.Epoch != current.Epoch || input.Epoch == "" {
		return Committed{}, reject(protocol.CodeSnapshotRequired, CauseMismatchedProjectionEpoch, protocol.ResyncRevisionMismatch, current.Revision, input.SequenceStart, -1)
	}
	if input.BaseRevision != current.Revision || input.Revision > domain.MaxProjectionRevision {
		return Committed{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, current.Revision, input.SequenceStart, -1)
	}
	if len(input.Operations) == 0 || len(input.Operations) > protocol.MaxDeltaOperations || input.SequenceStart == 0 || input.SequenceEnd < input.SequenceStart || input.SequenceEnd-input.SequenceStart+1 != uint64(len(input.Operations)) {
		return Committed{}, reject(protocol.CodeRevisionMismatch, CauseBoundsExceeded, protocol.ResyncSequenceGap, current.Revision, input.SequenceStart, -1)
	}
	if input.SequenceStart <= current.Sequence {
		return Committed{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncSequenceGap, current.Revision, current.Sequence+1, -1)
	}

	staged := make(map[domain.TabIdentity]domain.EligibleTabRecord, len(current.records)+len(input.Operations))
	for _, record := range current.records {
		staged[record.TabIdentity] = record
	}
	revision := current.Revision
	sequence := current.Sequence
	accepted := uint64(0)
	for index, operation := range input.Operations {
		wantSequence := input.SequenceStart + uint64(index)
		if operation.Sequence != wantSequence || wantSequence > domain.MaxProjectionRevision {
			return Committed{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncSequenceGap, revision, wantSequence, index)
		}
		if !operation.Kind.KnownOperation() {
			return Committed{}, reject(protocol.CodeInvalidFrame, CauseUnknownOperation, "", revision, wantSequence, index)
		}
		identity := operation.Identity
		if identity.ProfileID != current.Partition.ProfileID || identity.ContextKind != current.Partition.ContextKind || identity.TabID <= 0 || uint64(identity.TabID) > domain.MaxProjectionRevision {
			return Committed{}, reject(protocol.CodeProfileMismatch, CauseMismatchedTabIdentity, protocol.ResyncProfileChanged, revision, wantSequence, index)
		}
		existing, exists := staged[identity]
		if operation.Kind == protocol.OperationRemove {
			if operation.Record != nil || !exists {
				return Committed{}, reject(protocol.CodeSnapshotRequired, CauseUnexpectedRecord, protocol.ResyncSnapshotStale, revision, wantSequence, index)
			}
			delete(staged, identity)
			revision++
			sequence = wantSequence
			accepted++
			continue
		}
		if operation.Record == nil {
			return Committed{}, reject(protocol.CodeSnapshotRequired, CauseMissingRecord, protocol.ResyncSnapshotStale, revision, wantSequence, index)
		}
		record := *operation.Record
		if record.TabIdentity != identity || !record.Valid(current.Partition.ProfileID, current.Partition.ContextKind, current.Epoch) {
			return Committed{}, reject(protocol.CodeSnapshotRequired, CauseInvalidRecord, protocol.ResyncSnapshotStale, revision, wantSequence, index)
		}
		if operation.Kind == protocol.OperationCreate && exists {
			return Committed{}, reject(protocol.CodeSnapshotRequired, CauseUnexpectedRecord, protocol.ResyncSnapshotStale, revision, wantSequence, index)
		}
		if operation.Kind != protocol.OperationCreate && !exists {
			return Committed{}, reject(protocol.CodeSnapshotRequired, CauseUnexpectedRecord, protocol.ResyncSnapshotStale, revision, wantSequence, index)
		}
		if exists && recordsEqual(existing, record) {
			if record.ProjectionRevision != revision {
				return Committed{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, revision, wantSequence, index)
			}
			sequence = wantSequence
			accepted++
			continue
		}
		if revision >= domain.MaxProjectionRevision || record.ProjectionRevision != revision+1 {
			return Committed{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, revision, wantSequence, index)
		}
		staged[identity] = record
		revision++
		sequence = wantSequence
		accepted++
	}
	if revision != input.Revision || sequence != input.SequenceEnd {
		return Committed{}, reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, revision, sequence, -1)
	}
	if ^uint64(0)-current.Counters.EventsAccepted < accepted {
		return Committed{}, reject(protocol.CodeSnapshotRequired, CauseBoundsExceeded, protocol.ResyncSnapshotStale, revision, sequence, -1)
	}
	next := Committed{
		Partition: current.Partition,
		Status:    current.Status,
		Epoch:     current.Epoch,
		Revision:  revision,
		Sequence:  sequence,
		Counters:  current.Counters,
		records:   make([]domain.EligibleTabRecord, 0, len(staged)),
	}
	next.Counters.EventsAccepted += accepted
	for _, record := range staged {
		next.records = append(next.records, record)
	}
	sortCommitted(next.records)
	return next, nil
}

func recordsEqual(left, right domain.EligibleTabRecord) bool {
	return left.ProfileID == right.ProfileID && left.ContextKind == right.ContextKind && left.TabIdentity == right.TabIdentity &&
		left.WindowID == right.WindowID && equalInt64Pointer(left.GroupID, right.GroupID) &&
		left.TitleDisplay == right.TitleDisplay && left.TitleSearch == right.TitleSearch && left.URLSearch == right.URLSearch &&
		left.URLDisplay == right.URLDisplay && left.DomainDisplay == right.DomainDisplay && left.DomainSearch == right.DomainSearch &&
		equalStringPointer(left.WindowLabelDisplay, right.WindowLabelDisplay) && equalStringPointer(left.WindowLabelSearch, right.WindowLabelSearch) &&
		equalStringPointer(left.GroupLabelDisplay, right.GroupLabelDisplay) && equalStringPointer(left.GroupLabelSearch, right.GroupLabelSearch) &&
		left.Pinned == right.Pinned && left.Active == right.Active && left.Eligible == right.Eligible &&
		left.ObservedAt == right.ObservedAt && left.ProjectionEpoch == right.ProjectionEpoch && left.ProjectionRevision == right.ProjectionRevision
}

func equalInt64Pointer(left, right *int64) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func equalStringPointer(left, right *string) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func sortCommitted(records []domain.EligibleTabRecord) {
	sort.Slice(records, func(i, j int) bool { return identityLess(records[i].TabIdentity, records[j].TabIdentity) })
}
