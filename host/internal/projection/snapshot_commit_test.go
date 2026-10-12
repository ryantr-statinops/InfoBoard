package projection

import (
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

const replacementEpoch = "123e4567-e89b-42d3-a456-426614174003"

func snapshotInputFor(t *testing.T, epoch string, sequence uint64, rows ...protocol.TabRecord) SnapshotInput {
	t.Helper()
	reconciler := canonicalTestReconciler(t)
	for index := range rows {
		rows[index].ProjectionEpoch = epoch
		rows[index].ProjectionRevision = 1
	}
	input, rejection := reconciler.BindSnapshot(SnapshotRequest{
		ProfileID:   canonicalTestProfile,
		ContextKind: domain.ContextNormal,
		Epoch:       epoch,
		Revision:    1,
		Sequence:    sequence,
		Records:     rows,
	})
	if rejection != nil {
		t.Fatalf("bind snapshot epoch=%q: %v", epoch, rejection)
	}
	return input
}

func TestState_CommitSnapshotAtomicallyReplacesAndDetachesRecords(t *testing.T) {
	state := newState(Partition{ProfileID: mustCanonicalProfile(t), ContextKind: domain.ContextNormal})
	first := canonicalTestRecord(12)
	first.WindowLabelDisplay = new("Main")
	second := canonicalTestRecord(10)
	input := snapshotInputFor(t, canonicalTestEpoch, 1, first, second)

	outcome, rejection := state.CommitSnapshot(input)
	if rejection != nil || outcome != SnapshotCommitted {
		t.Fatalf("commit initial snapshot outcome=%q rejection=%v", outcome, rejection)
	}
	committed := state.Committed()
	if committed.Status != StatusSnapshotReady || committed.Revision != 1 || committed.Sequence != 1 || committed.Digest == "" || committed.Count() != 2 {
		t.Fatalf("initial committed state = %+v", committed)
	}
	if handoff := state.Handoff(); handoff.Queryable || handoff.Count != 2 {
		t.Fatalf("snapshot-ready handoff queryable=%t count=%d", handoff.Queryable, handoff.Count)
	}
	if duplicate, rejection := state.CommitSnapshot(input); rejection != nil || duplicate != SnapshotDuplicate {
		t.Fatalf("exact snapshot replay outcome=%q rejection=%v", duplicate, rejection)
	}

	*input.Records[1].WindowLabelDisplay = "mutated source"
	if record, ok := state.Committed().Lookup(12); !ok || record.WindowLabelDisplay == nil || *record.WindowLabelDisplay != "Main" {
		t.Fatalf("committed optional label aliases snapshot input: %+v present=%t", record, ok)
	}
	readCopy := state.Committed().Records()
	*readCopy[1].WindowLabelDisplay = "mutated read copy"
	if record, ok := state.Committed().Lookup(12); !ok || record.WindowLabelDisplay == nil || *record.WindowLabelDisplay != "Main" {
		t.Fatalf("committed optional label aliases a returned copy: %+v present=%t", record, ok)
	}

	replacement := snapshotInputFor(t, replacementEpoch, 3, canonicalTestRecord(11))
	replacement.Records[0].TitleDisplay = "tampered after validation"
	if _, rejection := state.CommitSnapshot(replacement); rejection == nil {
		t.Fatal("snapshot whose records no longer match its digest was committed")
	}
	afterFailure := state.Committed()
	if afterFailure.Epoch != canonicalTestEpoch || afterFailure.Revision != 1 || afterFailure.Count() != 2 || afterFailure.Status != StatusResyncRequired {
		t.Fatalf("failed staging changed the previous complete state: %+v", afterFailure)
	}
	if _, ok := afterFailure.Lookup(10); !ok {
		t.Fatal("failed replacement removed a previously committed identity")
	}

	retry := snapshotInputFor(t, replacementEpoch, 4, canonicalTestRecord(11))
	if outcome, rejection := state.CommitSnapshot(retry); rejection != nil || outcome != SnapshotCommitted {
		t.Fatalf("commit authoritative retry outcome=%q rejection=%v", outcome, rejection)
	}
	final := state.Committed()
	if final.Epoch != replacementEpoch || final.Revision != 1 || final.Sequence != 4 || final.Count() != 1 {
		t.Fatalf("replacement committed wrong lineage or count: %+v", final)
	}
	if _, ok := final.Lookup(10); ok {
		t.Fatal("authoritative replacement retained an absent old identity")
	}
	if _, ok := final.Lookup(11); !ok {
		t.Fatal("authoritative replacement omitted the new identity")
	}
}

func mustCanonicalProfile(t *testing.T) domain.ProfileID {
	t.Helper()
	profile, err := domain.ParseProfileID(canonicalTestProfile)
	if err != nil {
		t.Fatal(err)
	}
	return profile
}
func TestReconciler_ApplySnapshotPublishesOnlyAfterCommit(t *testing.T) {
	reconciler := canonicalTestReconciler(t)
	request := SnapshotRequest{
		ProfileID:   canonicalTestProfile,
		ContextKind: domain.ContextNormal,
		Epoch:       canonicalTestEpoch,
		Revision:    1,
		Sequence:    1,
		Records:     []protocol.TabRecord{},
	}
	outcome, handoff, rejection := reconciler.ApplySnapshot(request)
	if rejection != nil || outcome != SnapshotCommitted {
		t.Fatalf("apply empty authoritative snapshot outcome=%q rejection=%v", outcome, rejection)
	}
	if handoff.Status != StatusSnapshotReady || handoff.Count != 0 || handoff.Queryable {
		t.Fatalf("snapshot handoff = %+v", handoff)
	}
	if duplicate, _, rejection := reconciler.ApplySnapshot(request); rejection != nil || duplicate != SnapshotDuplicate {
		t.Fatalf("snapshot replay outcome=%q rejection=%v", duplicate, rejection)
	}
}
