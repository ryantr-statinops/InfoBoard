package rank

import "strings"

// matchCandidate requires token coverage and every explicit phrase.
func matchCandidate(index candidateIndex, tokens []string, phrases [][]string) ([]Evidence, bool) {
	evidence, ok := matchCoverage(index, tokens)
	if !ok {
		return nil, false
	}
	phraseEvidence, ok := matchPhrases(index, tokens, phrases)
	if !ok {
		return nil, false
	}
	return append(evidence, phraseEvidence...), true
}

// matchPhrases requires each phrase to occur contiguously inside one field.
func matchPhrases(index candidateIndex, queryTokens []string, phrases [][]string) ([]Evidence, bool) {
	evidence := make([]Evidence, 0, len(phrases))
	for _, phrase := range phrases {
		match, ok := index.phraseEvidence(phrase)
		if !ok {
			return nil, false
		}
		match.TokenIndex = phraseTokenIndex(queryTokens, phrase)
		evidence = append(evidence, match)
	}
	return evidence, true
}

// phraseEvidence finds the first contiguous occurrence in canonical field
// order. Primary fields outrank context fields.
func (index candidateIndex) phraseEvidence(phrase []string) (Evidence, bool) {
	if len(phrase) == 0 {
		return Evidence{}, false
	}
	var context *Evidence
	for _, field := range index.fields {
		position, ok := contiguousPosition(field.tokens, phrase)
		if !ok {
			continue
		}
		class := ClassContiguousPhrase
		if field.field.isContext() {
			class = ClassContextOnly
		}
		evidence := Evidence{
			Field:        field.field,
			Class:        class,
			Position:     position,
			Contribution: class.base(),
		}
		if !field.field.isContext() {
			return evidence, true
		}
		if context == nil {
			copy := evidence
			context = &copy
		}
	}
	if context != nil {
		return *context, true
	}
	return Evidence{}, false
}

func contiguousPosition(tokens, phrase []string) (int, bool) {
	if len(phrase) > len(tokens) {
		return 0, false
	}
	for start := 0; start+len(phrase) <= len(tokens); start++ {
		matches := true
		for offset, token := range phrase {
			if tokens[start+offset] != token {
				matches = false
				break
			}
		}
		if matches {
			return start, true
		}
	}
	return 0, false
}

func phraseTokenIndex(queryTokens, phrase []string) int {
	unique := uniqueTokens(queryTokens)
	if len(phrase) == 0 {
		return 0
	}
	position, ok := contiguousPosition(unique, phrase)
	if !ok {
		return 0
	}
	return position
}

// phraseText joins normalized phrase tokens for exact whole-field checks.
func phraseText(phrase []string) string {
	return strings.Join(phrase, " ")
}
