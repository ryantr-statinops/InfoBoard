package rank

// uniqueTokens removes duplicate query tokens while preserving first
// occurrence order. Coverage is AND-oriented over this stable set.
func uniqueTokens(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))
	unique := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		unique = append(unique, token)
	}
	return unique
}

// matchCoverage requires every unique query token to have evidence. Any
// missing token excludes the candidate from the result set.
func matchCoverage(index candidateIndex, tokens []string) ([]Evidence, bool) {
	unique := uniqueTokens(tokens)
	evidence := make([]Evidence, 0, len(unique))
	for tokenIndex, token := range unique {
		match, ok := index.exactEvidence(token)
		if !ok {
			return nil, false
		}
		match.TokenIndex = tokenIndex
		evidence = append(evidence, match)
	}
	return evidence, true
}
