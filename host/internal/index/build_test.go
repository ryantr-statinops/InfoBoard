package index

import "testing"

func TestStageRejectsEmptyProfile(t *testing.T) {
	if _, err := Stage("", 1, nil); err == nil {
		t.Fatalf("empty profile accepted")
	}
}

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
	if len(state.Provenance[1]) != 0 {
		t.Fatalf("ineligible record produced provenance: %v", state.Provenance[1])
	}
}

func TestCommitEmitsURLSegmentsNotScheme(t *testing.T) {
	staged, err := Stage("p", 1, []Record{{
		TabID: 1, Eligible: true, Title: "Guide",
		URL: "https://Docs.Example.test/guide/start",
	}})
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	state := staged.Commit()
	for _, token := range []string{"docs", "example", "test", "guide", "start"} {
		if len(state.Postings[token]) != 1 {
			t.Fatalf("missing url segment %q in postings", token)
		}
	}
	if _, ok := state.Postings["https"]; ok {
		t.Fatalf("scheme emitted as a token")
	}
}

func TestCommitRetainsFieldProvenance(t *testing.T) {
	staged, err := Stage("p", 1, []Record{{
		TabID: 7, Eligible: true, Pinned: true,
		Title: "Example Guide", Domain: "example.test",
	}})
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	state := staged.Commit()
	refs := state.Provenance[7]
	if len(refs) == 0 {
		t.Fatalf("missing provenance")
	}
	fields := make(map[Field]bool)
	for _, ref := range refs {
		fields[ref.Field] = true
		if ref.Position < 0 {
			t.Fatalf("negative position: %+v", ref)
		}
	}
	for _, want := range []Field{FieldTitle, FieldDomain, FieldPinned} {
		if !fields[want] {
			t.Fatalf("missing field provenance %s: %+v", want, refs)
		}
	}
}
