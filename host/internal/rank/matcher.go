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

// exactEvidence returns the strongest token-level match. Exact matches outrank
// prefixes, and primary fields outrank context fields.
func (index candidateIndex) exactEvidence(token string) (Evidence, bool) {
	for _, field := range index.fields {
		if field.field.isContext() {
			continue
		}
		for position, fieldToken := range field.tokens {
			if fieldToken != token {
				continue
			}
			class := ClassExactToken
			if field.text == token {
				class = ClassExactFullField
			}
			return Evidence{
				Field:        field.field,
				Class:        class,
				Position:     position,
				Contribution: class.base(),
			}, true
		}
	}
	if runeCount(token) >= 2 {
		for _, field := range index.fields {
			if field.field.isContext() {
				continue
			}
			for position, fieldToken := range field.tokens {
				if !hasPrefix(fieldToken, token) {
					continue
				}
				return Evidence{
					Field:        field.field,
					Class:        ClassTokenPrefix,
					Position:     position,
					Contribution: ClassTokenPrefix.base(),
				}, true
			}
		}
	}
	for _, field := range index.fields {
		if !field.field.isContext() {
			continue
		}
		for position, fieldToken := range field.tokens {
			if fieldToken == token {
				return Evidence{
					Field:        field.field,
					Class:        ClassContextOnly,
					Position:     position,
					Contribution: ClassContextOnly.base(),
				}, true
			}
		}
	}
	return Evidence{}, false
}

// fullFieldEvidence reports when the complete normalized query equals a
// complete primary field.
func (index candidateIndex) fullFieldEvidence(tokens []string) (Evidence, bool) {
	query := strings.Join(uniqueTokens(tokens), " ")
	if query == "" {
		return Evidence{}, false
	}
	for _, field := range index.fields {
		if field.field.isContext() || field.text != query {
			continue
		}
		return Evidence{
			Field:        field.field,
			Class:        ClassExactFullField,
			Position:     0,
			Contribution: ClassExactFullField.base(),
		}, true
	}
	return Evidence{}, false
}

func runeCount(value string) int {
	return len([]rune(value))
}

func hasPrefix(value, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
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
