package rank

import (
	"strings"
	"testing"
)

func validInput() Input {
	return Input{
		ProfileID:          "profile",
		ProjectionRevision: 1,
		ModelVersion:       ModelVersion,
		RankingTimestamp:   100,
		QueryTokens:        []string{"example"},
		Limit:              10,
		Candidates: []Candidate{{
			ProfileID: "profile",
			TabID:     1,
			WindowID:  2,
			Fields: []FieldText{{
				Field: FieldTitle, Text: "example", Tokens: []string{"example"},
			}},
		}},
	}
}

func TestValidateAcceptsBoundedInput(t *testing.T) {
	input := validInput()
	if err := validate(input); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
	input.Candidates = nil
	if err := validate(input); err != nil {
		t.Fatalf("empty candidate set rejected: %v", err)
	}
}

func TestValidateRejectsMalformedInput(t *testing.T) {
	longToken := strings.Repeat("a", MaxTokenScalars+1)
	longText := strings.Repeat("a", MaxFieldTextScalars+1)
	manyTokens := make([]string, MaxQueryTokens+1)
	for i := range manyTokens {
		manyTokens[i] = "token"
	}
	manyFieldTokens := make([]string, MaxTokensPerField+1)
	for i := range manyFieldTokens {
		manyFieldTokens[i] = "token"
	}
	manyPhrases := make([][]string, MaxPhrases+1)
	for i := range manyPhrases {
		manyPhrases[i] = []string{"token"}
	}
	manyCandidates := make([]Candidate, MaxCandidates+1)
	for i := range manyCandidates {
		manyCandidates[i] = Candidate{ProfileID: "profile", TabID: int64(i + 1), WindowID: 1}
	}
	zeroGroup := int64(0)
	group := int64(5)

	cases := []struct {
		name   string
		mutate func(*Input)
		code   ErrorCode
	}{
		{"profile", func(in *Input) { in.ProfileID = "" }, ErrInvalidProfile},
		{"revision", func(in *Input) { in.ProjectionRevision = 0 }, ErrInvalidRevision},
		{"model", func(in *Input) { in.ModelVersion = "rank-v2" }, ErrInvalidModel},
		{"timestamp", func(in *Input) { in.RankingTimestamp = -1 }, ErrInvalidTimestamp},
		{"limit low", func(in *Input) { in.Limit = 0 }, ErrInvalidLimit},
		{"limit high", func(in *Input) { in.Limit = MaxResultLimit + 1 }, ErrInvalidLimit},
		{"empty query", func(in *Input) { in.QueryTokens = nil }, ErrEmptyQuery},
		{"many query tokens", func(in *Input) { in.QueryTokens = manyTokens }, ErrTooManyQueryTokens},
		{"empty query token", func(in *Input) { in.QueryTokens = []string{""} }, ErrInvalidQueryToken},
		{"long query token", func(in *Input) { in.QueryTokens = []string{longToken} }, ErrInvalidQueryToken},
		{"invalid utf8 token", func(in *Input) { in.QueryTokens = []string{string([]byte{0xff})} }, ErrInvalidQueryToken},
		{"empty phrase", func(in *Input) { in.Phrases = [][]string{{}} }, ErrInvalidPhrase},
		{"invalid phrase token", func(in *Input) { in.Phrases = [][]string{{""}} }, ErrInvalidPhrase},
		{"too many phrases", func(in *Input) { in.Phrases = manyPhrases }, ErrTooManyPhrases},
		{"too many candidates", func(in *Input) { in.Candidates = manyCandidates }, ErrTooManyCandidates},
		{"cross profile", func(in *Input) { in.Candidates[0].ProfileID = "other" }, ErrCrossProfile},
		{"tab identity", func(in *Input) { in.Candidates[0].TabID = 0 }, ErrInvalidIdentity},
		{"window identity", func(in *Input) { in.Candidates[0].WindowID = 0 }, ErrInvalidIdentity},
		{"group identity", func(in *Input) { in.Candidates[0].GroupID = &zeroGroup }, ErrInvalidIdentity},
		{"duplicate identity", func(in *Input) { in.Candidates = append(in.Candidates, in.Candidates[0]) }, ErrDuplicateCandidate},
		{"invalid field", func(in *Input) { in.Candidates[0].Fields[0].Field = "unknown" }, ErrInvalidField},
		{"state field", func(in *Input) { in.Candidates[0].Fields[0].Field = FieldState }, ErrInvalidField},
		{"duplicate field", func(in *Input) {
			in.Candidates[0].Fields = append(in.Candidates[0].Fields, in.Candidates[0].Fields[0])
		}, ErrDuplicateField},
		{"unbounded field", func(in *Input) { in.Candidates[0].Fields[0].Text = longText }, ErrUnboundedField},
		{"many field tokens", func(in *Input) { in.Candidates[0].Fields[0].Tokens = manyFieldTokens }, ErrTooManyFieldTokens},
		{"invalid field token", func(in *Input) { in.Candidates[0].Fields[0].Tokens = []string{""} }, ErrInvalidFieldToken},
		{"activation", func(in *Input) { in.Candidates[0].ActivatedAt = -1 }, ErrInvalidActivation},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			input := validInput()
			test.mutate(&input)
			err := validate(input)
			if err == nil || err.Code != test.code {
				t.Fatalf("error = %+v, want %s", err, test.code)
			}
		})
	}

	// A valid non-nil group and activation timestamp remain accepted.
	input := validInput()
	input.Candidates[0].GroupID = &group
	input.Candidates[0].ActivatedAt = 50
	if err := validate(input); err != nil {
		t.Fatalf("valid optional fields rejected: %v", err)
	}
}
