package rank

import "testing"

func TestActiveWindowBonusUsesOpaqueIdentity(t *testing.T) {
	input := validInput()
	input.CurrentWindowID = 2
	candidate := validInput().Candidates[0]
	if got := activeWindowBonusFor(input, candidate); got != activeWindowBonus {
		t.Fatalf("bonus = %d", got)
	}
	candidate.WindowID = 3
	if got := activeWindowBonusFor(input, candidate); got != 0 {
		t.Fatalf("bonus = %d", got)
	}
	input.CurrentWindowID = 0
	if got := activeWindowBonusFor(input, candidate); got != 0 {
		t.Fatalf("zero context bonus = %d", got)
	}
}

func TestPinnedOrGroupBonusUsesOpaqueIdentity(t *testing.T) {
	group := int64(7)
	other := int64(8)
	input := validInput()
	input.CurrentGroupID = &group
	candidate := validInput().Candidates[0]
	candidate.GroupID = &group
	if got := pinnedOrGroupBonusFor(input, candidate); got != sameGroupBonus {
		t.Fatalf("group bonus = %d", got)
	}
	candidate.Pinned = true
	if got := pinnedOrGroupBonusFor(input, candidate); got != maxPinnedOrGroupBonus {
		t.Fatalf("combined bonus = %d", got)
	}
	candidate.GroupID = &other
	if got := pinnedOrGroupBonusFor(input, candidate); got != pinnedBonus {
		t.Fatalf("pinned bonus = %d", got)
	}
	candidate.Pinned = false
	if got := pinnedOrGroupBonusFor(input, candidate); got != 0 {
		t.Fatalf("no bonus = %d", got)
	}
}

func TestRecencyBonusUsesOneMinuteBuckets(t *testing.T) {
	cases := []struct {
		age  int64
		want int64
	}{
		{0, 30},
		{59, 30},
		{60, 29},
		{1799, 1},
		{1800, 0},
		{3600, 0},
		{-10, 30},
	}
	for _, test := range cases {
		if got := recencyBonus(1000, 1000-test.age); got != test.want {
			t.Fatalf("age %d: bonus = %d, want %d", test.age, got, test.want)
		}
	}
	if got := recencyBonus(1000, 0); got != 0 {
		t.Fatalf("absent activation bonus = %d", got)
	}
}

func TestScoreCandidateIncludesIdentityAndRecency(t *testing.T) {
	input := validInput()
	input.CurrentWindowID = 2
	input.CurrentGroupID = ptrInt64(3)
	input.RankingTimestamp = 100
	candidate := validInput().Candidates[0]
	candidate.GroupID = ptrInt64(3)
	candidate.Pinned = true
	candidate.ActivatedAt = 90
	scored, ok := scoreCandidate(input, candidate)
	if !ok {
		t.Fatal("candidate was not scored")
	}
	if scored.components.ActiveWindowBonus != activeWindowBonus {
		t.Fatalf("active window bonus = %d", scored.components.ActiveWindowBonus)
	}
	if scored.components.PinnedOrGroupBonus != maxPinnedOrGroupBonus {
		t.Fatalf("pinned/group bonus = %d", scored.components.PinnedOrGroupBonus)
	}
	if scored.components.RecencyBonus != 30 {
		t.Fatalf("recency bonus = %d", scored.components.RecencyBonus)
	}
}

func ptrInt64(value int64) *int64 { return &value }
