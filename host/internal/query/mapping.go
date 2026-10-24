package query

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"

	"github.com/ryantr-statinops/InfoBoard/host/internal/index"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/internal/rank"
)

// candidates converts the immutable index records into the bounded IP-11
// candidate contract. Only eligible records in the published profile state are
// visible, and every field is re-bounded through the IP-10 form contract.
func (s *Service) candidates(state *index.State, request protocol.QueryRequest) ([]rank.Candidate, error) {
	ids := make([]int64, 0, len(state.Records))
	for id := range state.Records {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(left, right int) bool { return ids[left] < ids[right] })

	candidates := make([]rank.Candidate, 0, len(ids))
	for _, id := range ids {
		record := state.Records[id]
		if !record.Eligible || record.TabID <= 0 || record.WindowID <= 0 {
			continue
		}
		fields, err := fieldTexts(record)
		if err != nil {
			return nil, err
		}
		activatedAt := int64(0)
		if s.recency != nil {
			if value, ok := s.recency.ActivatedAt(state.ProfileID, record.TabID); ok && value >= 0 {
				activatedAt = value
			}
		}
		candidates = append(candidates, rank.Candidate{
			ProfileID:   state.ProfileID,
			TabID:       record.TabID,
			WindowID:    record.WindowID,
			GroupID:     rankGroupID(record.GroupID),
			Fields:      fields,
			Pinned:      record.Pinned,
			Active:      record.Active,
			ActivatedAt: activatedAt,
		})
	}
	return candidates, nil
}

func fieldTexts(record index.Record) ([]rank.FieldText, error) {
	specs := []struct {
		indexField index.Field
		rankField  rank.Field
		raw        string
		maxScalars int
	}{
		{index.FieldTitle, rank.FieldTitle, record.Title, index.MaxTitleScalars},
		{index.FieldURL, rank.FieldURL, record.URL, index.MaxURIScalars},
		{index.FieldDomain, rank.FieldDomain, record.Domain, index.MaxDomainScalars},
		{index.FieldWindowLabel, rank.FieldWindow, record.WindowLabel, index.MaxLabelScalars},
		{index.FieldGroupLabel, rank.FieldGroup, record.GroupLabel, index.MaxLabelScalars},
	}
	fields := make([]rank.FieldText, 0, len(specs))
	for _, spec := range specs {
		if spec.raw == "" {
			continue
		}
		form, err := index.BuildForm(spec.indexField, spec.raw, spec.maxScalars)
		if err != nil {
			return nil, err
		}
		tokens := index.TokenizeSearch(form.Search)
		if len(tokens) == 0 {
			continue
		}
		fields = append(fields, rank.FieldText{
			Field:  spec.rankField,
			Text:   form.Search,
			Tokens: tokens,
		})
	}
	return fields, nil
}

// resultRows projects IP-11's ordered results onto the bounded wire rows. The
// index record remains the source of display metadata and opaque identities.
func resultRows(state *index.State, results []rank.Result) []protocol.ResultRow {
	rows := make([]protocol.ResultRow, 0, len(results))
	for _, result := range results {
		record, ok := state.Records[result.TabID]
		if !ok {
			continue
		}
		explanations := make([]string, 0, len(result.Explanations))
		for _, explanation := range result.Explanations {
			explanations = append(explanations, string(explanation))
		}
		rows = append(rows, protocol.ResultRow{
			ResultID:           resultID(state.ProfileID, state.Revision, result.TabID, result.WindowID),
			TabID:              result.TabID,
			WindowID:           result.WindowID,
			GroupID:            cloneGroup(record.GroupID),
			TitleDisplay:       record.TitleDisplay,
			DomainDisplay:      record.DomainDisplay,
			WindowLabelDisplay: cloneString(record.WindowLabelDisplay),
			GroupLabelDisplay:  cloneString(record.GroupLabelDisplay),
			Pinned:             record.Pinned,
			Score:              result.Score,
			BestClass:          string(result.BestClass),
			Explanation:        explanations,
		})
	}
	return rows
}

func resultID(profileID string, revision uint64, tabID, windowID int64) string {
	digest := sha256.Sum256([]byte(profileID + "\x00" + strconv.FormatUint(revision, 10) + "\x00" + strconv.FormatInt(tabID, 10) + "\x00" + strconv.FormatInt(windowID, 10)))
	return hex.EncodeToString(digest[:16])
}

// rankGroupID follows the IP-11 identity contract, which treats a non-positive
// group as absent. The wire result still carries the original index identity.
func rankGroupID(group *int64) *int64 {
	if group == nil || *group <= 0 {
		return nil
	}
	return cloneGroup(group)
}

func cloneGroup(group *int64) *int64 {
	if group == nil {
		return nil
	}
	value := *group
	return &value
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
