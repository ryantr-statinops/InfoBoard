package projection

import (
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

func TestReconciler_IsolatesProfileAndContextPartitions(t *testing.T) {
	const otherProfile = "123e4567-e89b-42d3-a456-426614174099"
	profileIDs := []string{canonicalTestProfile, otherProfile}
	contexts := []domain.ContextKind{domain.ContextNormal, domain.ContextPrivate}
	states := make(map[Partition]*Reconciler, len(profileIDs)*len(contexts))

	for _, profileID := range profileIDs {
		parsedProfile, err := domain.ParseProfileID(profileID)
		if err != nil {
			t.Fatal(err)
		}
		for _, contextKind := range contexts {
			partition := Partition{ProfileID: parsedProfile, ContextKind: contextKind}
			reconciler, rejection := New(partition)
			if rejection != nil {
				t.Fatalf("new partition %+v: %v", partition, rejection)
			}
			record := canonicalTestRecord(10)
			record.ProfileID = profileID
			record.ContextKind = contextKind
			record.ProjectionEpoch = canonicalTestEpoch
			record.ProjectionRevision = 1
			request := SnapshotRequest{
				ProfileID: profileID, ContextKind: contextKind, Epoch: canonicalTestEpoch,
				Revision: 1, Sequence: 1, Records: []protocol.TabRecord{record},
			}
			if outcome, handoff, failure := reconciler.ApplySnapshot(request); failure != nil || outcome != SnapshotCommitted {
				t.Fatalf("commit partition %+v outcome=%q rejection=%v", partition, outcome, failure)
			} else if handoff.ProfileID != parsedProfile || handoff.ContextKind != contextKind || handoff.Count != 1 || len(handoff.Identities) != 1 || handoff.Identities[0] != partition.Identity(10) {
				t.Fatalf("handoff escaped partition %+v: %+v", partition, handoff)
			}
			committed := reconciler.State().Committed()
			stored, ok := committed.Lookup(10)
			if !ok || stored.ProfileID != parsedProfile || stored.ContextKind != contextKind || stored.TabIdentity != partition.Identity(10) {
				t.Fatalf("same numeric tab id collided in partition %+v: record=%+v present=%t", partition, stored, ok)
			}
			if committed.Counters.SnapshotsCommitted != 1 {
				t.Fatalf("snapshot counter leaked across partition %+v: %+v", partition, committed.Counters)
			}
			states[partition] = reconciler
		}
	}

	for partition, reconciler := range states {
		if handoff := reconciler.State().Handoff(); handoff.ProfileID != partition.ProfileID || handoff.ContextKind != partition.ContextKind || handoff.Count != 1 || handoff.Identities[0] != partition.Identity(10) {
			t.Fatalf("partition handoff was overwritten by another profile/context: partition=%+v handoff=%+v", partition, handoff)
		}
	}

	bound := states[Partition{ProfileID: mustCanonicalProfile(t), ContextKind: domain.ContextNormal}]
	before := bound.State().Committed()
	mismatch := SnapshotRequest{
		ProfileID: otherProfile, ContextKind: domain.ContextNormal, Epoch: canonicalTestEpoch,
		Revision: 1, Sequence: 2, Records: []protocol.TabRecord{},
	}
	if _, rejection := bound.BindSnapshot(mismatch); rejection == nil || rejection.Cause != CauseMismatchedProfileID {
		t.Fatalf("mismatched profile snapshot was not refused: %v", rejection)
	}
	after := bound.State().Committed()
	if after.Epoch != before.Epoch || after.Revision != before.Revision || after.Count() != before.Count() || after.Counters != before.Counters {
		t.Fatalf("mismatched profile mutated bound partition: before=%+v after=%+v", before, after)
	}
}
