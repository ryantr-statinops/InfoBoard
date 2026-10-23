package index

import (
	"strings"
	"unicode"
)

// tokenize splits normalized search text on Unicode whitespace and punctuation.
// Separators are boundaries only; no generic stop-word list is applied.
func tokenize(text string) []string {
	tokens := make([]string, 0, 8)
	var builder strings.Builder
	flush := func() {
		if builder.Len() == 0 {
			return
		}
		token := builder.String()
		builder.Reset()
		if len([]rune(token)) > MaxTokenScalars {
			token = truncateScalars(token, MaxTokenScalars)
		}
		if len(tokens) < MaxTokensPerField {
			tokens = append(tokens, token)
		}
	}
	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			flush()
			continue
		}
		builder.WriteRune(r)
	}
	flush()
	return tokens
}
