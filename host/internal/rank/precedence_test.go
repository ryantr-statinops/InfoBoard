package rank

import "testing"

func TestBestEvidenceUsesClassPrecedence(t *testing.T) {
	order := []Class{
		ClassExactFullField,
		ClassExactToken,
		ClassTokenPrefix,
		ClassContiguousPhrase,
		ClassFuzzy,
		ClassContextOnly,
	}
	for index := 0; index < len(order); index++ {
		evidence := []Evidence{
			{Field: FieldTitle, Class: order[len(order)-1]},
			{Field: FieldTitle, Class: order[index]},
		}
		best, ok := bestEvidence(evidence)
		if !ok || best.Class != order[index] {
			t.Fatalf("index %d: best = %+v, want %s", index, best, order[index])
		}
	}
}
