package index

import "testing"

func TestStageRejectsDuplicateIDs(t *testing.T) {
	_, err := Stage("p", 1, []Record{{TabID: 1}, {TabID: 1}})
	if err == nil {
		t.Fatalf("duplicate ids accepted")
	}
}

func TestCommitEmptyIsReady(t *testing.T) {
	staged, err := Stage("p", 1, nil)
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	state := staged.Commit()
	if state.Count != 0 || state.Revision != 1 || state.NormalizerVersion != NormalizerVersion {
		t.Fatalf("unexpected state: %+v", state)
	}
}

func TestCommitBuildsPostings(t *testing.T) {
	staged, err := Stage("p", 1, []Record{
		{TabID: 2, Eligible: true, Title: "Example Page"},
		{TabID: 1, Eligible: true, Title: "example"},
	})
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	state := staged.Commit()
	ids := state.Postings["example"]
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("postings not deterministic: %v", ids)
	}
}

func TestIneligibleRecordHasNoPostings(t *testing.T) {
	staged, _ := Stage("p", 1, []Record{{TabID: 1, Eligible: false, Title: "example"}})
	state := staged.Commit()
	if len(state.Postings) != 0 {
		t.Fatalf("ineligible record produced postings: %v", state.Postings)
	}
}
