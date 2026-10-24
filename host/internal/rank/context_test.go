package rank

import "testing"

func TestScoreCandidateAddsDistinctContextBonus(t *testing.T) {
	input := validInput()
	input.QueryTokens = []string{"foo", "bar"}
	candidate := Candidate{
		ProfileID: "profile",
		TabID:     1,
		WindowID:  2,
		Fields: []FieldText{
			{Field: FieldTitle, Text: "foo", Tokens: []string{"foo"}},
			{Field: FieldWindow, Text: "bar", Tokens: []string{"bar"}},
		},
	}
	scored, ok := scoreCandidate(input, candidate)
	if !ok {
		t.Fatal("candidate was not scored")
	}
	if scored.components.ContextBonus != contextBonusPerField {
		t.Fatalf("context bonus = %d", scored.components.ContextBonus)
	}
}

func TestScoreCandidateCapsContextBonus(t *testing.T) {
	input := validInput()
	input.QueryTokens = []string{"foo", "bar", "pinned"}
	candidate := Candidate{
		ProfileID: "profile",
		TabID:     1,
		WindowID:  2,
		Pinned:    true,
		Fields: []FieldText{
			{Field: FieldWindow, Text: "foo", Tokens: []string{"foo"}},
			{Field: FieldGroup, Text: "bar", Tokens: []string{"bar"}},
		},
	}
	scored, ok := scoreCandidate(input, candidate)
	if !ok {
		t.Fatal("candidate was not scored")
	}
	if scored.components.ContextBonus != maxContextBonus {
		t.Fatalf("context bonus = %d", scored.components.ContextBonus)
	}
}

func TestScoreCandidateWithoutContextHasNoBonus(t *testing.T) {
	input := validInput()
	scored, ok := scoreCandidate(input, validInput().Candidates[0])
	if !ok {
		t.Fatal("candidate was not scored")
	}
	if scored.components.ContextBonus != 0 {
		t.Fatalf("context bonus = %d", scored.components.ContextBonus)
	}
}
