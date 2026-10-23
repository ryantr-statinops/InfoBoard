package index

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"testing"
)

// postingsDigest canonicalizes every token and its ordered candidate IDs so two
// builds can be compared without depending on Go map iteration order.
func postingsDigest(state *State) string {
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

func TestDeterministicPostingsIgnoreInputOrder(t *testing.T) {
	records := []Record{
		{TabID: 1, Eligible: true, Title: "Example Page", Domain: "example.test", Pinned: true},
		{TabID: 2, Eligible: true, Title: "example page", URL: "https://example.test/guide/start"},
		{TabID: 3, Eligible: true, Title: "Other", GroupLabel: "Research"},
	}
	reversed := []Record{records[2], records[1], records[0]}

	first := mustCommit(t, records)
	second := mustCommit(t, reversed)

	if first.NormalizerVersion != second.NormalizerVersion {
		t.Fatalf("version mismatch: %s vs %s", first.NormalizerVersion, second.NormalizerVersion)
	}
	if postingsDigest(first) != postingsDigest(second) {
		t.Fatalf("postings digest changed with input order")
	}
	for token := range first.Postings {
		left := first.Postings[token]
		right := second.Postings[token]
		if len(left) != len(right) {
			t.Fatalf("token %q candidate count changed: %v vs %v", token, left, right)
		}
		for i := range left {
			if left[i] != right[i] {
				t.Fatalf("token %q candidate order changed: %v vs %v", token, left, right)
			}
		}
	}
}

func TestDeterministicRepeatedBuilds(t *testing.T) {
	records := []Record{
		{TabID: 10, Eligible: true, Title: "Docs", WindowLabel: "Work"},
		{TabID: 11, Eligible: true, Title: "Docs", GroupLabel: "Reading", Active: true},
	}
	digests := make(map[string]struct{}, 5)
	for run := 0; run < 5; run++ {
		digests[postingsDigest(mustCommit(t, records))] = struct{}{}
	}
	if len(digests) != 1 {
		t.Fatalf("repeated builds produced %d digests", len(digests))
	}
}

func TestPreparedQueryIsStable(t *testing.T) {
	for _, raw := range []string{"  CAFÉ  Menu ", "CAFÉ Menu", "café menu"} {
		prepared, err := PrepareQuery(raw)
		if err != nil {
			t.Fatalf("prepare %q: %v", raw, err)
		}
		if len(prepared.Tokens) != 2 || prepared.Tokens[0] != "café" || prepared.Tokens[1] != "menu" {
			t.Fatalf("unstable query tokens for %q: %v", raw, prepared.Tokens)
		}
	}
}

func mustCommit(t *testing.T, records []Record) *State {
	t.Helper()
	staged, err := Stage("p", 1, records)
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	return staged.Commit()
}
