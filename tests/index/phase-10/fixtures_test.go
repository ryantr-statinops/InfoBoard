package phase10

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/index"
)

type fixtureFile struct {
	FixtureID  string          `json:"fixture_id"`
	OwnerPhase string          `json:"owner_phase"`
	Input      json.RawMessage `json:"input"`
	Expected   json.RawMessage `json:"expected"`
}

func loadFixtures(t *testing.T) map[string]fixtureFile {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "..", "..", "fixtures", "lexical", "phase-10", "*.json"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("no lexical fixtures found")
	}
	fixtures := make(map[string]fixtureFile, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var fixture fixtureFile
		if err := json.Unmarshal(raw, &fixture); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		if fixture.FixtureID == "" || fixture.OwnerPhase != "IP-10" {
			t.Fatalf("invalid fixture metadata in %s", path)
		}
		fixtures[fixture.FixtureID] = fixture
	}
	return fixtures
}

func TestLexicalFixtures(t *testing.T) {
	fixtures := loadFixtures(t)
	for _, id := range []string{
		"FX-LEX-UNICODE-CASE", "FX-LEX-URL-SEGMENTS", "FX-LEX-FIELD-BOUNDS",
		"FX-LEX-STATE-PROVENANCE", "FX-LEX-EMPTY", "FX-LEX-MALFORMED",
		"FX-LEX-REBUILD", "FX-LEX-DETERMINISTIC", "FX-LEX-1000-TABS",
	} {
		if _, ok := fixtures[id]; !ok {
			t.Fatalf("missing fixture %s", id)
		}
	}

	t.Run("unicode_case", func(t *testing.T) {
		var input struct {
			Title string `json:"title"`
			Raw   string `json:"raw"`
		}
		var expected struct {
			Display           string `json:"display"`
			Search            string `json:"search"`
			NormalizerVersion string `json:"normalizer_version"`
		}
		decode(t, fixtures["FX-LEX-UNICODE-CASE"], &input, &expected)
		form, err := index.BuildForm(index.FieldTitle, input.Raw, index.MaxTitleScalars)
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if form.Display != expected.Display || form.Search != expected.Search {
			t.Fatalf("form = %q/%q, want %q/%q", form.Display, form.Search, expected.Display, expected.Search)
		}
		if expected.NormalizerVersion != index.NormalizerVersion {
			t.Fatalf("normalizer version = %q", expected.NormalizerVersion)
		}
	})

	t.Run("url_segments", func(t *testing.T) {
		var input struct {
			URL string `json:"url"`
		}
		var expected struct {
			HostTokens []string `json:"host_tokens"`
			PathTokens []string `json:"path_tokens"`
		}
		decode(t, fixtures["FX-LEX-URL-SEGMENTS"], &input, &expected)
		segments, err := index.ParseURLSegments(input.URL)
		if err != nil {
			t.Fatalf("parse url: %v", err)
		}
		if fmt.Sprint(segments.Host) != fmt.Sprint(expected.HostTokens) || fmt.Sprint(segments.Path) != fmt.Sprint(expected.PathTokens) {
			t.Fatalf("segments = %v/%v", segments.Host, segments.Path)
		}
	})

	t.Run("field_bounds", func(t *testing.T) {
		var input struct {
			Fields map[string]*string `json:"fields"`
		}
		var expected struct {
			MissingOptionalTokens bool `json:"missing_optional_fields_emit_tokens"`
			TitleBound            int  `json:"title_bounded_scalars"`
			URLBound              int  `json:"url_bounded_scalars"`
		}
		decode(t, fixtures["FX-LEX-FIELD-BOUNDS"], &input, &expected)
		if expected.MissingOptionalTokens || expected.TitleBound != index.MaxTitleScalars || expected.URLBound != index.MaxURIScalars {
			t.Fatalf("field bounds drifted: %+v", expected)
		}
		form, err := index.BuildForm(index.FieldGroupLabel, "", index.MaxLabelScalars)
		if err != nil || form.Search != "" {
			t.Fatalf("missing optional label fabricated a token: %+v %v", form, err)
		}
	})

	t.Run("state_provenance", func(t *testing.T) {
		var input struct {
			Pinned bool `json:"pinned"`
			Active bool `json:"active"`
		}
		var expected struct {
			StateLabels      []string `json:"state_labels"`
			ProvenanceFields []string `json:"provenance_fields"`
			IneligiblePosts  bool     `json:"ineligible_emits_postings"`
		}
		decode(t, fixtures["FX-LEX-STATE-PROVENANCE"], &input, &expected)
		labels := index.StateLabels(input.Pinned, input.Active)
		if len(labels) != len(expected.StateLabels) || labels[0].Search != expected.StateLabels[0] {
			t.Fatalf("state labels = %+v", labels)
		}
		staged, err := index.Stage("profile", 1, []index.Record{{TabID: 1, Eligible: true, Title: "x", Pinned: input.Pinned, Active: input.Active}})
		if err != nil {
			t.Fatalf("stage: %v", err)
		}
		state := staged.Commit()
		for _, field := range expected.ProvenanceFields {
			found := false
			for _, ref := range state.Provenance[1] {
				if string(ref.Field) == field {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing provenance field %s", field)
			}
		}
		ineligible, _ := index.Stage("profile", 2, []index.Record{{TabID: 2, Eligible: false, Title: "x"}})
		if len(ineligible.Commit().Postings) != 0 {
			t.Fatalf("ineligible record emitted postings")
		}
	})

	t.Run("empty", func(t *testing.T) {
		var expected struct {
			Status   string `json:"status"`
			Count    int    `json:"count"`
			Postings int    `json:"postings"`
		}
		var input struct{}
		decode(t, fixtures["FX-LEX-EMPTY"], &input, &expected)
		publisher := index.NewPublisher()
		if err := publisher.Publish(context.Background(), "profile", domain.ContextNormal, "epoch", 1, nil); err != nil {
			t.Fatalf("publish empty: %v", err)
		}
		state := publisher.State()
		if publisher.QueryGate(1) != index.QueryReady || state.Count != expected.Count || len(state.Postings) != expected.Postings {
			t.Fatalf("empty index not ready/zero: count=%d postings=%d", state.Count, len(state.Postings))
		}
	})

	t.Run("malformed", func(t *testing.T) {
		var input struct {
			InvalidUTF8Hex string `json:"invalid_utf8_hex"`
			Query          string `json:"query"`
		}
		var expected struct {
			InvalidUTF8     string `json:"invalid_utf8"`
			EmptyQueryEqual bool   `json:"empty_query_equal"`
			IndexUnchanged  bool   `json:"index_unchanged"`
		}
		decode(t, fixtures["FX-LEX-MALFORMED"], &input, &expected)
		invalid, err := hex.DecodeString(input.InvalidUTF8Hex)
		if err != nil {
			t.Fatalf("decode invalid bytes: %v", err)
		}
		if _, err := index.PrepareQuery(string(invalid)); err == nil || string(err.Code) != expected.InvalidUTF8 {
			t.Fatalf("invalid utf8 not rejected: %+v", err)
		}
		prepared, queryErr := index.PrepareQuery(input.Query)
		if queryErr != nil {
			t.Fatalf("whitespace query rejected: %v", queryErr)
		}
		if prepared.Empty != expected.EmptyQueryEqual {
			t.Fatalf("whitespace query not empty: %+v", prepared)
		}
	})

	t.Run("rebuild", func(t *testing.T) {
		var expected struct {
			Sequence             []string `json:"sequence"`
			PublishedOnSuccess   bool     `json:"published_on_success"`
			PreviousStateOnError bool     `json:"previous_state_on_failure"`
		}
		var input struct{}
		decode(t, fixtures["FX-LEX-REBUILD"], &input, &expected)
		publisher := index.NewPublisher()
		ctx := context.Background()
		if err := publisher.Publish(ctx, "profile", domain.ContextNormal, "epoch", 1, nil); err != nil {
			t.Fatalf("publish revision 1: %v", err)
		}
		publisher.BeginRebuild()
		if status := publisher.QueryGate(2); status != index.QueryRebuilding {
			t.Fatalf("rebuilding gate = %s", status)
		}
		if err := publisher.Publish(ctx, "profile", domain.ContextNormal, "epoch", 2, nil); err != nil {
			t.Fatalf("publish revision 2: %v", err)
		}
		if publisher.QueryGate(2) != index.QueryReady || !expected.PublishedOnSuccess {
			t.Fatalf("revision 2 not ready")
		}
		if err := publisher.Publish(ctx, "profile", domain.ContextNormal, "epoch", 3, []domain.EligibleTabRecord{
			{TabIdentity: domain.TabIdentity{TabID: 1}},
			{TabIdentity: domain.TabIdentity{TabID: 1}},
		}); err == nil {
			t.Fatalf("duplicate rebuild accepted")
		}
		if publisher.QueryGate(2) != index.QueryReady || !expected.PreviousStateOnError {
			t.Fatalf("previous state lost after failed rebuild")
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		var input struct {
			Records []struct {
				TabID int64  `json:"tab_id"`
				Title string `json:"title"`
			} `json:"records"`
			Runs int `json:"runs"`
		}
		var expected struct {
			PostingsDigestEqual bool `json:"postings_digest_equal"`
			CandidateIDsEqual   bool `json:"candidate_ids_equal"`
		}
		decode(t, fixtures["FX-LEX-DETERMINISTIC"], &input, &expected)
		records := make([]index.Record, 0, len(input.Records))
		for _, record := range input.Records {
			records = append(records, index.Record{TabID: record.TabID, Eligible: true, Title: record.Title})
		}
		digests := make(map[string]bool, input.Runs)
		for run := 0; run < input.Runs; run++ {
			if run%2 == 1 {
				reversed := make([]index.Record, len(records))
				for i := range records {
					reversed[len(records)-1-i] = records[i]
				}
				records = reversed
			}
			digests[fixtureDigest(records)] = true
		}
		if len(digests) != 1 || !expected.PostingsDigestEqual || !expected.CandidateIDsEqual {
			t.Fatalf("postings changed across runs: %d digests", len(digests))
		}
	})

	t.Run("scale", func(t *testing.T) {
		var input struct {
			EligibleTabs int `json:"eligible_tabs"`
			QueryScalars int `json:"query_scalars"`
		}
		var expected struct {
			P95Max int `json:"p95_ms_max"`
		}
		decode(t, fixtures["FX-LEX-1000-TABS"], &input, &expected)
		if input.EligibleTabs != benchmarkTabs || expected.P95Max != 50 || input.QueryScalars > 64 {
			t.Fatalf("scale fixture drifted: %+v %+v", input, expected)
		}
	})
}

func decode(t *testing.T, fixture fixtureFile, input, expected any) {
	t.Helper()
	if err := json.Unmarshal(fixture.Input, input); err != nil {
		t.Fatalf("%s input: %v", fixture.FixtureID, err)
	}
	if err := json.Unmarshal(fixture.Expected, expected); err != nil {
		t.Fatalf("%s expected: %v", fixture.FixtureID, err)
	}
}

func fixtureDigest(records []index.Record) string {
	staged, err := index.Stage("profile", 1, records)
	if err != nil {
		return "error:" + err.Error()
	}
	state := staged.Commit()
	tokens := make([]string, 0, len(state.Postings))
	for token := range state.Postings {
		tokens = append(tokens, token)
	}
	sort.Strings(tokens)
	sum := sha256.New()
	for _, token := range tokens {
		fmt.Fprintf(sum, "%s:%v\n", token, state.Postings[token])
	}
	return hex.EncodeToString(sum.Sum(nil))
}
