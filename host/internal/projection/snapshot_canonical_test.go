package projection

import (
	"reflect"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

const canonicalTestProfile = "123e4567-e89b-42d3-a456-426614174000"
const canonicalTestEpoch = "123e4567-e89b-42d3-a456-426614174002"

func canonicalTestRecord(tabID int64) protocol.TabRecord {
	return protocol.TabRecord{
		ProfileID:          canonicalTestProfile,
		ContextKind:        domain.ContextNormal,
		TabID:              tabID,
		WindowID:           1,
		TitleDisplay:       "Tab",
		TitleSearch:        "tab",
		URLSearch:          "https://example.test/",
		URLDisplay:         "https://example.test/",
		DomainDisplay:      "example.test",
		DomainSearch:       "example.test",
		Eligible:           true,
		ObservedAt:         1,
		ProjectionEpoch:    canonicalTestEpoch,
		ProjectionRevision: 1,
	}
}

func canonicalTestReconciler(t *testing.T) *Reconciler {
	t.Helper()
	profile, err := domain.ParseProfileID(canonicalTestProfile)
	if err != nil {
		t.Fatal(err)
	}
	reconciler, rejection := New(Partition{ProfileID: profile, ContextKind: domain.ContextNormal})
	if rejection != nil {
		t.Fatalf("bind canonical test partition: %v", rejection)
	}
	return reconciler
}

func canonicalTestRequest(records ...protocol.TabRecord) SnapshotRequest {
	return SnapshotRequest{
		ProfileID:   canonicalTestProfile,
		ContextKind: domain.ContextNormal,
		Epoch:       canonicalTestEpoch,
		Revision:    1,
		Sequence:    1,
		Records:     records,
	}
}

func TestBindSnapshot_CanonicalOrderDigestAndOptionalFields(t *testing.T) {
	rows := []protocol.TabRecord{canonicalTestRecord(12), canonicalTestRecord(10), canonicalTestRecord(11)}
	rows[0].GroupID = new(int64(4))
	rows[0].GroupLabelDisplay = new("Ops")
	rows[1].WindowLabelDisplay = new("Main")

	first, rejection := canonicalTestReconciler(t).BindSnapshot(canonicalTestRequest(rows...))
	if rejection != nil {
		t.Fatalf("bind snapshot: %v", rejection)
	}
	wantIDs := []int64{10, 11, 12}
	gotIDs := make([]int64, len(first.Records))
	for index, record := range first.Records {
		gotIDs[index] = record.TabIdentity.TabID
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("canonical identity order = %v, want %v", gotIDs, wantIDs)
	}
	if len(first.Digest) != 64 {
		t.Fatalf("snapshot digest has %d hex characters, want 64", len(first.Digest))
	}
	if first.Records[1].WindowLabelDisplay != nil || first.Records[1].GroupLabelDisplay != nil {
		t.Fatal("absent optional labels were materialized while canonicalizing")
	}

	reversed := append([]protocol.TabRecord(nil), rows...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	second, rejection := canonicalTestReconciler(t).BindSnapshot(canonicalTestRequest(reversed...))
	if rejection != nil {
		t.Fatalf("bind reordered snapshot: %v", rejection)
	}
	if first.Digest != second.Digest || !reflect.DeepEqual(first.Records, second.Records) {
		t.Fatal("equivalent input order did not produce the same canonical records and digest")
	}

	emptyLabelRows := append([]protocol.TabRecord(nil), rows...)
	emptyLabelRows[2].WindowLabelDisplay = new("")
	withEmptyLabel, rejection := canonicalTestReconciler(t).BindSnapshot(canonicalTestRequest(emptyLabelRows...))
	if rejection != nil {
		t.Fatalf("bind explicit empty optional label: %v", rejection)
	}
	if withEmptyLabel.Records[1].WindowLabelDisplay == nil || *withEmptyLabel.Records[1].WindowLabelDisplay != "" {
		t.Fatal("explicit empty optional label was not preserved")
	}
	if withEmptyLabel.Digest == first.Digest {
		t.Fatal("digest did not distinguish an absent optional field from an explicit empty value")
	}
}

func TestBindSnapshot_RejectsDuplicateAndCrossProfileRecords(t *testing.T) {
	first := canonicalTestRecord(10)
	duplicate := first
	if _, rejection := canonicalTestReconciler(t).BindSnapshot(canonicalTestRequest(first, duplicate)); rejection == nil || rejection.Cause != CauseDuplicateTabIdentity || rejection.Code != protocol.CodeSnapshotRequired {
		t.Fatalf("duplicate identity rejection = %v, want SNAPSHOT_REQUIRED/duplicate_tab_identity", rejection)
	}

	foreign := canonicalTestRecord(11)
	foreign.ProfileID = "123e4567-e89b-42d3-a456-426614174099"
	if _, rejection := canonicalTestReconciler(t).BindSnapshot(canonicalTestRequest(foreign)); rejection == nil || rejection.Cause != CauseMismatchedProfileID {
		t.Fatalf("cross-profile rejection = %v, want mismatched_profile_id", rejection)
	}
}
func TestBindSnapshot_RequiresRevisionOneForFreshEpoch(t *testing.T) {
	record := canonicalTestRecord(10)
	record.ProjectionRevision = 2
	request := canonicalTestRequest(record)
	request.Revision = 2
	_, rejection := canonicalTestReconciler(t).BindSnapshot(request)
	if rejection == nil || rejection.Cause != CauseMismatchedProjectionRevision || rejection.Code != protocol.CodeSnapshotRequired {
		t.Fatalf("revision-2 fresh snapshot rejection = %v, want SNAPSHOT_REQUIRED/mismatched_projection_revision", rejection)
	}
}
