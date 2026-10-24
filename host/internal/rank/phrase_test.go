package rank

import "testing"

func TestMatchPhrasesRequiresContiguousOrder(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{
			{Field: FieldTitle, Text: "quick brown fox", Tokens: []string{"quick", "brown", "fox"}},
		},
	})
	evidence, ok := matchPhrases(index, []string{"quick", "brown", "fox"}, [][]string{{"quick", "brown"}})
	if !ok {
		t.Fatal("contiguous phrase was rejected")
	}
	if len(evidence) != 1 || evidence[0].Class != ClassContiguousPhrase || evidence[0].Position != 0 {
		t.Fatalf("evidence = %+v", evidence)
	}
	if _, ok := matchPhrases(index, []string{"quick", "brown", "fox"}, [][]string{{"brown", "quick"}}); ok {
		t.Fatal("reversed phrase was accepted")
	}
}

func TestMatchPhrasesRequiresOneField(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{
			{Field: FieldTitle, Text: "quick", Tokens: []string{"quick"}},
			{Field: FieldURL, Text: "brown", Tokens: []string{"brown"}},
		},
	})
	if _, ok := matchPhrases(index, []string{"quick", "brown"}, [][]string{{"quick", "brown"}}); ok {
		t.Fatal("phrase spanning fields was accepted")
	}
}

func TestMatchCandidateCombinesCoverageAndPhrases(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{
			{Field: FieldWindow, Text: "quick brown", Tokens: []string{"quick", "brown"}},
		},
	})
	evidence, ok := matchCandidate(index, []string{"quick", "brown"}, [][]string{{"quick", "brown"}})
	if !ok {
		t.Fatal("covered phrase was rejected")
	}
	if evidence[len(evidence)-1].Class != ClassContextOnly {
		t.Fatalf("phrase evidence = %+v, want context_only", evidence[len(evidence)-1])
	}
}
