package projection

import (
	"reflect"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

func deltaTestRecord(tabID int64, revision uint64, title string) domain.EligibleTabRecord {
	profile, _ := domain.ParseProfileID(canonicalTestProfile)
	return domain.EligibleTabRecord{
		ProfileID:          profile,
		ContextKind:        domain.ContextNormal,
		TabIdentity:        domain.TabIdentity{ProfileID: profile, ContextKind: domain.ContextNormal, TabID: tabID},
		WindowID:           1,
		TitleDisplay:       title,
		TitleSearch:        title,
		URLSearch:          "https://example.test/",
		URLDisplay:         "https://example.test/",
		DomainDisplay:      "example.test",
		DomainSearch:       "example.test",
		Eligible:           true,
		ObservedAt:         1,
		ProjectionEpoch:    canonicalTestEpoch,
		ProjectionRevision: revision,
	}
}

func deltaTestState() *State {
	profile, _ := domain.ParseProfileID(canonicalTestProfile)
	partition := Partition{ProfileID: profile, ContextKind: domain.ContextNormal}
	state := newState(partition)
	state.committed = &Committed{
		Partition: partition,
		Status:    StatusReady,
		Epoch:     canonicalTestEpoch,
		Revision:  1,
		Sequence:  3,
		Counters:  Counters{},
		records:   []domain.EligibleTabRecord{deltaTestRecord(10, 1, "Ten"), deltaTestRecord(11, 1, "Eleven")},
	}
	return state
}

func TestState_ApplyDeltaStagesEffectiveAndNoopOperations(t *testing.T) {
	state := deltaTestState()
	first := deltaTestRecord(10, 1, "Ten")
	updated := deltaTestRecord(11, 2, "Eleven updated")
	input := DeltaInput{
		Partition:     state.partition,
		Epoch:         canonicalTestEpoch,
		BaseRevision:  1,
		Revision:      3,
		SequenceStart: 4,
		SequenceEnd:   6,
		Operations: []DeltaOperation{
			{Sequence: 4, Kind: protocol.OperationUpdate, Identity: first.TabIdentity, Record: &first},
			{Sequence: 5, Kind: protocol.OperationUpdate, Identity: updated.TabIdentity, Record: &updated},
			{Sequence: 6, Kind: protocol.OperationRemove, Identity: first.TabIdentity},
		},
	}
	if rejection := state.ApplyDelta(input); rejection != nil {
		t.Fatalf("apply ordered delta: %v", rejection)
	}
	committed := state.Committed()
	if committed.Revision != 3 || committed.Sequence != 6 || committed.Status != StatusReady || committed.Count() != 1 {
		t.Fatalf("committed revision=%d sequence=%d status=%s count=%d", committed.Revision, committed.Sequence, committed.Status, committed.Count())
	}
	if record, ok := committed.Lookup(11); !ok || record.TitleDisplay != "Eleven updated" || record.ProjectionRevision != 2 {
		t.Fatalf("updated record = %+v, present=%t", record, ok)
	}
	if _, ok := committed.Lookup(10); ok {
		t.Fatal("removed identity remained in the committed projection")
	}
}

func TestState_ApplyDeltaRejectsInvalidMemberWithoutPartialPublication(t *testing.T) {
	state := deltaTestState()
	before := state.Committed()
	valid := deltaTestRecord(11, 2, "Eleven updated")
	unknown := deltaTestRecord(12, 3, "Unknown")
	input := DeltaInput{
		Partition:     state.partition,
		Epoch:         canonicalTestEpoch,
		BaseRevision:  1,
		Revision:      3,
		SequenceStart: 4,
		SequenceEnd:   5,
		Operations: []DeltaOperation{
			{Sequence: 4, Kind: protocol.OperationUpdate, Identity: valid.TabIdentity, Record: &valid},
			{Sequence: 5, Kind: protocol.OperationUpdate, Identity: unknown.TabIdentity, Record: &unknown},
		},
	}
	if rejection := state.ApplyDelta(input); rejection == nil {
		t.Fatal("batch containing an update for an unknown tab was accepted")
	}
	after := state.Committed()
	if after.Revision != before.Revision || after.Sequence != before.Sequence || !reflect.DeepEqual(after.records, before.records) {
		t.Fatalf("rejected batch partially changed committed state: before=%+v after=%+v", before, after)
	}
}
func TestBindDeltaPreservesRecordRevisionAndSequence(t *testing.T) {
	reconciler := canonicalTestReconciler(t)
	noop := canonicalTestRecord(10)
	updated := canonicalTestRecord(11)
	updated.ProjectionRevision = 2
	delta, rejection := reconciler.BindDelta(DeltaRequest{
		ProfileID:   canonicalTestProfile,
		ContextKind: domain.ContextNormal,
		Epoch:       canonicalTestEpoch,
		Revision:    2,
		Delta: protocol.Delta{
			BaseRevision:  1,
			SequenceStart: 4,
			SequenceEnd:   5,
			Operations: []protocol.Operation{
				{Sequence: 4, Kind: protocol.OperationUpdate, TabID: 10, Record: &noop},
				{Sequence: 5, Kind: protocol.OperationUpdate, TabID: 11, Record: &updated},
			},
		},
	})
	if rejection != nil {
		t.Fatalf("bind mixed no-op/effective delta: %v", rejection)
	}
	if delta.BaseRevision != 1 || delta.Revision != 2 || delta.SequenceStart != 4 || delta.SequenceEnd != 5 || len(delta.Operations) != 2 {
		t.Fatalf("delta fence was rewritten: %+v", delta)
	}
	if delta.Operations[0].Record == nil || delta.Operations[0].Record.ProjectionRevision != 1 || delta.Operations[1].Record == nil || delta.Operations[1].Record.ProjectionRevision != 2 {
		t.Fatalf("operation record revisions were renumbered: %+v", delta.Operations)
	}
}
