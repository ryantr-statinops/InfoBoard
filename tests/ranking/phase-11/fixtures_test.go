package phase11

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/ryantr-statinops/InfoBoard/host/internal/rank"
)

type fixtureFile struct {
	FixtureID  string          `json:"fixture_id"`
	OwnerPhase string          `json:"owner_phase"`
	Input      json.RawMessage `json:"input"`
	Expected   json.RawMessage `json:"expected"`
}

var fixtureIDs = []string{
	"FX-RANKING-EXACT-PRECEDENCE",
	"FX-RANKING-DUPLICATE-TITLE",
	"FX-RANKING-TYPO-FUZZY",
	"FX-RANKING-MULTI-WINDOW-CONTEXT",
	"FX-RANKING-PINNED-GROUP-CONTEXT",
	"FX-RANKING-STALE-MISSING",
	"FX-RANKING-EXPLANATION-REPRODUCIBILITY",
	"FX-RANKING-1000-TABS",
}

func loadFixtures(t testing.TB) map[string]fixtureFile {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "..", "..", "fixtures", "ranking", "phase-11", "*.json"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(paths) != len(fixtureIDs) {
		t.Fatalf("fixture files = %d, want %d", len(paths), len(fixtureIDs))
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
		if fixture.FixtureID == "" || fixture.OwnerPhase != "IP-11" {
			t.Fatalf("invalid fixture metadata in %s", path)
		}
		fixtures[fixture.FixtureID] = fixture
	}
	return fixtures
}

func TestRankingFixtures(t *testing.T) {
	fixtures := loadFixtures(t)
	for _, id := range fixtureIDs {
		if _, ok := fixtures[id]; !ok {
			t.Fatalf("missing fixture %s", id)
		}
	}

	for _, id := range fixtureIDs[:len(fixtureIDs)-1] {
		fixture := fixtures[id]
		t.Run(id, func(t *testing.T) {
			var input rank.Input
			if err := json.Unmarshal(fixture.Input, &input); err != nil {
				t.Fatalf("decode input: %v", err)
			}
			var expected rank.Output
			if err := json.Unmarshal(fixture.Expected, &expected); err != nil {
				t.Fatalf("decode expected: %v", err)
			}
			actual, err := rank.Rank(input)
			if err != nil {
				t.Fatalf("rank: %v", err)
			}
			assertOutputEqual(t, actual, expected)
			if actual.ModelVersion != rank.ModelVersion || actual.ExplanationSchemaVersion != rank.ExplanationSchemaVersion {
				t.Fatalf("output versions = %+v", actual)
			}
			if len(actual.Results) > input.Limit {
				t.Fatalf("results = %d, limit = %d", len(actual.Results), input.Limit)
			}
			for _, result := range actual.Results {
				if len(result.Evidence) > rank.MaxQueryTokens+rank.MaxPhrases+1 {
					t.Fatalf("unbounded evidence: %d", len(result.Evidence))
				}
			}

			reversed := input
			reversed.Candidates = append([]rank.Candidate(nil), input.Candidates...)
			for left, right := 0, len(reversed.Candidates)-1; left < right; left, right = left+1, right-1 {
				reversed.Candidates[left], reversed.Candidates[right] = reversed.Candidates[right], reversed.Candidates[left]
			}
			reversedOutput, err := rank.Rank(reversed)
			if err != nil {
				t.Fatalf("reversed rank: %v", err)
			}
			assertOutputEqual(t, reversedOutput, actual)
		})
	}

	t.Run("FX-RANKING-1000-TABS", func(t *testing.T) {
		fixture := fixtures["FX-RANKING-1000-TABS"]
		var input rank.Input
		if err := json.Unmarshal(fixture.Input, &input); err != nil {
			t.Fatalf("decode input: %v", err)
		}
		if len(input.Candidates) != 1000 || input.Limit != rank.MaxResultLimit {
			t.Fatalf("fixture bounds = %d/%d", len(input.Candidates), input.Limit)
		}
		if len(input.QueryTokens) != 1 || utf8.RuneCountInString(input.QueryTokens[0]) != 64 {
			t.Fatalf("query token = %+v", input.QueryTokens)
		}
		var expected struct {
			ResultCount         int     `json:"result_count"`
			Limit               int     `json:"limit"`
			RankingModelVersion string  `json:"ranking_model_version"`
			TopTabIDs           []int64 `json:"top_tab_ids"`
		}
		if err := json.Unmarshal(fixture.Expected, &expected); err != nil {
			t.Fatalf("decode expected: %v", err)
		}
		actual, err := rank.Rank(input)
		if err != nil {
			t.Fatalf("rank: %v", err)
		}
		if len(actual.Results) != expected.ResultCount || expected.ResultCount != expected.Limit {
			t.Fatalf("results = %d, expected = %+v", len(actual.Results), expected)
		}
		if actual.ModelVersion != expected.RankingModelVersion {
			t.Fatalf("model = %s", actual.ModelVersion)
		}
		for index, tabID := range expected.TopTabIDs {
			if actual.Results[index].TabID != tabID {
				t.Fatalf("result %d = %d, want %d", index, actual.Results[index].TabID, tabID)
			}
		}
	})
}

func TestRankingFixtureRespectsLimit(t *testing.T) {
	fixtures := loadFixtures(t)
	var input rank.Input
	if err := json.Unmarshal(fixtures["FX-RANKING-EXACT-PRECEDENCE"].Input, &input); err != nil {
		t.Fatalf("decode input: %v", err)
	}
	input.Limit = 2
	output, err := rank.Rank(input)
	if err != nil {
		t.Fatalf("rank: %v", err)
	}
	if len(output.Results) != 2 {
		t.Fatalf("results = %d, want 2", len(output.Results))
	}
}

func assertOutputEqual(t *testing.T, actual, expected rank.Output) {
	t.Helper()
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("marshal actual: %v", err)
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("marshal expected: %v", err)
	}
	if !bytes.Equal(actualJSON, expectedJSON) {
		t.Fatalf("payload mismatch:\nactual=%s\nexpected=%s", actualJSON, expectedJSON)
	}
}
