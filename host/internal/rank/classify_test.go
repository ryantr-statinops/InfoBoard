package rank

import "testing"

func TestExactEvidenceClassifiesFullFieldAndToken(t *testing.T) {
	whole := buildCandidateIndex(Candidate{
		Fields: []FieldText{{Field: FieldTitle, Text: "example", Tokens: []string{"example"}}},
	})
	evidence, ok := whole.exactEvidence("example")
	if !ok || evidence.Class != ClassExactFullField {
		t.Fatalf("whole field evidence = %+v, %v", evidence, ok)
	}

	partial := buildCandidateIndex(Candidate{
		Fields: []FieldText{{Field: FieldTitle, Text: "example page", Tokens: []string{"example", "page"}}},
	})
	evidence, ok = partial.exactEvidence("example")
	if !ok || evidence.Class != ClassExactToken {
		t.Fatalf("partial field evidence = %+v, %v", evidence, ok)
	}
}

func TestFullFieldEvidenceMatchesCompleteQuery(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{{Field: FieldTitle, Text: "example page", Tokens: []string{"example", "page"}}},
	})
	evidence, ok := index.fullFieldEvidence([]string{"example", "page"})
	if !ok || evidence.Class != ClassExactFullField {
		t.Fatalf("full field evidence = %+v, %v", evidence, ok)
	}
	if _, ok := index.fullFieldEvidence([]string{"example"}); ok {
		t.Fatal("partial query matched the complete field")
	}
}

func TestExactEvidenceUsesBoundedPrefix(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{{Field: FieldTitle, Text: "example page", Tokens: []string{"example", "page"}}},
	})
	evidence, ok := index.exactEvidence("exa")
	if !ok || evidence.Class != ClassTokenPrefix {
		t.Fatalf("prefix evidence = %+v, %v", evidence, ok)
	}
	if _, ok := index.exactEvidence("e"); ok {
		t.Fatal("single-rune prefix matched")
	}
}

func TestExactEvidencePrefersExactOverPrefix(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{
			{Field: FieldDomain, Text: "ex", Tokens: []string{"ex"}},
			{Field: FieldTitle, Text: "example page", Tokens: []string{"example", "page"}},
		},
	})
	evidence, ok := index.exactEvidence("example")
	if !ok || evidence.Field != FieldTitle || evidence.Class != ClassExactToken {
		t.Fatalf("evidence = %+v, %v", evidence, ok)
	}
}

func TestExactEvidenceDoesNotPrefixContextFields(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{{Field: FieldWindow, Text: "example", Tokens: []string{"example"}}},
	})
	if _, ok := index.exactEvidence("exa"); ok {
		t.Fatal("context prefix matched")
	}
}
