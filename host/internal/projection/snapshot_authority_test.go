package projection

import (
	"strings"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

const authorityProfile = "123e4567-e89b-42d3-a456-426614174000"
const authorityEpoch = "123e4567-e89b-42d3-a456-426614174001"

// unshapedEpochValue is present, bounded, and printable but is not the IP-02
// lineage shape, so it separates a record-contract refusal from a fence refusal.
const unshapedEpochValue = "epoch-not-a-uuid"

// authorityRecord is one well-formed eligible record of the bound partition. The
// fence values are supplied by the caller so a case can also state a lineage the
// record must refuse.
func authorityRecord(tabID int64, epoch string, revision uint64) protocol.TabRecord {
	group := int64(7)
	label := "Work"
	return protocol.TabRecord{
		ProfileID:          authorityProfile,
		ContextKind:        domain.ContextNormal,
		TabID:              tabID,
		WindowID:           1,
		GroupID:            &group,
		TitleDisplay:       "Alpha",
		TitleSearch:        "alpha",
		URLDisplay:         "https://alpha.example/a",
		URLSearch:          "https://alpha.example/a",
		DomainDisplay:      "alpha.example",
		DomainSearch:       "alpha.example",
		GroupLabelDisplay:  &label,
		GroupLabelSearch:   &label,
		Pinned:             false,
		Active:             true,
		Eligible:           true,
		ObservedAt:         1000,
		ProjectionEpoch:    epoch,
		ProjectionRevision: revision,
	}
}

func authorityRequest(records ...protocol.TabRecord) SnapshotRequest {
	return SnapshotRequest{
		ProfileID:   authorityProfile,
		ContextKind: domain.ContextNormal,
		Epoch:       authorityEpoch,
		Revision:    1,
		Sequence:    1,
		Records:     records,
	}
}

func authorityReconciler(t *testing.T) *Reconciler {
	t.Helper()
	reconciler, rejection := New(Partition{ProfileID: authorityProfile, ContextKind: domain.ContextNormal})
	if rejection != nil {
		t.Fatalf("binding the authority partition: %v", rejection)
	}
	return reconciler
}

// decodedRecords routes the records through the real IP-07 decoder first, so a
// refused record is proven to be one the wire contract accepted and the projection
// boundary still refused, rather than one the decoder already rejected.
func decodedRecords(t *testing.T, records ...protocol.TabRecord) []protocol.TabRecord {
	t.Helper()
	if records == nil {
		// The wire contract carries the record list as an explicit array, so an
		// intentional empty snapshot states no record rather than a null list.
		records = []protocol.TabRecord{}
	}
	payload, failure := protocol.PayloadOf(protocol.Snapshot{Tabs: records, Sequence: 1})
	if failure != nil {
		t.Fatalf("encoding the snapshot payload: %v", failure)
	}
	snapshot, failure := protocol.DecodeSnapshot(payload, protocol.DefaultLimits())
	if failure != nil {
		t.Fatalf("the IP-07 decoder refused a payload the boundary case needs: %v", failure)
	}
	return snapshot.Tabs
}

// assertNoAuthority proves a refused snapshot changed nothing: the boundary returns
// the zero SnapshotInput, every cause stays inside the closed vocabulary, and the
// committed lineage of the partition is untouched and still not queryable.
func assertNoAuthority(t *testing.T, reconciler *Reconciler, input SnapshotInput, rejection *Rejection, want Cause) {
	t.Helper()
	if rejection == nil {
		t.Fatal("the boundary accepted a snapshot that is not authoritative")
	}
	if rejection.Cause != want {
		t.Fatalf("cause %q, want %q", rejection.Cause, want)
	}
	if !rejection.Cause.KnownCause() {
		t.Fatalf("cause %q is outside the closed cause vocabulary", rejection.Cause)
	}
	if input.Partition != (Partition{}) || input.Epoch != "" || input.Revision != 0 || input.Sequence != 0 || len(input.Records) != 0 {
		t.Fatalf("a refused snapshot returned a partial input %+v", input)
	}
	committed := reconciler.State().Committed()
	if committed.Status != StatusUninitialized || committed.Epoch != "" || committed.Revision != 0 || committed.Count() != 0 {
		t.Fatalf("a refused snapshot left committed status=%s epoch=%q revision=%d count=%d",
			committed.Status, committed.Epoch, committed.Revision, committed.Count())
	}
	if handoff := reconciler.State().Handoff(); handoff.Queryable || handoff.Count != 0 {
		t.Fatalf("a refused snapshot published handoff queryable=%t count=%d", handoff.Queryable, handoff.Count)
	}
}

// TestBindSnapshot_EmptyFenceIsAuthoritative proves the zero-record case: a fully
// fenced snapshot of the known profile/context is authority even with nothing in it,
// so it is the one empty projection the host must accept rather than treat as a
// permission failure, a partial read, or unavailable profile state.
func TestBindSnapshot_EmptyFenceIsAuthoritative(t *testing.T) {
	reconciler := authorityReconciler(t)
	for _, records := range [][]protocol.TabRecord{nil, decodedRecords(t)} {
		input, rejection := reconciler.BindSnapshot(authorityRequest(records...))
		if rejection != nil {
			t.Fatalf("a fully fenced empty snapshot was refused: %v", rejection)
		}
		if len(input.Records) != 0 {
			t.Fatalf("an empty snapshot mapped %d records", len(input.Records))
		}
		if input.Partition != reconciler.Partition() || input.Epoch != authorityEpoch || input.Revision != 1 || input.Sequence != 1 {
			t.Fatalf("an empty snapshot lost its fence: %+v", input)
		}
		// Authority is not commit: IP-05-T07 owns publishing a committed state, so a
		// validated empty snapshot must still leave the partition uninitialized and
		// unqueryable here rather than exposing an empty map.
		committed := reconciler.State().Committed()
		if committed.Status != StatusUninitialized || committed.Revision != 0 {
			t.Fatalf("validation published committed status=%s revision=%d", committed.Status, committed.Revision)
		}
		if reconciler.State().Handoff().Queryable {
			t.Fatal("a validated empty snapshot made the partition queryable before commit")
		}
	}
}

// TestBindSnapshot_RefusesIncompleteFence proves no partial authority: a snapshot
// that cannot state its profile, context, epoch, revision, or sequence, or that
// exceeds the bounded record budget, is refused before any SnapshotInput exists.
func TestBindSnapshot_RefusesIncompleteFence(t *testing.T) {
	oversized := make([]protocol.TabRecord, 0, protocol.MaxSnapshotRecords+1)
	for len(oversized) < protocol.MaxSnapshotRecords+1 {
		oversized = append(oversized, authorityRecord(int64(len(oversized)+1), authorityEpoch, 1))
	}
	unboundedEpoch := strings.Repeat("e", protocol.MaxEpochBytes+1)
	otherProfile := "123e4567-e89b-42d3-a456-426614174099"
	cases := []struct {
		name    string
		request SnapshotRequest
		cause   Cause
	}{
		{name: "absent_profile_id", request: SnapshotRequest{ContextKind: domain.ContextNormal, Epoch: authorityEpoch, Revision: 1, Sequence: 1}, cause: CauseMissingProfileID},
		{name: "unparsable_profile_id", request: SnapshotRequest{ProfileID: "unknown-profile", ContextKind: domain.ContextNormal, Epoch: authorityEpoch, Revision: 1, Sequence: 1}, cause: CauseMissingProfileID},
		{name: "other_profile_id", request: SnapshotRequest{ProfileID: otherProfile, ContextKind: domain.ContextNormal, Epoch: authorityEpoch, Revision: 1, Sequence: 1}, cause: CauseMismatchedProfileID},
		{name: "absent_context_kind", request: SnapshotRequest{ProfileID: authorityProfile, Epoch: authorityEpoch, Revision: 1, Sequence: 1}, cause: CauseMissingContextKind},
		{name: "placeholder_context_kind", request: SnapshotRequest{ProfileID: authorityProfile, ContextKind: "default", Epoch: authorityEpoch, Revision: 1, Sequence: 1}, cause: CauseMissingContextKind},
		{name: "other_context_kind", request: SnapshotRequest{ProfileID: authorityProfile, ContextKind: domain.ContextPrivate, Epoch: authorityEpoch, Revision: 1, Sequence: 1}, cause: CauseMismatchedContextKind},
		{name: "absent_projection_epoch", request: SnapshotRequest{ProfileID: authorityProfile, ContextKind: domain.ContextNormal, Revision: 1, Sequence: 1}, cause: CauseMissingProjectionEpoch},
		{name: "unbounded_projection_epoch", request: SnapshotRequest{ProfileID: authorityProfile, ContextKind: domain.ContextNormal, Epoch: unboundedEpoch, Revision: 1, Sequence: 1}, cause: CauseMissingProjectionEpoch},
		{name: "zero_projection_revision", request: SnapshotRequest{ProfileID: authorityProfile, ContextKind: domain.ContextNormal, Epoch: authorityEpoch, Sequence: 1}, cause: CauseMissingProjectionRevision},
		{name: "zero_sequence", request: SnapshotRequest{ProfileID: authorityProfile, ContextKind: domain.ContextNormal, Epoch: authorityEpoch, Revision: 1}, cause: CauseMissingProjectionRevision},
		{name: "record_budget_exceeded", request: authorityRequest(oversized...), cause: CauseBoundsExceeded},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			reconciler := authorityReconciler(t)
			input, rejection := reconciler.BindSnapshot(item.request)
			assertNoAuthority(t, reconciler, input, rejection, item.cause)
		})
	}
}

