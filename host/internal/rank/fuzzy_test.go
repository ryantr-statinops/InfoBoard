package rank

import "testing"

func TestExactEvidenceUsesBoundedFuzzyMatch(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{{Field: FieldTitle, Text: "example page", Tokens: []string{"example", "page"}}},
	})
	evidence, ok := index.exactEvidence("exampld")
	if !ok || evidence.Class != ClassFuzzy || evidence.Field != FieldTitle {
		t.Fatalf("fuzzy evidence = %+v, %v", evidence, ok)
	}
	if _, ok := index.exactEvidence("ex"); ok {
		t.Fatal("short token matched fuzzily")
	}
	if _, ok := index.exactEvidence("exampel"); ok {
		t.Fatal("two-edit token matched")
	}
}

func TestExactMatchOutranksFuzzyAcrossFields(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{
			{Field: FieldTitle, Text: "example", Tokens: []string{"example"}},
			{Field: FieldDomain, Text: "exampl", Tokens: []string{"exampl"}},
		},
	})
	evidence, ok := index.exactEvidence("exampl")
	if !ok || evidence.Field != FieldDomain || evidence.Class != ClassExactFullField {
		t.Fatalf("evidence = %+v, %v, want domain exact_full_field", evidence, ok)
	}
}

func TestFuzzyDoesNotMatchContextFields(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{{Field: FieldWindow, Text: "example", Tokens: []string{"example"}}},
	})
	if _, ok := index.exactEvidence("exampl"); ok {
		t.Fatal("context fuzzy match accepted")
	}
}
