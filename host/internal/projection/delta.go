package projection

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

const maxRememberedDeltaOperations = 1000

// DeltaOutcome is the bounded result of reducing one ordered delta batch.
type DeltaOutcome string

const (
	DeltaApplied   DeltaOutcome = "APPLIED"
	DeltaDuplicate DeltaOutcome = "DUPLICATE"
)

// ApplyDelta stages a complete validated batch against one committed projection.
// Readers observe the previous immutable value until every operation succeeds,
// then see the new canonical state through one pointer replacement.
func (state *State) ApplyDelta(input DeltaInput) (DeltaOutcome, *Rejection) {
	started := time.Now()
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.committed == nil {
		return "", reject(protocol.CodeSnapshotRequired, CauseMissingProjectionRevision, protocol.ResyncRevisionMismatch, 0, input.SequenceStart, -1)
	}
	next, outcome, rejection := reduceDelta(*state.committed, input)
	if rejection != nil {
		if rejection.ResyncRequired() {
			current := *state.committed
			current.Status = StatusResyncRequired
			current.ErrorClass = rejection.Cause
			current.DurationMillis = time.Since(started).Milliseconds()
			if current.Counters.ResyncRequests < ^uint64(0) {
				current.Counters.ResyncRequests++
			}
			if rejection.Code == protocol.CodeRevisionMismatch {
				if current.Counters.EventsStale < ^uint64(0) {
					current.Counters.EventsStale++
				}
			} else if current.Counters.EventsRejected < ^uint64(0) {
				current.Counters.EventsRejected++
			}
			state.committed = &current
		}
		return "", rejection
	}
	next.DurationMillis = time.Since(started).Milliseconds()
	next.ErrorClass = ""
	state.committed = &next
	return outcome, nil
}

