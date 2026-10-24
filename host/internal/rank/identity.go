package rank

const (
	activeWindowBonus     int64 = 20
	pinnedBonus           int64 = 5
	sameGroupBonus        int64 = 5
	maxPinnedOrGroupBonus int64 = 10

	recencyHorizonSeconds int64 = 30 * 60
	recencyMaxBonus       int64 = 30
)

// activeWindowBonusFor compares opaque window identity only.
func activeWindowBonusFor(input Input, candidate Candidate) int64 {
	if input.CurrentWindowID > 0 && candidate.WindowID == input.CurrentWindowID {
		return activeWindowBonus
	}
	return 0
}

// pinnedOrGroupBonusFor compares opaque group identity only.
func pinnedOrGroupBonusFor(input Input, candidate Candidate) int64 {
	bonus := int64(0)
	if candidate.Pinned {
		bonus += pinnedBonus
	}
	if sameGroup(input.CurrentGroupID, candidate.GroupID) {
		bonus += sameGroupBonus
	}
	if bonus > maxPinnedOrGroupBonus {
		return maxPinnedOrGroupBonus
	}
	return bonus
}

func sameGroup(left, right *int64) bool {
	return left != nil && right != nil && *left == *right
}

// recencyBonus clamps future activation to age zero and ignores absent or
// out-of-horizon activation timestamps.
func recencyBonus(rankingTimestamp, activatedAt int64) int64 {
	if activatedAt <= 0 {
		return 0
	}
	age := rankingTimestamp - activatedAt
	if age < 0 {
		age = 0
	}
	if age >= recencyHorizonSeconds {
		return 0
	}
	bonus := recencyMaxBonus - age/60
	if bonus < 0 {
		return 0
	}
	return bonus
}
