package rank

import "sort"

// canonicalEvidence returns a sorted, bounded provenance copy with contributions
// recomputed from the fixed class bases.
func canonicalEvidence(items []Evidence) []Evidence {
	evidence := make([]Evidence, len(items))
	copy(evidence, items)
	for index := range evidence {
		evidence[index].Contribution = evidence[index].Class.base()
	}
	sort.SliceStable(evidence, func(left, right int) bool {
		a := evidence[left]
		b := evidence[right]
		if a.TokenIndex != b.TokenIndex {
			return a.TokenIndex < b.TokenIndex
		}
		if a.Class.rankIndex() != b.Class.rankIndex() {
			return a.Class.rankIndex() < b.Class.rankIndex()
		}
		if a.Field.rankIndex() != b.Field.rankIndex() {
			return a.Field.rankIndex() < b.Field.rankIndex()
		}
		return a.Position < b.Position
	})
	return evidence
}

func cloneGroup(group *int64) *int64 {
	if group == nil {
		return nil
	}
	value := *group
	return &value
}
