package rank

import "testing"

func TestBestEvidencePrefersClassThenField(t *testing.T) {
	evidence := []Evidence{
		{Field: FieldTitle, Class: ClassExactToken, TokenIndex: 0},
		{Field: FieldDomain, Class: ClassExactFullField, TokenIndex: 0},
		{Field: FieldTitle, Class: ClassExactToken, TokenIndex: 1},
	}
	best, ok := bestEvidence(evidence)
	if !ok || best.Class != ClassExactFullField || best.Field != FieldDomain {
		t.Fatalf("best = %+v, %v", best, ok)
	}

	best, ok = bestEvidence([]Evidence{
		{Field: FieldURL, Class: ClassExactToken, TokenIndex: 0},
		{Field: FieldTitle, Class: ClassExactToken, TokenIndex: 1},
	})
	if !ok || best.Field != FieldTitle {
		t.Fatalf("best = %+v, %v", best, ok)
	}
}

func TestBestEvidenceUsesStablePosition(t *testing.T) {
	best, ok := bestEvidence([]Evidence{
		{Field: FieldTitle, Class: ClassExactToken, TokenIndex: 1, Position: 0},
		{Field: FieldTitle, Class: ClassExactToken, TokenIndex: 0, Position: 2},
	})
	if !ok || best.TokenIndex != 0 {
		t.Fatalf("best = %+v, %v", best, ok)
	}
}

func TestBestEvidenceEmpty(t *testing.T) {
	if _, ok := bestEvidence(nil); ok {
		t.Fatal("empty evidence produced a match")
	}
}
