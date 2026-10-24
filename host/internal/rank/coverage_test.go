package rank

import "testing"

func TestMatchCoverageRequiresEveryUniqueToken(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{
			{Field: FieldTitle, Text: "example page", Tokens: []string{"example", "page"}},
		},
	})
	evidence, ok := matchCoverage(index, []string{"example", "page", "example"})
	if !ok {
		t.Fatal("covered query was rejected")
	}
	if len(evidence) != 2 {
		t.Fatalf("evidence = %d, want 2 after deduplication", len(evidence))
	}
	if _, ok := matchCoverage(index, []string{"example", "missing"}); ok {
		t.Fatal("partially covered query was accepted")
	}
}

func TestMatchCoveragePrefersPrimaryFields(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{
			{Field: FieldWindow, Text: "example", Tokens: []string{"example"}},
			{Field: FieldTitle, Text: "example page", Tokens: []string{"example", "page"}},
		},
	})
	evidence, ok := matchCoverage(index, []string{"example"})
	if !ok {
		t.Fatal("covered query was rejected")
	}
	if evidence[0].Field != FieldTitle || evidence[0].Class != ClassExactToken {
		t.Fatalf("evidence = %+v, want title exact_token", evidence[0])
	}
}

func TestMatchCoverageAllowsContextOnlyMatches(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{
			{Field: FieldWindow, Text: "example", Tokens: []string{"example"}},
		},
	})
	evidence, ok := matchCoverage(index, []string{"example"})
	if !ok {
		t.Fatal("context match was rejected")
	}
	if evidence[0].Class != ClassContextOnly || evidence[0].Field != FieldWindow {
		t.Fatalf("evidence = %+v, want context_only window", evidence[0])
	}
}
