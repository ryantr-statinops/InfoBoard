package domain

import (
	"encoding/json"
	"os"
	"testing"
)

func TestProfileTabIdentity(t *testing.T) {
	p, err := ParseProfileID("123e4567-e89b-42d3-a456-426614174000")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseProfileID("bad"); err == nil {
		t.Fatal("malformed ID accepted")
	}
	if (TabIdentity{p, ContextNormal, 7}).Equal(TabIdentity{p, ContextPrivate, 7}) {
		t.Fatal("context collision")
	}
}

func TestProjectionSnapshotAtomicityAndRevision(t *testing.T) {
	p, _ := ParseProfileID("123e4567-e89b-42d3-a456-426614174000")
	epoch := "123e4567-e89b-42d3-a456-426614174001"
	state := NewProjection(p, ContextNormal)
	record := EligibleTabRecord{ProfileID: p, ContextKind: ContextNormal, TabIdentity: TabIdentity{p, ContextNormal, 7}, WindowID: 1, TitleDisplay: "one", TitleSearch: "one", URLSearch: "https://example.test", URLDisplay: "example.test", DomainDisplay: "example.test", DomainSearch: "example.test", Eligible: true, ObservedAt: 1, ProjectionEpoch: epoch, ProjectionRevision: 1}
	if err := state.Snapshot(epoch, []EligibleTabRecord{record}); err != nil {
		t.Fatal(err)
	}
	bad := record
	bad.ProfileID = "bad"
	if err := state.Snapshot(epoch, []EligibleTabRecord{record, bad}); err == nil {
		t.Fatal("invalid snapshot accepted")
	}
	if len(state.Records) != 1 || state.Revision != 1 {
		t.Fatal("partial snapshot mutation")
	}
	record.TabIdentity.TabID = 8
	event := ProjectionEvent{ProfileID: p, ContextKind: ContextNormal, Epoch: epoch, Sequence: 2, PreviousRevision: 1, Operation: "upsert", Identity: record.TabIdentity, Record: &record}
	if got := state.Apply(event); got != "APPLIED" {
		t.Fatal(got)
	}
	if got := state.Apply(event); got != "DUPLICATE" {
		t.Fatal(got)
	}
	event.Sequence = 4
	if got := state.Apply(event); got != "SNAPSHOT_REQUIRED" {
		t.Fatal(got)
	}
	if len(state.Records) != 2 || state.Revision != 2 {
		t.Fatal("invalid event mutated state")
	}
}

func TestActivationRetention(t *testing.T) {
	p, _ := ParseProfileID("123e4567-e89b-42d3-a456-426614174000")
	rows := make([]ActivationMetadata, 502)
	for i := range rows {
		rows[i] = ActivationMetadata{ProfileID: p, ContextKind: ContextNormal, TabIdentity: TabIdentity{p, ContextNormal, int64(i)}, Domain: "example.test", ActivatedAt: 1000 + int64(i), Source: ActivationKeyboardEnter, StorageSequence: uint64(i)}
	}
	rows[0].ContextKind = ContextPrivate
	got := RetainActivations(rows, 1600)
	if len(got) != 500 || got[0].StorageSequence != 2 {
		t.Fatalf("retention count=%d", len(got))
	}
}

func TestSharedPhase02Fixtures(t *testing.T) {
	raw, err := os.ReadFile("../../../fixtures/domain/phase-02/phase-02.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Fixtures []struct {
			FixtureID string                     `json:"fixture_id"`
			Input     map[string]json.RawMessage `json:"input"`
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Fixtures) != 6 {
		t.Fatalf("fixture count=%d", len(catalog.Fixtures))
	}
	for _, fixture := range catalog.Fixtures {
		if fixture.FixtureID != "FX-PROFILE-ISOLATION" {
			continue
		}
		var tabID int64
		var profiles []string
		if err := json.Unmarshal(fixture.Input["same_tab_id"], &tabID); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(fixture.Input["profile_ids"], &profiles); err != nil {
			t.Fatal(err)
		}
		if len(profiles) != 2 {
			t.Fatal("shared isolation fixture malformed")
		}
		first, err := ParseProfileID(profiles[0])
		if err != nil {
			t.Fatal(err)
		}
		second, err := ParseProfileID(profiles[1])
		if err != nil {
			t.Fatal(err)
		}
		if (TabIdentity{first, ContextNormal, tabID}) == (TabIdentity{second, ContextNormal, tabID}) {
			t.Fatal("shared fixture identity collision")
		}
	}
}
