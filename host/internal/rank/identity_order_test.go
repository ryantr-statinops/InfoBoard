package rank

import "testing"

func TestRankCandidatesUsesOpaqueIdentityTieBreak(t *testing.T) {
	input := validInput()
	input.QueryTokens = []string{"example"}
	input.Limit = 10
	first := Candidate{ProfileID: "profile", TabID: 2, WindowID: 2, Fields: []FieldText{{Field: FieldTitle, Text: "example", Tokens: []string{"example"}}}}
	second := Candidate{ProfileID: "profile", TabID: 1, WindowID: 1, Fields: []FieldText{{Field: FieldTitle, Text: "example", Tokens: []string{"example"}}}}
	input.Candidates = []Candidate{first, second}
	scored := rankCandidates(input)
	if len(scored) != 2 || scored[0].candidate.TabID != 1 || scored[1].candidate.TabID != 2 {
		t.Fatalf("forward order = %+v", scored)
	}

	input.Candidates = []Candidate{second, first}
	reversed := rankCandidates(input)
	if len(reversed) != 2 || reversed[0].candidate.TabID != 1 || reversed[1].candidate.TabID != 2 {
		t.Fatalf("reversed order = %+v", reversed)
	}
}
