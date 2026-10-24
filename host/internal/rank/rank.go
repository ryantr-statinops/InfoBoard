// Package rank owns deterministic ranking and explanations (IP-11).
//
// The package is pure: it consumes already-normalized query and field tokens,
// never reads the wall clock, and performs no network, storage, browser, page
// evaluation, or process work.
package rank

const (
	// ModelVersion stamps every ranked result. Any weight or class change must
	// increment this value and update the phase-11 fixtures.
	ModelVersion = "rank-v1"

	// ExplanationSchemaVersion versions the bounded explanation payload.
	ExplanationSchemaVersion = 1
)

// Class is the closed set of lexical evidence classes, strongest first.
type Class string

const (
	ClassExactFullField   Class = "exact_full_field"
	ClassExactToken       Class = "exact_token"
	ClassTokenPrefix      Class = "token_prefix"
	ClassContiguousPhrase Class = "contiguous_phrase"
	ClassFuzzy            Class = "fuzzy"
	ClassContextOnly      Class = "context_only"
)

// classPrecedence is the canonical strongest-to-weakest class order.
var classPrecedence = []Class{
	ClassExactFullField,
	ClassExactToken,
	ClassTokenPrefix,
	ClassContiguousPhrase,
	ClassFuzzy,
	ClassContextOnly,
}

// base returns the fixed rank-v1 lexical base for a class.
func (c Class) base() int64 {
	switch c {
	case ClassExactFullField:
		return 1000
	case ClassExactToken:
		return 900
	case ClassTokenPrefix:
		return 700
	case ClassContiguousPhrase:
		return 680
	case ClassFuzzy:
		return 400
	case ClassContextOnly:
		return 100
	default:
		return 0
	}
}

// rankIndex returns the precedence position, lower being stronger.
func (c Class) rankIndex() int {
	for index, candidate := range classPrecedence {
		if candidate == c {
			return index
		}
	}
	return len(classPrecedence)
}

// Field is the closed set of evidence fields.
type Field string

const (
	FieldTitle  Field = "title"
	FieldDomain Field = "domain"
	FieldURL    Field = "url"
	FieldWindow Field = "window"
	FieldGroup  Field = "group"
	FieldState  Field = "state"
)

// fieldOrder is the canonical field-preference order used for tie-breaking.
var fieldOrder = []Field{
	FieldTitle,
	FieldDomain,
	FieldURL,
	FieldWindow,
	FieldGroup,
	FieldState,
}

// weight returns the fixed rank-v1 field weight.
func (f Field) weight() int64 {
	switch f {
	case FieldTitle:
		return 80
	case FieldDomain:
		return 60
	case FieldURL:
		return 40
	case FieldWindow:
		return 20
	case FieldGroup:
		return 15
	case FieldState:
		return 10
	default:
		return 0
	}
}

func (f Field) rankIndex() int {
	for index, candidate := range fieldOrder {
		if candidate == f {
			return index
		}
	}
	return len(fieldOrder)
}

func (f Field) isContext() bool {
	return f == FieldWindow || f == FieldGroup || f == FieldState
}

// Explanation is a stable, user-facing explanation code.
type Explanation string

const (
	ExplanationTitleMatch     Explanation = "title_match"
	ExplanationDomainMatch    Explanation = "domain_match"
	ExplanationURLMatch       Explanation = "url_match"
	ExplanationWindowContext  Explanation = "window_context"
	ExplanationGroupContext   Explanation = "group_context"
	ExplanationStateContext   Explanation = "state_context"
	ExplanationRecent         Explanation = "recent_activation"
	ExplanationCurrentWindow  Explanation = "current_window"
	ExplanationSameGroup      Explanation = "same_group"
	ExplanationPinned         Explanation = "pinned"
	ExplanationStaleOrMissing Explanation = "stale_or_missing"
)

// explanationOrder is the fixed emitted order for explanation codes.
var explanationOrder = []Explanation{
	ExplanationTitleMatch,
	ExplanationDomainMatch,
	ExplanationURLMatch,
	ExplanationWindowContext,
	ExplanationGroupContext,
	ExplanationStateContext,
	ExplanationRecent,
	ExplanationCurrentWindow,
	ExplanationSameGroup,
	ExplanationPinned,
	ExplanationStaleOrMissing,
}
