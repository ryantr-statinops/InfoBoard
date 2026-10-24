package rank

import "sort"

type classHistogram [6]int
type fieldSignature [6]int

func histogramOf(evidence []Evidence) classHistogram {
	var histogram classHistogram
	for _, item := range evidence {
		index := item.Class.rankIndex()
		if index >= 0 && index < len(histogram) {
			histogram[index]++
		}
	}
	return histogram
}

func signatureOf(evidence []Evidence) fieldSignature {
	var signature fieldSignature
	for _, item := range evidence {
		index := item.Field.rankIndex()
		if index >= 0 && index < len(signature) {
			signature[index]++
		}
	}
	return signature
}

// rankCandidates scores, orders, and limits eligible candidates.
func rankCandidates(input Input) []scoredCandidate {
	scored := make([]scoredCandidate, 0, len(input.Candidates))
	for _, candidate := range input.Candidates {
		if item, ok := scoreCandidate(input, candidate); ok {
			scored = append(scored, item)
		}
	}
	sort.SliceStable(scored, func(left, right int) bool {
		return lessScored(input, scored[left], scored[right])
	})
	if len(scored) > input.Limit {
		scored = scored[:input.Limit]
	}
	return scored
}

// lessScored implements the deterministic rank-v1 ordering. Identity is the
// final tie-break and is added separately from the evidence signature.
func lessScored(input Input, left, right scoredCandidate) bool {
	leftScore := left.components.Total()
	rightScore := right.components.Total()
	if leftScore != rightScore {
		return leftScore > rightScore
	}
	leftClasses := histogramOf(left.evidence)
	rightClasses := histogramOf(right.evidence)
	for index := range leftClasses {
		if leftClasses[index] != rightClasses[index] {
			return leftClasses[index] > rightClasses[index]
		}
	}
	leftFields := signatureOf(left.evidence)
	rightFields := signatureOf(right.evidence)
	for index := range leftFields {
		if leftFields[index] != rightFields[index] {
			return leftFields[index] > rightFields[index]
		}
	}
	if left.candidate.ActivatedAt != right.candidate.ActivatedAt {
		return left.candidate.ActivatedAt > right.candidate.ActivatedAt
	}
	leftActive := activeWindowBonusFor(input, left.candidate) > 0
	rightActive := activeWindowBonusFor(input, right.candidate) > 0
	if leftActive != rightActive {
		return leftActive
	}
	if left.candidate.Pinned != right.candidate.Pinned {
		return left.candidate.Pinned
	}
	if left.candidate.TabID != right.candidate.TabID {
		return left.candidate.TabID < right.candidate.TabID
	}
	return left.candidate.WindowID < right.candidate.WindowID
}