// TestBindSnapshot_RefusesRecordOutsideIP02Contract proves every record is proved
// against the IP-02 record contract before a snapshot becomes authority. Each case
// is a record the IP-07 decoder accepts: its scalar budgets are inside the wire
// limits, so only the projection boundary can catch the domain violation.
func TestBindSnapshot_RefusesRecordOutsideIP02Contract(t *testing.T) {
	longURL := strings.Repeat("\U0001f600", 600)    // inside MaxURLBytes scalars, over IP-02 bytes
	longDomain := strings.Repeat("\U0001f600", 100) // inside MaxDomainBytes scalars, over IP-02 bytes
	oversizedWindow := int64(domain.MaxProjectionRevision) + 1
	oversizedGroup := int64(domain.MaxProjectionRevision) + 1
	ineligible := authorityRecord(10, authorityEpoch, 1)
	ineligible.Eligible = false
	unboundedWindow := authorityRecord(10, authorityEpoch, 1)
	unboundedWindow.WindowID = oversizedWindow
	unboundedGroup := authorityRecord(10, authorityEpoch, 1)
	unboundedGroup.GroupID = &oversizedGroup
	oversizedURL := authorityRecord(10, authorityEpoch, 1)
	oversizedURL.URLDisplay = "https://alpha.example/" + longURL
	oversizedURL.URLSearch = oversizedURL.URLDisplay
	oversizedDomain := authorityRecord(10, authorityEpoch, 1)
	oversizedDomain.DomainDisplay = longDomain
	oversizedDomain.DomainSearch = longDomain
	foreignLineage := authorityRecord(10, authorityEpoch, 2)
	cases := []struct {
		name    string
		request SnapshotRequest
		cause   Cause
	}{
		{name: "ineligible_record", request: authorityRequest(ineligible), cause: CauseInvalidRecord},
		{name: "unbounded_window_id", request: authorityRequest(unboundedWindow), cause: CauseInvalidRecord},
		{name: "unbounded_group_id", request: authorityRequest(unboundedGroup), cause: CauseInvalidRecord},
		{name: "url_over_ip02_byte_budget", request: authorityRequest(oversizedURL), cause: CauseInvalidRecord},
		{name: "domain_over_ip02_byte_budget", request: authorityRequest(oversizedDomain), cause: CauseInvalidRecord},
		{name: "record_revision_past_fence", request: authorityRequest(foreignLineage), cause: CauseMismatchedProjectionRevision},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			reconciler := authorityReconciler(t)
			request := item.request
			request.Records = decodedRecords(t, request.Records...)
			input, rejection := reconciler.BindSnapshot(request)
			assertNoAuthority(t, reconciler, input, rejection, item.cause)
		})
	}

	// An epoch that is present, bounded, and printable but is not the IP-02 lineage
	// shape reaches this boundary from both sides of the fence: the fence admits it
	// as an opaque value and the record matches it, so only the IP-02 record
	// contract can refuse it. IP-05-T04 owns requiring the epoch shape of the fence
	// itself, including for a zero-record snapshot.
	t.Run("epoch_outside_ip02_shape", func(t *testing.T) {
		unshaped := authorityRecord(10, unshapedEpochValue, 1)
		request := authorityRequest(unshaped)
		request.Epoch = unshapedEpochValue
		reconciler := authorityReconciler(t)
		request.Records = decodedRecords(t, request.Records...)
		input, rejection := reconciler.BindSnapshot(request)
		assertNoAuthority(t, reconciler, input, rejection, CauseInvalidRecord)
	})
}

