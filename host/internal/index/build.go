package index

import (
	"fmt"
	"sort"
)

// Record is the bounded input to one index build. It carries display metadata,
// never cookies, tokens, page bodies, or full query history.
type Record struct {
	TabID       int64
	ProfileID   string
	Revision    uint64
	Eligible    bool
	Pinned      bool
	Active      bool
	Title       string
	URL         string
	Domain      string
	WindowLabel string
	GroupLabel  string
}

// State is one immutable published index generation.
type State struct {
	ProfileID         string
	Revision          uint64
	NormalizerVersion string
	Records           map[int64]Record
	Postings          map[string][]int64
	Count             int
}

// Staged is an in-progress build. It is never observable to readers.
type Staged struct {
	profileID string
	revision  uint64
	records   map[int64]Record
}

// Stage validates the boundary and records before any postings are allocated.
func Stage(profileID string, revision uint64, records []Record) (*Staged, error) {
	seen := make(map[int64]struct{}, len(records))
	staged := &Staged{
		profileID: profileID,
		revision:  revision,
		records:   make(map[int64]Record, len(records)),
	}
	for _, record := range records {
		if record.TabID <= 0 {
			return nil, fmt.Errorf("INVALID_TAB_ID")
		}
		if _, exists := seen[record.TabID]; exists {
			return nil, fmt.Errorf("DUPLICATE_TAB_ID")
		}
		seen[record.TabID] = struct{}{}
		staged.records[record.TabID] = record
	}
	return staged, nil
}

// sortedIDs returns deterministic iteration order for postings.
func (s *Staged) sortedIDs() []int64 {
	ids := make([]int64, 0, len(s.records))
	for id := range s.records {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
