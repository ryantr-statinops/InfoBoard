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
	record.ProjectionRevision = 2
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

func TestProjectionBatchAtomicityAndActivationReference(t *testing.T) {
	profile, _ := ParseProfileID("123e4567-e89b-42d3-a456-426614174000")
	epoch := "123e4567-e89b-42d3-a456-426614174001"
	state := NewProjection(profile, ContextNormal)
	first := EligibleTabRecord{ProfileID: profile, ContextKind: ContextNormal, TabIdentity: TabIdentity{profile, ContextNormal, 7}, WindowID: 1, TitleDisplay: "first", TitleSearch: "first", URLSearch: "https://one.test", URLDisplay: "one.test", DomainDisplay: "one.test", DomainSearch: "one.test", Eligible: true, ObservedAt: 1, ProjectionEpoch: epoch, ProjectionRevision: 1}
	if err := state.Snapshot(epoch, []EligibleTabRecord{first}); err != nil {
		t.Fatal(err)
	}
	second := first
	second.TabIdentity.TabID = 8
	second.ProjectionRevision = 2
	add := ProjectionEvent{ProfileID: profile, ContextKind: ContextNormal, Epoch: epoch, Sequence: 2, PreviousRevision: 1, Operation: "upsert", Identity: second.TabIdentity, Record: &second}
	invalid := add
	invalid.Sequence = 4
	invalid.PreviousRevision = 3
	invalid.Identity.TabID = 9
	invalid.Record = &second
	if got := state.ApplyBatch([]ProjectionEvent{add, invalid}); got != "SNAPSHOT_REQUIRED" {
		t.Fatal(got)
	}
	if state.Revision != 1 || len(state.Records) != 1 {
		t.Fatal("batch partially mutated committed state")
	}
	remove := ProjectionEvent{ProfileID: profile, ContextKind: ContextNormal, Epoch: epoch, Sequence: 3, PreviousRevision: 2, Operation: "remove", Identity: first.TabIdentity}
	if got := state.ApplyBatch([]ProjectionEvent{add, remove}); got != "APPLIED" {
		t.Fatal(got)
	}
	if state.Revision != 3 || len(state.Records) != 1 {
		t.Fatal("valid batch did not commit atomically")
	}
	reference := ActivationReference{ProfileID: profile, ContextKind: ContextNormal, TabID: 8, ProjectionEpoch: epoch, ProjectionRevision: 3, ResultID: "request-1"}
	if !reference.Valid(state) {
		t.Fatal("current reference rejected")
	}
	reference.ProjectionRevision = 2
	if reference.Valid(state) {
		t.Fatal("stale revision accepted")
	}
}

func TestSharedProjectionLifecycleFixture(t *testing.T) {
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
	var profiles []string
	var lifecycle map[string]json.RawMessage
	for _, fixture := range catalog.Fixtures {
		if fixture.FixtureID == "FX-PROFILE-ISOLATION" {
			_ = json.Unmarshal(fixture.Input["profile_ids"], &profiles)
		}
		if fixture.FixtureID == "FX-PROJECTION-LIFECYCLE" {
			lifecycle = fixture.Input
		}
	}
	if len(profiles) != 2 || lifecycle == nil {
		t.Fatal("required shared fixture input missing")
	}
	profile, err := ParseProfileID(profiles[0])
	if err != nil {
		t.Fatal(err)
	}
	var initial []int64
	var added, removed int64
	if err = json.Unmarshal(lifecycle["initial_tab_ids"], &initial); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(lifecycle["add_tab_id"], &added); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(lifecycle["remove_tab_id"], &removed); err != nil {
		t.Fatal(err)
	}
	epoch := "123e4567-e89b-42d3-a456-426614174001"
	makeRecord := func(tabID int64) EligibleTabRecord {
		return EligibleTabRecord{ProfileID: profile, ContextKind: ContextNormal, TabIdentity: TabIdentity{profile, ContextNormal, tabID}, WindowID: 1, TitleDisplay: "same", TitleSearch: "same", URLSearch: "https://example.test", URLDisplay: "example.test", DomainDisplay: "example.test", DomainSearch: "example.test", Eligible: true, ObservedAt: 1, ProjectionEpoch: epoch, ProjectionRevision: 1}
	}
	rows := make([]EligibleTabRecord, 0, len(initial))
	for _, id := range initial {
		rows = append(rows, makeRecord(id))
	}
	state := NewProjection(profile, ContextNormal)
	if err = state.Snapshot(epoch, rows); err != nil {
		t.Fatal(err)
	}
	newRecord := makeRecord(added)
	newRecord.ProjectionRevision = 2
	batch := []ProjectionEvent{{ProfileID: profile, ContextKind: ContextNormal, Epoch: epoch, Sequence: 2, PreviousRevision: 1, Operation: "upsert", Identity: newRecord.TabIdentity, Record: &newRecord}, {ProfileID: profile, ContextKind: ContextNormal, Epoch: epoch, Sequence: 3, PreviousRevision: 2, Operation: "remove", Identity: TabIdentity{profile, ContextNormal, removed}}}
	if got := state.ApplyBatch(batch); got != "APPLIED" {
		t.Fatal(got)
	}
	if state.Revision != 3 || len(state.Records) != 1 {
		t.Fatal("shared lifecycle fixture did not converge")
	}
	if _, ok := state.Records[newRecord.TabIdentity]; !ok {
		t.Fatal("shared lifecycle fixture left wrong identity set")
	}
}
