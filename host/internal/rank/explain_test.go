package rank

import (
	"reflect"
	"testing"
)

func TestExplanationsAreFixedOrderAndDeduplicated(t *testing.T) {
	input := validInput()
	input.QueryTokens = []string{"example", "work"}
	input.CurrentWindowID = 2
	input.CurrentGroupID = ptrInt64(3)
	input.RankingTimestamp = 100
	candidate := Candidate{
		ProfileID:   "profile",
		TabID:       1,
		WindowID:    2,
		GroupID:     ptrInt64(3),
		Pinned:      true,
		ActivatedAt: 90,
		Fields: []FieldText{
			{Field: FieldTitle, Text: "example", Tokens: []string{"example"}},
			{Field: FieldWindow, Text: "work", Tokens: []string{"work"}},
		},
	}
	scored, ok := scoreCandidate(input, candidate)
	if !ok {
		t.Fatal("candidate was not scored")
	}
	got := explanationsFor(input, scored)
	want := []Explanation{
		ExplanationTitleMatch,
		ExplanationWindowContext,
		ExplanationRecent,
		ExplanationCurrentWindow,
		ExplanationSameGroup,
		ExplanationPinned,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("explanations = %v, want %v", got, want)
	}
}

func TestExplanationsIncludeStaleOrMissingLast(t *testing.T) {
	input := validInput()
	candidate := validInput().Candidates[0]
	candidate.Stale = true
	scored, ok := scoreCandidate(input, candidate)
	if !ok {
		t.Fatal("candidate was not scored")
	}
	got := explanationsFor(input, scored)
	if len(got) != 2 || got[0] != ExplanationTitleMatch || got[1] != ExplanationStaleOrMissing {
		t.Fatalf("explanations = %v", got)
	}
}
