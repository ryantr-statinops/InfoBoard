package rank

// scoredCandidate carries the matched evidence and named score components for
// one eligible candidate.
type scoredCandidate struct {
	candidate  Candidate
	evidence   []Evidence
	components Components
	bestClass  Class
}

// scoreCandidate matches one candidate and computes its lexical and field
// components. It returns false when token coverage or a phrase fails.
func scoreCandidate(input Input, candidate Candidate) (scoredCandidate, bool) {
	index := buildCandidateIndex(candidate)
	evidence, ok := matchCandidate(index, input.QueryTokens, input.Phrases)
	if !ok {
		return scoredCandidate{}, false
	}
	best, ok := bestEvidence(evidence)
	if !ok {
		return scoredCandidate{}, false
	}
	components := Components{
		Lexical:               best.Class.base(),
		FieldWeight:           best.Field.weight(),
		ContextBonus:          contextBonus(evidence),
		StaleOrMissingPenalty: staleOrMissingPenalty(candidate),
	}
	return scoredCandidate{
		candidate:  candidate,
		evidence:   evidence,
		components: components,
		bestClass:  best.Class,
	}, true
}

const (
	contextBonusPerField = 10
	maxContextBonus      = 20
)

const (
	stalePenalty             int64 = 200
	missingOptionalPenalty   int64 = 40
	maxStaleOrMissingPenalty int64 = 200
)

func staleOrMissingPenalty(candidate Candidate) int64 {
	if candidate.Stale {
		return stalePenalty
	}
	if candidate.MissingOptional {
		return missingOptionalPenalty
	}
	return 0
}

func contextBonus(evidence []Evidence) int64 {
	seen := make(map[Field]struct{}, 3)
	for _, item := range evidence {
		if item.Field.isContext() {
			seen[item.Field] = struct{}{}
		}
	}
	bonus := int64(len(seen) * contextBonusPerField)
	if bonus > maxContextBonus {
		return maxContextBonus
	}
	return bonus
}
