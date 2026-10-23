package index

import (
	"strings"
)

// PreparedQuery is the bounded, normalized query handed to IP-11/IP-12.
type PreparedQuery struct {
	Tokens  []string
	Phrases [][]string
	Empty   bool
}

// PrepareQuery normalizes a query without mutating the index. Empty and
// whitespace-only inputs are equivalent and return Empty.
func PrepareQuery(raw string) (PreparedQuery, *NormalizeError) {
	if err := validate(raw); err != nil {
		return PreparedQuery{}, err
	}
	normalized := collapseWhitespace(normalizeForm(truncateScalars(raw, MaxQueryScalars)))
	if normalized == "" {
		return PreparedQuery{Empty: true}, nil
	}
	phrases := make([][]string, 0, 4)
	for _, phrase := range strings.Split(normalized, "\"") {
		phrase = strings.TrimSpace(phrase)
		if phrase == "" {
			continue
		}
		tokens := tokenize(phrase)
		if len(tokens) > 0 {
			phrases = append(phrases, tokens)
		}
	}
	tokens := tokenize(normalized)
	return PreparedQuery{Tokens: tokens, Phrases: phrases, Empty: len(tokens) == 0}, nil
}
