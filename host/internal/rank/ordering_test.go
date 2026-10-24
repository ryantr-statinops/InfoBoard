package rank

import "testing"

func orderItem(tab, window, score int64, classes []Class, fields []Field) scoredCandidate {
	evidence := make([]Evidence, 0, len(classes))
	for index := range classes {
		evidence = append(evidence, Evidence{Class: classes[index], Field: fields[index]})
	}
	return scoredCandidate{
		candidate:  Candidate{TabID: tab, WindowID: window},
		evidence:   evidence,
		components: Components{Lexical: score},
	}
}

func TestLessScoredOrdersByScoreThenSignature(t *testing.T) {
	input := validInput()
	low := orderItem(1, 1, 100, []Class{ClassExactToken}, []Field{FieldTitle})
	high := orderItem(2, 2, 200, []Class{ClassFuzzy}, []Field{FieldURL})
	if !lessScored(input, high, low) {
		t.Fatal("higher score did not sort first")
	}

	full := orderItem(1, 1, 100, []Class{ClassExactFullField, ClassFuzzy}, []Field{FieldTitle, FieldURL})
	tokens := orderItem(2, 2, 100, []Class{ClassExactToken, ClassExactToken}, []Field{FieldTitle, FieldTitle})
	if !lessScored(input, full, tokens) {
		t.Fatal("stronger class histogram did not sort first")
	}

	title := orderItem(1, 1, 100, []Class{ClassExactToken, ClassExactToken}, []Field{FieldTitle, FieldURL})
	domain := orderItem(2, 2, 100, []Class{ClassExactToken, ClassExactToken}, []Field{FieldDomain, FieldDomain})
	if !lessScored(input, title, domain) {
		t.Fatal("stronger field signature did not sort first")
	}
}

func TestLessScoredOrdersByActivationAndFlags(t *testing.T) {
	input := validInput()
	recent := orderItem(1, 1, 100, []Class{ClassExactToken}, []Field{FieldTitle})
	recent.candidate.ActivatedAt = 50
	older := orderItem(2, 2, 100, []Class{ClassExactToken}, []Field{FieldTitle})
	older.candidate.ActivatedAt = 40
	if !lessScored(input, recent, older) {
		t.Fatal("recent activation did not sort first")
	}

	input.CurrentWindowID = 2
	active := orderItem(1, 2, 100, []Class{ClassExactToken}, []Field{FieldTitle})
	inactive := orderItem(2, 1, 100, []Class{ClassExactToken}, []Field{FieldTitle})
	if !lessScored(input, active, inactive) {
		t.Fatal("active window did not sort first")
	}

	input.CurrentWindowID = 0
	pinned := orderItem(1, 1, 100, []Class{ClassExactToken}, []Field{FieldTitle})
	pinned.candidate.Pinned = true
	unpinned := orderItem(2, 2, 100, []Class{ClassExactToken}, []Field{FieldTitle})
	if !lessScored(input, pinned, unpinned) {
		t.Fatal("pinned candidate did not sort first")
	}
}

func TestRankCandidatesFiltersAndLimits(t *testing.T) {
	input := validInput()
	input.QueryTokens = []string{"example"}
	input.Limit = 1
	input.Candidates = []Candidate{
		{ProfileID: "profile", TabID: 1, WindowID: 1, Fields: []FieldText{{Field: FieldTitle, Text: "example", Tokens: []string{"example"}}}},
		{ProfileID: "profile", TabID: 2, WindowID: 2, Fields: []FieldText{{Field: FieldTitle, Text: "other", Tokens: []string{"other"}}}},
		{ProfileID: "profile", TabID: 3, WindowID: 3, Fields: []FieldText{{Field: FieldTitle, Text: "example", Tokens: []string{"example"}}}},
	}
	scored := rankCandidates(input)
	if len(scored) != 1 || scored[0].candidate.TabID != 1 {
		t.Fatalf("scored = %+v", scored)
	}
}
