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
