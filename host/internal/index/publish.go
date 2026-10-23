package index

import (
	"sort"
)

// Commit builds sorted postings off to the side and returns one immutable ready
// state. Empty input is a successful ready index with zero records.
func (s *Staged) Commit() *State {
	postings := make(map[string][]int64)
	records := make(map[int64]Record, len(s.records))
	provenance := make(map[int64][]TokenRef, len(s.records))
	for _, id := range s.sortedIDs() {
		record := s.records[id]
		records[id] = record
		if !(Eligibility{Eligible: record.Eligible}).AllowsPostings() {
			continue
		}
		tokens, refs := recordTokens(record)
		provenance[id] = refs
		for _, token := range tokens {
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
		Provenance:        provenance,
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

// recordTokens returns the normalized tokens and provenance contributed by one
// record. Iteration follows the closed field table so field order is stable.
func recordTokens(record Record) ([]string, []TokenRef) {
	tokens := make([]string, 0, 16)
	refs := make([]TokenRef, 0, 16)
	for _, spec := range FieldTable {
		if spec.Field == FieldPinned || spec.Field == FieldActive {
			continue
		}
		form, err := BuildForm(spec.Field, fieldValue(record, spec.Field), spec.MaxScalars)
		if err != nil || form.Search == "" {
			continue
		}
		for position, token := range tokenize(form.Search) {
			tokens = append(tokens, token)
			refs = append(refs, TokenRef{Field: spec.Field, Position: position, Source: string(spec.Field)})
		}
	}
	for _, label := range StateLabels(record.Pinned, record.Active) {
		for position, token := range tokenize(label.Search) {
			tokens = append(tokens, token)
			refs = append(refs, TokenRef{Field: label.Field, Position: position, Source: string(label.Field)})
		}
	}
	return tokens, refs
}

// fieldValue maps a record to the closed field table without accepting unknown
// field names from callers.
func fieldValue(record Record, field Field) string {
	switch field {
	case FieldTitle:
		return record.Title
	case FieldURL:
		return record.URL
	case FieldDomain:
		return record.Domain
	case FieldWindowLabel:
		return record.WindowLabel
	case FieldGroupLabel:
		return record.GroupLabel
	default:
		return ""
	}
}
