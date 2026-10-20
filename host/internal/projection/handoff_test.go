package projection

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

func TestHandoff_ExposesVersionedDigestAndBoundedDiagnosticsOnly(t *testing.T) {
	reconciler := canonicalTestReconciler(t)
	record := canonicalTestRecord(10)
	record.TitleDisplay = "PRIVATE_TITLE_SENTINEL"
	record.TitleSearch = "PRIVATE_QUERY_SENTINEL"
	record.URLDisplay = "https://private.invalid/PRIVATE_PATH_SENTINEL"
	record.URLSearch = "private.invalid/private_path_sentinel"
	request := SnapshotRequest{
		ProfileID: canonicalTestProfile, ContextKind: domain.ContextNormal, Epoch: canonicalTestEpoch,
		Revision: 1, Sequence: 1, Records: []protocol.TabRecord{record},
	}
	if outcome, _, rejection := reconciler.ApplySnapshot(request); rejection != nil || outcome != SnapshotCommitted {
		t.Fatalf("commit safe-handoff fixture outcome=%q rejection=%v", outcome, rejection)
	}
	initial := reconciler.State().Handoff()
	assertSafeHandoffMetadata(t, initial, 1, 1)
	if initial.Digest == "" || len(initial.Digest) != 64 || initial.DigestVersion != ProjectionDigestVersion || initial.ErrorClass != "" {
		t.Fatalf("initial handoff metadata = %+v", initial)
	}
	assertNoTabContent(t, initial)

	updated := record
	updated.TitleDisplay = "UPDATED_TITLE_SENTINEL"
	updated.TitleSearch = "updated_query_sentinel"
	updated.ProjectionRevision = 2
	delta, rejection := reconciler.BindDelta(DeltaRequest{
		ProfileID: canonicalTestProfile, ContextKind: domain.ContextNormal, Epoch: canonicalTestEpoch, Revision: 2,
		Delta: protocol.Delta{BaseRevision: 1, SequenceStart: 2, SequenceEnd: 2, Operations: []protocol.Operation{
			{Sequence: 2, Kind: protocol.OperationUpdate, TabID: 10, Record: &updated},
		}},
	})
	if rejection != nil {
		t.Fatalf("bind metadata delta: %v", rejection)
	}
	if _, rejection := reconciler.State().ApplyDelta(delta); rejection != nil {
		t.Fatalf("apply metadata delta: %v", rejection)
	}
	afterDelta := reconciler.State().Handoff()
	assertSafeHandoffMetadata(t, afterDelta, 2, 2)
	if afterDelta.Digest == initial.Digest || afterDelta.DigestVersion != ProjectionDigestVersion || afterDelta.ErrorClass != "" {
		t.Fatalf("delta did not publish current versioned digest: %+v", afterDelta)
	}
	assertNoTabContent(t, afterDelta)

	_, rejection = reconciler.State().ApplyDelta(DeltaInput{
		Partition: reconciler.Partition(), Epoch: replacementEpoch, BaseRevision: 2, Revision: 3,
		SequenceStart: 3, SequenceEnd: 3,
	})
	if rejection == nil || rejection.Cause != CauseMismatchedProjectionEpoch {
		t.Fatalf("stale epoch rejection = %v", rejection)
	}
	resync := reconciler.State().Handoff()
	if resync.Status != StatusResyncRequired || resync.Queryable || resync.ErrorClass != CauseMismatchedProjectionEpoch || !resync.ErrorClass.KnownCause() {
		t.Fatalf("handoff did not expose a closed error class and resync state: %+v", resync)
	}
	assertNoTabContent(t, resync)
}

func assertSafeHandoffMetadata(t *testing.T, handoff Handoff, sequence, revision uint64) {
	t.Helper()
	if handoff.Sequence != sequence || handoff.Revision != revision || handoff.Count != len(handoff.Identities) || handoff.DurationMillis < 0 {
		t.Fatalf("handoff metadata sequence=%d revision=%d count=%d duration=%d: %+v", sequence, revision, handoff.Count, handoff.DurationMillis, handoff)
	}
}

func assertNoTabContent(t *testing.T, handoff Handoff) {
	t.Helper()
	encoded, err := json.Marshal(handoff)
	if err != nil {
		t.Fatalf("marshal safe handoff: %v", err)
	}
	for _, forbidden := range []string{
		"PRIVATE_TITLE_SENTINEL", "PRIVATE_QUERY_SENTINEL", "PRIVATE_PATH_SENTINEL",
		"UPDATED_TITLE_SENTINEL", "updated_query_sentinel", "https://private.invalid",
	} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("handoff disclosed tab content %q: %s", forbidden, encoded)
		}
	}
}
