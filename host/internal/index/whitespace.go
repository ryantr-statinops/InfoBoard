package index

import (
	"strings"
	"unicode"
)

// collapseWhitespace trims and folds every run of Unicode whitespace into one
// ASCII space. Meaningful punctuation separators are preserved for tokenization.
func collapseWhitespace(input string) string {
	var builder strings.Builder
	builder.Grow(len(input))
	pendingSpace := false
	for _, r := range input {
		if unicode.IsSpace(r) {
			if builder.Len() > 0 {
				pendingSpace = true
			}
			continue
		}
		if pendingSpace {
			builder.WriteByte(' ')
			pendingSpace = false
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
