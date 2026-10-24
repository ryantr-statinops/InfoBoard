package rank

import "testing"

func TestScoreCandidateAppliesStaleAndMissingPenalty(t *testing.T) {
	input := validInput()
	candidate := validInput().Candidates[0]
	candidate.MissingOptional = true
	scored, ok := scoreCandidate(input, candidate)
	if !ok {
		t.Fatal("candidate was not scored")
	}
	if scored.components.StaleOrMissingPenalty != missingOptionalPenalty {
		t.Fatalf("penalty = %d", scored.components.StaleOrMissingPenalty)
	}

	candidate.Stale = true
	scored, ok = scoreCandidate(input, candidate)
	if !ok {
		t.Fatal("candidate was not scored")
	}
	if scored.components.StaleOrMissingPenalty != stalePenalty {
		t.Fatalf("penalty = %d", scored.components.StaleOrMissingPenalty)
	}
}

func TestScoreCandidatePenaltyUsesMaximum(t *testing.T) {
	candidate := validInput().Candidates[0]
	candidate.Stale = true
	candidate.MissingOptional = true
	if got := staleOrMissingPenalty(candidate); got != maxStaleOrMissingPenalty {
		t.Fatalf("penalty = %d, want %d", got, maxStaleOrMissingPenalty)
	}
}