// TestBindSnapshot_RefusesWholeBatchOnOneInvalidRecord proves a snapshot is judged
// as a whole: one invalid record refuses the entire batch at its own position, so a
// later commit can never observe the records that happened to precede it.
func TestBindSnapshot_RefusesWholeBatchOnOneInvalidRecord(t *testing.T) {
	ineligible := authorityRecord(11, authorityEpoch, 1)
	ineligible.Eligible = false
	request := authorityRequest()
	request.Records = decodedRecords(t,
		authorityRecord(10, authorityEpoch, 1),
		authorityRecord(11, authorityEpoch, 1),
		authorityRecord(12, authorityEpoch, 1),
	)
	ineligibleIndex := 1
	request.Records[ineligibleIndex] = ineligible

	reconciler := authorityReconciler(t)
	input, rejection := reconciler.BindSnapshot(request)
	assertNoAuthority(t, reconciler, input, rejection, CauseInvalidRecord)
	if rejection.Index != ineligibleIndex {
		t.Fatalf("rejection index %d, want the invalid record position %d", rejection.Index, ineligibleIndex)
	}
	if rejection.Code != protocol.CodeSnapshotRequired {
		t.Fatalf("code %q, want %q so a complete authoritative snapshot repairs the refusal", rejection.Code, protocol.CodeSnapshotRequired)
	}
	if rejection.Resync != protocol.ResyncSnapshotStale || !rejection.Resync.KnownReason() {
		t.Fatalf("resync reason %q, want the closed %q", rejection.Resync, protocol.ResyncSnapshotStale)
	}
	if !rejection.ResyncRequired() {
		t.Fatal("a refused snapshot must ask for a new authoritative snapshot")
	}
}

