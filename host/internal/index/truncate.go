package index

import "unicode/utf8"

// truncateScalars returns at most maxScalars Unicode code points. It never
// splits a code point, so the result is always valid UTF-8.
func truncateScalars(input string, maxScalars int) string {
	if maxScalars <= 0 {
		return ""
	}
	if utf8.RuneCountInString(input) <= maxScalars {
		return input
	}
	count := 0
	for offset := range input {
		if count == maxScalars {
			return input[:offset]
		}
		count++
	}
	return input
}
