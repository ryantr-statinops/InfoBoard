package rank

// bestEvidence selects the strongest evidence by class, then field preference,
// then stable token and position order.
func bestEvidence(items []Evidence) (Evidence, bool) {
	var best Evidence
	found := false
	for _, item := range items {
		if !found || strongerEvidence(item, best) {
			best = item
			found = true
		}
	}
	return best, found
}

func strongerEvidence(left, right Evidence) bool {
	leftClass := left.Class.rankIndex()
	rightClass := right.Class.rankIndex()
	if leftClass != rightClass {
		return leftClass < rightClass
	}
	leftField := left.Field.rankIndex()
	rightField := right.Field.rankIndex()
	if leftField != rightField {
		return leftField < rightField
	}
	if left.TokenIndex != right.TokenIndex {
		return left.TokenIndex < right.TokenIndex
	}
	return left.Position < right.Position
}