// TestBindSnapshot_AcceptsRecordsThatSatisfyIP02 proves the accepted path maps a
// conforming record rather than repairing it: every field the peer declared is
// carried into the IP-02 record under the bound partition identity, and the same
// snapshot is accepted whether its records arrive in any order, because canonical
// ordering is IP-05-T03's rule and not this boundary's.
func TestBindSnapshot_AcceptsRecordsThatSatisfyIP02(t *testing.T) {
	reconciler := authorityReconciler(t)
	first := authorityRecord(10, authorityEpoch, 1)
	second := authorityRecord(12, authorityEpoch, 1)
	second.TitleDisplay = "Beta"
	second.TitleSearch = "beta"
	second.URLDisplay = "https://beta.example/b"
	second.URLSearch = "https://beta.example/b"
	second.DomainDisplay = "beta.example"
	second.DomainSearch = "beta.example"
	declared := map[int64]protocol.TabRecord{first.TabID: first, second.TabID: second}

	for _, records := range [][]protocol.TabRecord{{first, second}, {second, first}} {
		request := authorityRequest()
		request.Records = decodedRecords(t, records...)
		input, rejection := reconciler.BindSnapshot(request)
		if rejection != nil {
			t.Fatalf("a conforming snapshot was refused: %v", rejection)
		}
		if len(input.Records) != len(declared) {
			t.Fatalf("mapped %d records, want %d", len(input.Records), len(declared))
		}
		for index, record := range input.Records {
			// The boundary preserves the declared order; canonical ordering is
			// IP-05-T03's rule, so this asserts nothing about sorted output.
			source := records[index]
			wanted := declared[source.TabID]
			if wanted.TabID != source.TabID || wanted.TitleDisplay != source.TitleDisplay {
				t.Fatal("the declared record table does not describe the sent record")
			}
			identity := reconciler.Partition().Identity(wanted.TabID)
			if record.TabIdentity != identity || record.ProfileID != identity.ProfileID || record.ContextKind != identity.ContextKind {
				t.Fatalf("record %d identity %+v, want %+v", index, record.TabIdentity, identity)
			}
			if record.TitleDisplay != wanted.TitleDisplay || record.URLDisplay != wanted.URLDisplay || record.DomainDisplay != wanted.DomainDisplay {
				t.Fatalf("record %d metadata was not mapped verbatim: %+v", index, record)
			}
			if record.ProjectionEpoch != authorityEpoch || record.ProjectionRevision != 1 || !record.Eligible {
				t.Fatalf("record %d lost its declared lineage: %+v", index, record)
			}
		}
	}
	if reconciler.State().Committed().Status != StatusUninitialized {
		t.Fatal("validating a conforming snapshot published committed state before IP-05-T07")
	}
}