func reduceDelta(current Committed, input DeltaInput) (Committed, DeltaOutcome, *Rejection) {
	if input.Partition != current.Partition {
		return Committed{}, "", reject(protocol.CodeProfileMismatch, CauseMismatchedProfileID, protocol.ResyncProfileChanged, current.Revision, input.SequenceStart, -1)
	}
	if input.Epoch != current.Epoch || input.Epoch == "" {
		return Committed{}, "", reject(protocol.CodeSnapshotRequired, CauseMismatchedProjectionEpoch, protocol.ResyncRevisionMismatch, current.Revision, input.SequenceStart, -1)
	}
	if len(input.Operations) == 0 || len(input.Operations) > protocol.MaxDeltaOperations || input.SequenceStart == 0 || input.SequenceEnd < input.SequenceStart || input.SequenceEnd-input.SequenceStart+1 != uint64(len(input.Operations)) {
		return Committed{}, "", reject(protocol.CodeRevisionMismatch, CauseBoundsExceeded, protocol.ResyncSequenceGap, current.Revision, input.SequenceStart, -1)
	}

	signatures := make([]string, len(input.Operations))
	duplicateCount := 0
	for index, operation := range input.Operations {
		wantSequence := input.SequenceStart + uint64(index)
		if operation.Sequence != wantSequence || wantSequence > domain.MaxProjectionRevision {
			return Committed{}, "", reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncSequenceGap, current.Revision, wantSequence, index)
		}
		signature, err := deltaOperationSignature(input.Epoch, operation)
		if err != nil {
			return Committed{}, "", reject(protocol.CodeSnapshotRequired, CauseInvalidRecord, protocol.ResyncSnapshotStale, current.Revision, wantSequence, index)
		}
		signatures[index] = signature
		if previous, exists := current.seen[wantSequence]; exists {
			if previous != signature {
				return Committed{}, "", reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, current.Revision, wantSequence, index)
			}
			duplicateCount++
		}
	}
	if duplicateCount > 0 {
		if duplicateCount != len(input.Operations) {
			return Committed{}, "", reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncSequenceGap, current.Revision, input.SequenceStart, -1)
		}
		if ^uint64(0)-current.Counters.EventsDuplicate < uint64(duplicateCount) {
			return Committed{}, "", reject(protocol.CodeSnapshotRequired, CauseBoundsExceeded, protocol.ResyncSnapshotStale, current.Revision, current.Sequence, -1)
		}
		duplicate := current
		duplicate.Counters.EventsDuplicate += uint64(duplicateCount)
		return duplicate, DeltaDuplicate, nil
	}
	if input.SequenceStart <= current.Sequence {
		return Committed{}, "", reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncSequenceGap, current.Revision, current.Sequence+1, -1)
	}
	if input.BaseRevision != current.Revision || input.Revision > domain.MaxProjectionRevision {
		return Committed{}, "", reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, current.Revision, input.SequenceStart, -1)
	}
	if current.Status != StatusSnapshotReady && current.Status != StatusReady {
		return Committed{}, "", reject(protocol.CodeSnapshotRequired, CauseMissingProjectionRevision, protocol.ResyncRevisionMismatch, current.Revision, input.SequenceStart, -1)
	}

	staged := make(map[domain.TabIdentity]domain.EligibleTabRecord, len(current.records)+len(input.Operations))
	for _, record := range current.records {
		staged[record.TabIdentity] = record
	}
	seen := make(map[uint64]string, len(current.seen)+len(input.Operations))
	for sequence, signature := range current.seen {
		seen[sequence] = signature
	}
	revision := current.Revision
	accepted := uint64(0)
	for index, operation := range input.Operations {
		sequence := operation.Sequence
		identity := operation.Identity
		if identity.ProfileID != current.Partition.ProfileID || identity.ContextKind != current.Partition.ContextKind || identity.TabID <= 0 || uint64(identity.TabID) > domain.MaxProjectionRevision {
			return Committed{}, "", reject(protocol.CodeProfileMismatch, CauseMismatchedTabIdentity, protocol.ResyncProfileChanged, revision, sequence, index)
		}
		existing, exists := staged[identity]
		if operation.Kind == protocol.OperationRemove {
			if operation.Record != nil || !exists {
				return Committed{}, "", reject(protocol.CodeSnapshotRequired, CauseUnexpectedRecord, protocol.ResyncSnapshotStale, revision, sequence, index)
			}
			if revision >= domain.MaxProjectionRevision {
				return Committed{}, "", reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, revision, sequence, index)
			}
			delete(staged, identity)
			revision++
		} else {
			if !operation.Kind.KnownOperation() || operation.Record == nil {
				return Committed{}, "", reject(protocol.CodeInvalidFrame, CauseMissingRecord, protocol.ResyncSnapshotStale, revision, sequence, index)
			}
			record := *operation.Record
			if record.TabIdentity != identity || !record.Valid(current.Partition.ProfileID, current.Partition.ContextKind, current.Epoch) {
				return Committed{}, "", reject(protocol.CodeSnapshotRequired, CauseInvalidRecord, protocol.ResyncSnapshotStale, revision, sequence, index)
			}
			if operation.Kind == protocol.OperationCreate && exists || operation.Kind != protocol.OperationCreate && !exists {
				return Committed{}, "", reject(protocol.CodeSnapshotRequired, CauseUnexpectedRecord, protocol.ResyncSnapshotStale, revision, sequence, index)
			}
			if exists && recordsEqual(existing, record) {
				if record.ProjectionRevision != revision {
					return Committed{}, "", reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, revision, sequence, index)
				}
			} else {
				if revision >= domain.MaxProjectionRevision || record.ProjectionRevision != revision+1 {
					return Committed{}, "", reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, revision, sequence, index)
				}
				staged[identity] = record
				revision++
			}
		}
		seen[sequence] = signatures[index]
		accepted++
	}
	if revision != input.Revision {
		return Committed{}, "", reject(protocol.CodeRevisionMismatch, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, revision, input.SequenceEnd, -1)
	}
	if ^uint64(0)-current.Counters.EventsAccepted < accepted {
		return Committed{}, "", reject(protocol.CodeSnapshotRequired, CauseBoundsExceeded, protocol.ResyncSnapshotStale, revision, input.SequenceEnd, -1)
	}
	oldestRetained := uint64(0)
	if input.SequenceEnd > maxRememberedDeltaOperations {
		oldestRetained = input.SequenceEnd - maxRememberedDeltaOperations
	}
	for sequence := range seen {
		if sequence < oldestRetained {
			delete(seen, sequence)
		}
	}
	next := Committed{
		Partition: current.Partition, Status: current.Status, Epoch: current.Epoch,
		Revision: revision, Sequence: input.SequenceEnd,
		DigestVersion: current.DigestVersion, Counters: current.Counters,
		records: make([]domain.EligibleTabRecord, 0, len(staged)), seen: seen,
	}
	next.Counters.EventsAccepted += accepted
	for _, record := range staged {
		next.records = append(next.records, record)
	}
	sortCommitted(next.records)
	digest, err := canonicalSnapshotDigest(next.records)
	if err != nil {
		return Committed{}, "", reject(protocol.CodeSnapshotRequired, CauseInvalidRecord, protocol.ResyncSnapshotStale, revision, input.SequenceEnd, -1)
	}
	next.Digest = digest
	next.DigestVersion = ProjectionDigestVersion
	return next, DeltaApplied, nil
}

func deltaOperationSignature(epoch string, operation DeltaOperation) (string, error) {
	encoded, err := json.Marshal(struct {
		Epoch    string
		Sequence uint64
		Kind     protocol.OperationKind
		Identity domain.TabIdentity
		Record   *domain.EligibleTabRecord
	}{epoch, operation.Sequence, operation.Kind, operation.Identity, operation.Record})
	return string(encoded), err
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
