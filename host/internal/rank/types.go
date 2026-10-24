package rank

// FieldText is one already-normalized field supplied by the caller. Text is the
// complete normalized field; Tokens are its normalized tokens in order.
type FieldText struct {
	Field  Field    `json:"field"`
	Text   string   `json:"text"`
	Tokens []string `json:"tokens"`
}

// Candidate is one bounded eligible record. Identity values are opaque and are
// never derived from display text.
type Candidate struct {
	ProfileID       string      `json:"profile_id"`
	TabID           int64       `json:"tab_id"`
	WindowID        int64       `json:"window_id"`
	GroupID         *int64      `json:"group_id,omitempty"`
	Fields          []FieldText `json:"fields"`
	Pinned          bool        `json:"pinned"`
	Active          bool        `json:"active"`
	ActivatedAt     int64       `json:"activated_at"`
	Stale           bool        `json:"stale"`
	MissingOptional bool        `json:"missing_optional"`
}

// Input is one complete ranking request. The caller supplies the normalized
// query, explicit timestamp, context identities, and bounded candidates.
type Input struct {
	ProfileID          string      `json:"profile_id"`
	ProjectionRevision uint64      `json:"projection_revision"`
	ModelVersion       string      `json:"ranking_model_version"`
	RankingTimestamp   int64       `json:"ranking_timestamp"`
	QueryTokens        []string    `json:"query_tokens"`
	Phrases            [][]string  `json:"phrases,omitempty"`
	CurrentWindowID    int64       `json:"current_window_id"`
	CurrentGroupID     *int64      `json:"current_group_id,omitempty"`
	Limit              int         `json:"limit"`
	Candidates         []Candidate `json:"candidates"`
}

// Components is the named rank-v1 integer score breakdown.
type Components struct {
	Lexical               int64 `json:"lexical"`
	FieldWeight           int64 `json:"field_weight"`
	ContextBonus          int64 `json:"context_bonus"`
	RecencyBonus          int64 `json:"recency_bonus"`
	ActiveWindowBonus     int64 `json:"active_window_bonus"`
	PinnedOrGroupBonus    int64 `json:"pinned_or_group_bonus"`
	StaleOrMissingPenalty int64 `json:"stale_or_missing_penalty"`
}

// Total returns the signed rank-v1 total score.
func (c Components) Total() int64 {
	return c.Lexical + c.FieldWeight + c.ContextBonus + c.RecencyBonus +
		c.ActiveWindowBonus + c.PinnedOrGroupBonus - c.StaleOrMissingPenalty
}

// Evidence is one bounded provenance item. It never carries raw field text.
type Evidence struct {
	TokenIndex   int   `json:"token_index"`
	Field        Field `json:"field"`
	Class        Class `json:"class"`
	Position     int   `json:"position"`
	Contribution int64 `json:"contribution"`
}

// Result is one deterministic ranked result.
type Result struct {
	TabID        int64         `json:"tab_id"`
	WindowID     int64         `json:"window_id"`
	GroupID      *int64        `json:"group_id,omitempty"`
	Score        int64         `json:"score"`
	Components   Components    `json:"components"`
	BestClass    Class         `json:"best_class"`
	Evidence     []Evidence    `json:"evidence"`
	Explanations []Explanation `json:"explanations"`
}

// Output is the versioned IP-12 handoff contract.
type Output struct {
	ProfileID                string   `json:"profile_id"`
	ProjectionRevision       uint64   `json:"projection_revision"`
	ModelVersion             string   `json:"ranking_model_version"`
	ExplanationSchemaVersion int      `json:"explanation_schema_version"`
	Results                  []Result `json:"results"`
}
