package index

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// normalizeForm applies NFKC compatibility normalization and full Unicode case
// folding. The output is deterministic for a given NormalizerVersion.
func normalizeForm(input string) string {
	compat := norm.NFKC.String(input)
	return cases.Fold().String(compat)
}
