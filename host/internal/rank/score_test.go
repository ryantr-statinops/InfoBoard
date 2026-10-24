package rank

import "testing"

func TestScoreCandidateUsesBestLexicalEvidence(t *testing.T) {
	input := validInput()
	input.QueryTokens = []string{"example", "page"}
	candidate := Candidate{
		ProfileID: "profile",
		TabID:     1,
		WindowID:  2,
		Fields: []FieldText{
			{Field: FieldURL, Text: "example.com", Tokens: []string{"example", "com"}},
			{Field: FieldTitle, Text: "example page", Tokens: []string{"example", "page"}},
		},
	}
	scored, ok := scoreCandidate(input, candidate)
	if !ok {
		t.Fatal("candidate was not scored")
	}
	if scored.components.Lexical != ClassExactFullField.base() {
		t.Fatalf("lexical = %d", scored.components.Lexical)
	}
	if scored.components.FieldWeight != FieldTitle.weight() {
		t.Fatalf("field weight = %d", scored.components.FieldWeight)
	}
	if scored.bestClass != ClassExactFullField {
		t.Fatalf("best class = %s", scored.bestClass)
	}
}

func TestScoreCandidateRejectsIncompleteCoverage(t *testing.T) {
	input := validInput()
	input.QueryTokens = []string{"example", "missing"}
	candidate := Candidate{
		ProfileID: "profile",
		TabID:     1,
		WindowID:  2,
		Fields: []FieldText{
			{Field: FieldTitle, Text: "example page", Tokens: []string{"example", "page"}},
		},
	}
	if _, ok := scoreCandidate(input, candidate); ok {
		t.Fatal("incomplete candidate was scored")
	}
}
