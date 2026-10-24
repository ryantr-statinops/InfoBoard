package rank

import (
	"reflect"
	"testing"
)

func TestRankReturnsVersionedOrderedResults(t *testing.T) {
	input := validInput()
	output, err := Rank(input)
	if err != nil {
		t.Fatalf("rank failed: %v", err)
	}
	if output.ProfileID != input.ProfileID || output.ProjectionRevision != input.ProjectionRevision {
		t.Fatalf("output header = %+v", output)
	}
	if output.ModelVersion != ModelVersion || output.ExplanationSchemaVersion != ExplanationSchemaVersion {
		t.Fatalf("output versions = %+v", output)
	}
	if len(output.Results) != 1 {
		t.Fatalf("results = %d", len(output.Results))
	}
	result := output.Results[0]
	if result.TabID != 1 || result.WindowID != 2 {
		t.Fatalf("identity = %+v", result)
	}
	if result.Score != 1080 || result.Components.Lexical != 1000 || result.Components.FieldWeight != 80 {
		t.Fatalf("score = %+v", result)
	}
	if result.BestClass != ClassExactFullField {
		t.Fatalf("best class = %s", result.BestClass)
	}
	if len(result.Evidence) == 0 {
		t.Fatal("evidence is empty")
	}
}

func TestRankRejectsInvalidInput(t *testing.T) {
	input := validInput()
	input.Limit = 0
	if _, err := Rank(input); err == nil {
		t.Fatal("invalid input was ranked")
	}
}

func TestRankIsIndependentOfCandidateOrder(t *testing.T) {
	input := validInput()
	input.QueryTokens = []string{"example"}
	input.Candidates = []Candidate{
		{ProfileID: "profile", TabID: 1, WindowID: 1, Fields: []FieldText{{Field: FieldTitle, Text: "example", Tokens: []string{"example"}}}},
		{ProfileID: "profile", TabID: 2, WindowID: 2, Fields: []FieldText{{Field: FieldTitle, Text: "example", Tokens: []string{"example"}}}},
	}
	forward, err := Rank(input)
	if err != nil {
		t.Fatalf("forward rank failed: %v", err)
	}
	input.Candidates[0], input.Candidates[1] = input.Candidates[1], input.Candidates[0]
	reversed, err := Rank(input)
	if err != nil {
		t.Fatalf("reversed rank failed: %v", err)
	}
	if !reflect.DeepEqual(forward, reversed) {
		t.Fatalf("order-dependent output:\nforward=%+v\nreversed=%+v", forward, reversed)
	}
}
