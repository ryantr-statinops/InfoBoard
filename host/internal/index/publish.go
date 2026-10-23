package index

import (
	"sort"
)

// Commit builds sorted postings off to the side and returns one immutable ready
// state. Empty input is a successful ready index with zero records.
func (s *Staged) Commit() *State {
	postings := make(map[string][]int64)
	records := make(map[int64]Record, len(s.records))
	for _, id := range s.sortedIDs() {
		record := s.records[id]
		records[id] = record
		if !(Eligibility{Eligible: record.Eligible}).AllowsPostings() {
			continue
		}
		for _, token := range recordTokens(record) {
			postings[token] = append(postings[token], id)
		}
	}
	for token := range postings {
		ids := postings[token]
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		postings[token] = uniqueSorted(ids)
	}
	return &State{
		ProfileID:         s.profileID,
		Revision:          s.revision,
		NormalizerVersion: NormalizerVersion,
		Records:           records,
		Postings:          postings,
		Count:             len(records),
	}
}

func uniqueSorted(ids []int64) []int64 {
	if len(ids) < 2 {
		return ids
	}
	out := ids[:1]
	for _, id := range ids[1:] {
		if id != out[len(out)-1] {
			out = append(out, id)
		}
	}
	return out
}

// recordTokens returns the normalized tokens contributed by one record.
func recordTokens(record Record) []string {
	forms := []struct {
		field Field
		raw   string
		max   int
	}{
		{FieldTitle, record.Title, MaxTitleScalars},
		{FieldURL, record.URL, MaxURIScalars},
		{FieldDomain, record.Domain, MaxDomainScalars},
		{FieldWindowLabel, record.WindowLabel, MaxLabelScalars},
		{FieldGroupLabel, record.GroupLabel, MaxLabelScalars},
	}
	tokens := make([]string, 0, 16)
	for _, item := range forms {
		form, err := BuildForm(item.field, item.raw, item.max)
		if err != nil || form.Search == "" {
			continue
		}
		tokens = append(tokens, tokenize(form.Search)...)
	}
	for _, label := range StateLabels(record.Pinned, record.Active) {
		tokens = append(tokens, tokenize(label.Search)...)
	}
	return tokens
}
