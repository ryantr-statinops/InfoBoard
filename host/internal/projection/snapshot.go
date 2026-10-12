package projection

import (
	"sort"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

// SnapshotCommitOutcome is the bounded result of publishing one staged snapshot.
type SnapshotCommitOutcome string

const (
	SnapshotCommitted SnapshotCommitOutcome = "COMMITTED"
	SnapshotDuplicate SnapshotCommitOutcome = "DUPLICATE"
)

// CommitSnapshot validates a detached snapshot candidate completely before one
// immutable committed-state pointer replaces the current view. A failed candidate
// leaves the previous records and lineage intact and removes query readiness.
func (state *State) CommitSnapshot(input SnapshotInput) (SnapshotCommitOutcome, *Rejection) {
	state.mu.Lock()
	defer state.mu.Unlock()

	rejectCandidate := func(code protocol.ErrorCode, cause Cause, reason protocol.ResyncReason, index int) (SnapshotCommitOutcome, *Rejection) {
		if state.committed != nil && input.Partition == state.partition {
			current := *state.committed
			current.Status = StatusResyncRequired
			if current.Counters.SnapshotsRejected < ^uint64(0) {
				current.Counters.SnapshotsRejected++
			}
			if current.Counters.ResyncRequests < ^uint64(0) {
				current.Counters.ResyncRequests++
			}
			state.committed = &current
		}
		return "", reject(code, cause, reason, 1, input.Sequence, index)
	}

	if input.Partition != state.partition || !state.partition.Valid() {
		return rejectCandidate(protocol.CodeProfileMismatch, CauseMismatchedProfileID, protocol.ResyncProfileChanged, -1)
	}
	if !validProjectionEpoch(input.Epoch) {
		return rejectCandidate(protocol.CodeSnapshotRequired, CauseMissingProjectionEpoch, protocol.ResyncRevisionMismatch, -1)
	}
	if input.Revision != 1 || input.Sequence == 0 || input.Sequence > domain.MaxProjectionRevision {
		return rejectCandidate(protocol.CodeSnapshotRequired, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, -1)
	}
	if len(input.Records) > protocol.MaxSnapshotRecords {
		return rejectCandidate(protocol.CodePayloadLimit, CauseBoundsExceeded, protocol.ResyncSnapshotStale, -1)
	}

	candidateRecords := make([]domain.EligibleTabRecord, 0, len(input.Records))
	seen := make(map[domain.TabIdentity]struct{}, len(input.Records))
	for index, record := range input.Records {
		if !record.Valid(state.partition.ProfileID, state.partition.ContextKind, input.Epoch) || record.TabIdentity.TabID <= 0 || record.ProjectionRevision != 1 {
			return rejectCandidate(protocol.CodeSnapshotRequired, CauseInvalidRecord, protocol.ResyncSnapshotStale, index)
		}
		if _, exists := seen[record.TabIdentity]; exists {
			return rejectCandidate(protocol.CodeSnapshotRequired, CauseDuplicateTabIdentity, protocol.ResyncSnapshotStale, index)
		}
		seen[record.TabIdentity] = struct{}{}
		candidateRecords = append(candidateRecords, cloneEligibleRecord(record))
	}
	sort.Slice(candidateRecords, func(i, j int) bool {
		return identityLess(candidateRecords[i].TabIdentity, candidateRecords[j].TabIdentity)
	})
	digest, err := canonicalSnapshotDigest(candidateRecords)
	if err != nil || digest != input.Digest {
		return rejectCandidate(protocol.CodeSnapshotRequired, CauseInvalidRecord, protocol.ResyncSnapshotStale, -1)
	}
	if state.committed != nil && state.committed.Epoch == input.Epoch {
		current := state.committed
		if current.Revision == input.Revision && current.Sequence == input.Sequence && current.Digest == digest && snapshotRecordsEqual(current.records, candidateRecords) {
			return SnapshotDuplicate, nil
		}
		return rejectCandidate(protocol.CodeSnapshotRequired, CauseMismatchedProjectionRevision, protocol.ResyncRevisionMismatch, -1)
	}
	if state.committed != nil && state.committed.Counters.SnapshotsCommitted == ^uint64(0) {
		return rejectCandidate(protocol.CodeSnapshotRequired, CauseBoundsExceeded, protocol.ResyncSnapshotStale, -1)
	}

	counters := Counters{}
	if state.committed != nil {
		counters = state.committed.Counters
	}
	counters.SnapshotsCommitted++
	state.committed = &Committed{
		Partition: state.partition,
		Status:    StatusSnapshotReady,
		Epoch:     input.Epoch,
		Revision:  1,
		Sequence:  input.Sequence,
		Digest:    digest,
		Counters:  counters,
		records:   candidateRecords,
		seen:      make(map[uint64]string),
	}
	return SnapshotCommitted, nil
}

func validProjectionEpoch(epoch string) bool {
	_, err := domain.ParseProfileID(epoch)
	return err == nil && len(epoch) <= protocol.MaxEpochBytes
}
func cloneEligibleRecord(record domain.EligibleTabRecord) domain.EligibleTabRecord {
	cloned := record
	if record.GroupID != nil {
		cloned.GroupID = new(*record.GroupID)
	}
	cloned.WindowLabelDisplay = cloneString(record.WindowLabelDisplay)
	cloned.WindowLabelSearch = cloneString(record.WindowLabelSearch)
	cloned.GroupLabelDisplay = cloneString(record.GroupLabelDisplay)
	cloned.GroupLabelSearch = cloneString(record.GroupLabelSearch)
	return cloned
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	return new(*value)
}

func snapshotRecordsEqual(left, right []domain.EligibleTabRecord) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !recordsEqual(left[index], right[index]) {
			return false
		}
	}
	return true
}

// ApplySnapshot binds and atomically publishes one authoritative snapshot. The
// returned handoff is produced only after the candidate has become committed.
func (reconciler *Reconciler) ApplySnapshot(request SnapshotRequest) (SnapshotCommitOutcome, Handoff, *Rejection) {
	input, rejection := reconciler.BindSnapshot(request)
	if rejection != nil {
		return "", Handoff{}, rejection
	}
	outcome, rejection := reconciler.state.CommitSnapshot(input)
	if rejection != nil {
		return "", Handoff{}, rejection
	}
	return outcome, reconciler.state.Handoff(), nil
}
