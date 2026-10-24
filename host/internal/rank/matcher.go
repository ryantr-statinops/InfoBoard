package rank

import "strings"

// indexedField is one bounded field with its normalized tokens in order.
type indexedField struct {
	field  Field
	text   string
	tokens []string
}

// candidateIndex holds a candidate's fields in canonical preference order.
type candidateIndex struct {
	fields []indexedField
}

// buildCandidateIndex copies the candidate's bounded fields into canonical
// order and derives state tokens only from the pinned and active flags.
func buildCandidateIndex(candidate Candidate) candidateIndex {
	byField := make(map[Field]FieldText, len(candidate.Fields))
	for _, field := range candidate.Fields {
		byField[field.Field] = field
	}
	fields := make([]indexedField, 0, len(fieldOrder))
	for _, field := range fieldOrder {
		if field == FieldState {
			state := stateTokens(candidate)
			if len(state) > 0 {
				fields = append(fields, indexedField{
					field:  FieldState,
					text:   strings.Join(state, " "),
					tokens: state,
				})
			}
			continue
		}
		text, exists := byField[field]
		if !exists || (text.Text == "" && len(text.Tokens) == 0) {
			continue
		}
		fields = append(fields, indexedField{
			field:  field,
			text:   text.Text,
			tokens: text.Tokens,
		})
	}
	return candidateIndex{fields: fields}
}

// exactEvidence returns the strongest exact-token match for one token. Primary
// fields outrank context fields regardless of their canonical order.
func (index candidateIndex) exactEvidence(token string) (Evidence, bool) {
	var context *Evidence
	for _, field := range index.fields {
		for position, fieldToken := range field.tokens {
			if fieldToken != token {
				continue
			}
			class := ClassExactToken
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
	}
	if context != nil {
		return *context, true
	}
	return Evidence{}, false
}

func stateTokens(candidate Candidate) []string {
	tokens := make([]string, 0, 2)
	if candidate.Pinned {
		tokens = append(tokens, "pinned")
	}
	if candidate.Active {
		tokens = append(tokens, "active")
	}
	return tokens
}
