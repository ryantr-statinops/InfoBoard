package rank

// explanationsFor emits fixed-order, deduplicated explanation codes. Raw field
// text and query text are never included.
func explanationsFor(input Input, scored scoredCandidate) []Explanation {
	present := make(map[Explanation]bool, len(explanationOrder))
	for _, item := range scored.evidence {
		switch item.Field {
		case FieldTitle:
			present[ExplanationTitleMatch] = true
		case FieldDomain:
			present[ExplanationDomainMatch] = true
		case FieldURL:
			present[ExplanationURLMatch] = true
		case FieldWindow:
			present[ExplanationWindowContext] = true
		case FieldGroup:
			present[ExplanationGroupContext] = true
		case FieldState:
			present[ExplanationStateContext] = true
		}
	}
	if scored.components.RecencyBonus > 0 {
		present[ExplanationRecent] = true
	}
	if scored.components.ActiveWindowBonus > 0 {
		present[ExplanationCurrentWindow] = true
	}
	if sameGroup(input.CurrentGroupID, scored.candidate.GroupID) {
		present[ExplanationSameGroup] = true
	}
	if scored.candidate.Pinned {
		present[ExplanationPinned] = true
	}
	if scored.components.StaleOrMissingPenalty > 0 {
		present[ExplanationStaleOrMissing] = true
	}
	explanations := make([]Explanation, 0, len(present))
	for _, code := range explanationOrder {
		if present[code] {
			explanations = append(explanations, code)
		}
	}
	return explanations
}
