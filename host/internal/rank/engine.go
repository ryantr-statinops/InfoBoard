package rank

// Rank validates one bounded request and returns deterministic ordered results.
// It is pure: the caller supplies the explicit timestamp and normalized tokens.
func Rank(input Input) (Output, error) {
	if err := validate(input); err != nil {
		return Output{}, err
	}
	output := Output{
		ProfileID:                input.ProfileID,
		ProjectionRevision:       input.ProjectionRevision,
		ModelVersion:             ModelVersion,
		ExplanationSchemaVersion: ExplanationSchemaVersion,
		Results:                  make([]Result, 0, input.Limit),
	}
	for _, scored := range rankCandidates(input) {
		output.Results = append(output.Results, Result{
			TabID:        scored.candidate.TabID,
			WindowID:     scored.candidate.WindowID,
			GroupID:      cloneGroup(scored.candidate.GroupID),
			Score:        scored.components.Total(),
			Components:   scored.components,
			BestClass:    scored.bestClass,
			Evidence:     canonicalEvidence(scored.evidence),
			Explanations: nil,
		})
	}
	return output, nil
}
