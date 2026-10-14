package phase05

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/projection"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

type reconciliationCorpus struct {
	BaseArtifact string               `json:"base_artifact"`
	BaseCaseID   string               `json:"base_case_id"`
	ProfileID    string               `json:"profile_id"`
	OtherProfile string               `json:"other_profile_id"`
	FreshEpoch   string               `json:"fresh_epoch"`
	Fixtures     []reconciliationCase `json:"fixtures"`
}

type reconciliationCase struct {
	FixtureID          string                 `json:"fixture_id"`
	Events             []reconciliationEvent  `json:"events"`
	ReplayExactly      bool                   `json:"replay_exactly"`
	FreshSnapshot      *fixtureSnapshot       `json:"fresh_snapshot"`
	Profiles           []string               `json:"profiles"`
	Contexts           []domain.ContextKind   `json:"contexts"`
	SameTabIDs         []int64                `json:"same_tab_ids"`
	InputRecordOrder   []int64                `json:"input_record_order"`
	RawValues          []string               `json:"raw_values"`
	Expected           reconciliationExpected `json:"expected"`
	Cycles             []reconciliationCycle  `json:"cycles"`
	EligibleBrowserIDs []int64                `json:"eligible_browser_ids"`
}

type reconciliationEvent struct {
	Kind             string `json:"kind"`
	TabID            int64  `json:"tab_id"`
	Sequence         uint64 `json:"sequence"`
	PreviousRevision uint64 `json:"previous_revision"`
	Revision         uint64 `json:"revision"`
	TitleDisplay     string `json:"title_display"`
	TitleSearch      string `json:"title_search"`
}

type fixtureSnapshot struct {
	Epoch     string  `json:"epoch"`
	RecordIDs []int64 `json:"record_ids"`
	Sequence  uint64  `json:"sequence"`
}
type reconciliationCycle struct {
	StartSnapshot    *fixtureSnapshot            `json:"start_snapshot"`
	RequireSnapshot  bool                        `json:"require_snapshot"`
	AcceptedEvent    reconciliationEvent         `json:"accepted_event"`
	DroppedEvent     reconciliationEvent         `json:"dropped_event"`
	RecoverySnapshot fixtureSnapshot             `json:"recovery_snapshot"`
	Expected         reconciliationCycleExpected `json:"expected"`
}

type reconciliationCycleExpected struct {
	AcceptedOutcome   string  `json:"accepted_outcome"`
	DroppedOutcome    string  `json:"dropped_outcome"`
	ResyncRequired    bool    `json:"resync_required"`
	ResyncRevision    uint64  `json:"resync_revision"`
	ResyncSequence    uint64  `json:"resync_sequence"`
	ResyncIdentityIDs []int64 `json:"resync_identity_ids"`
	HandoffStatus     string  `json:"handoff_status"`
	IdentityIDs       []int64 `json:"identity_ids"`
}

type reconciliationExpected struct {
	Outcomes                 []string `json:"outcomes"`
	Outcome                  string   `json:"outcome"`
	Epoch                    string   `json:"epoch"`
	Revision                 uint64   `json:"revision"`
	Sequence                 uint64   `json:"sequence"`
	IdentityIDs              []int64  `json:"identity_ids"`
	RecordTitle              string   `json:"record_title"`
	ResyncRequired           bool     `json:"resync_required"`
	PartitionCount           int      `json:"partition_count"`
	RecordsPerPartition      int      `json:"records_per_partition"`
	MismatchedProfileOutcome string   `json:"mismatched_profile_outcome"`
	MismatchMutates          bool     `json:"mismatch_mutates"`
	DigestVersion            string   `json:"digest_version"`
	ErrorClassBounded        bool     `json:"error_class_bounded"`
	OptionalAbsenceTabID     int64    `json:"optional_absence_tab_id"`
	FinalEpoch               string   `json:"final_epoch"`
	FinalRevision            uint64   `json:"final_revision"`
	FinalSequence            uint64   `json:"final_sequence"`
	UniqueIdentityCount      int      `json:"unique_identity_count"`
	HandoffStatus            string   `json:"handoff_status"`
}

type authorityProjectionCase struct {
	CaseID  string `json:"case_id"`
	Handoff struct {
		ProfileID          string                     `json:"profile_id"`
		ContextKind        domain.ContextKind         `json:"context_kind"`
		ProjectionEpoch    string                     `json:"projection_epoch"`
		PreviousRevision   uint64                     `json:"previous_revision"`
		ProjectionRevision uint64                     `json:"projection_revision"`
		EventSequence      uint64                     `json:"event_sequence"`
		Records            []domain.EligibleTabRecord `json:"records"`
	} `json:"handoff"`
}

func loadReconciliationFixtures(t *testing.T) (reconciliationCorpus, authorityProjectionCase) {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	var corpus reconciliationCorpus
	data, err := os.ReadFile(filepath.Join(root, "fixtures/projection/phase-05/reconciliation.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("decode reconciliation fixtures: %v", err)
	}
	data, err = os.ReadFile(filepath.Join(root, corpus.BaseArtifact))
	if err != nil {
		t.Fatal(err)
	}
	var authority struct {
		Cases []authorityProjectionCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &authority); err != nil {
		t.Fatalf("decode base authority fixture: %v", err)
	}
	for _, item := range authority.Cases {
		if item.CaseID == corpus.BaseCaseID {
			return corpus, item
		}
	}
	t.Fatalf("base authority case %q is missing", corpus.BaseCaseID)
	return reconciliationCorpus{}, authorityProjectionCase{}
}

func reconciliationCaseByID(t *testing.T, corpus reconciliationCorpus, id string) reconciliationCase {
	t.Helper()
	for _, item := range corpus.Fixtures {
		if item.FixtureID == id {
			return item
		}
	}
	t.Fatalf("fixture %q is missing", id)
	return reconciliationCase{}
}

func wireRecords(rows []domain.EligibleTabRecord, profileID string, contextKind domain.ContextKind, epoch string, ids []int64) []protocol.TabRecord {
	selected := make(map[int64]bool, len(ids))
	for _, id := range ids {
		selected[id] = true
	}
	result := make([]protocol.TabRecord, 0, len(rows))
	for _, row := range rows {
		if ids != nil && !selected[row.TabIdentity.TabID] {
			continue
		}
		result = append(result, protocol.TabRecord{
			ProfileID: profileID, ContextKind: contextKind, TabID: row.TabIdentity.TabID,
			WindowID: row.WindowID, GroupID: row.GroupID,
			TitleDisplay: row.TitleDisplay, TitleSearch: row.TitleSearch, URLDisplay: row.URLDisplay, URLSearch: row.URLSearch,
			DomainDisplay: row.DomainDisplay, DomainSearch: row.DomainSearch,
			WindowLabelDisplay: row.WindowLabelDisplay, WindowLabelSearch: row.WindowLabelSearch,
			GroupLabelDisplay: row.GroupLabelDisplay, GroupLabelSearch: row.GroupLabelSearch,
			Pinned: row.Pinned, Active: row.Active, Eligible: row.Eligible, ObservedAt: row.ObservedAt,
			ProjectionEpoch: epoch, ProjectionRevision: 1,
		})
	}
	return result
}
func wireRecordsInOrder(rows []domain.EligibleTabRecord, profileID string, contextKind domain.ContextKind, epoch string, ids []int64) []protocol.TabRecord {
	recordsByID := make(map[int64]protocol.TabRecord)
	for _, record := range wireRecords(rows, profileID, contextKind, epoch, nil) {
		recordsByID[record.TabID] = record
	}
	ordered := make([]protocol.TabRecord, 0, len(ids))
	for _, id := range ids {
		ordered = append(ordered, recordsByID[id])
	}
	return ordered
}

func newReconciliationState(t *testing.T, profileID string, contextKind domain.ContextKind, epoch string, sequence uint64, rows []domain.EligibleTabRecord, ids []int64) *projection.Reconciler {
	t.Helper()
	profile, err := domain.ParseProfileID(profileID)
	if err != nil {
		t.Fatal(err)
	}
	reconciler, rejection := projection.New(projection.Partition{ProfileID: profile, ContextKind: contextKind})
	if rejection != nil {
		t.Fatalf("bind fixture partition: %v", rejection)
	}
	outcome, _, rejection := reconciler.ApplySnapshot(projection.SnapshotRequest{
		ProfileID: profileID, ContextKind: contextKind, Epoch: epoch, Revision: 1,
		Sequence: sequence, Records: wireRecords(rows, profileID, contextKind, epoch, ids),
	})
	if rejection != nil || outcome != projection.SnapshotCommitted {
		t.Fatalf("commit fixture snapshot outcome=%q rejection=%v", outcome, rejection)
	}
	return reconciler
}

func applyFixtureEvent(reconciler *projection.Reconciler, rows []domain.EligibleTabRecord, profileID string, contextKind domain.ContextKind, epoch string, event reconciliationEvent) (string, *projection.Rejection) {
	var record *protocol.TabRecord
	for _, candidate := range wireRecords(rows, profileID, contextKind, epoch, nil) {
		if candidate.TabID != event.TabID {
			continue
		}
		candidate.ProjectionRevision = event.Revision
		if event.TitleDisplay != "" {
			candidate.TitleDisplay = event.TitleDisplay
		}
		if event.TitleSearch != "" {
			candidate.TitleSearch = event.TitleSearch
		}
		record = &candidate
		break
	}
	if record == nil {
		panic("fixture event refers to a missing tab")
	}
	kind := protocol.OperationUpdate
	if event.Kind == "remove" {
		kind = protocol.OperationRemove
		record = nil
	}
	input, rejection := reconciler.BindDelta(projection.DeltaRequest{
		ProfileID: profileID, ContextKind: contextKind, Epoch: epoch, Revision: event.Revision,
		Delta: protocol.Delta{
			BaseRevision: event.PreviousRevision, SequenceStart: event.Sequence, SequenceEnd: event.Sequence,
			Operations: []protocol.Operation{{Sequence: event.Sequence, Kind: kind, TabID: event.TabID, Record: record}},
		},
	})
	if rejection != nil {
		return string(protocol.CodeSnapshotRequired), rejection
	}
	outcome, rejection := reconciler.State().ApplyDelta(input)
	if rejection != nil {
		return string(protocol.CodeSnapshotRequired), rejection
	}
	return string(outcome), nil
}

func identityTabIDs(state projection.Committed) []int64 {
	identities := state.Identities()
	ids := make([]int64, len(identities))
	for index, identity := range identities {
		ids[index] = identity.TabID
	}
	return ids
}

func TestSharedReconciliationFixturesDriveHostProjection(t *testing.T) {
	corpus, base := loadReconciliationFixtures(t)
	ordered := reconciliationCaseByID(t, corpus, "FX-PROJECTION-ORDERED-DELTAS")
	state := newReconciliationState(t, corpus.ProfileID, domain.ContextNormal, base.Handoff.ProjectionEpoch, base.Handoff.EventSequence, base.Handoff.Records, nil)
	var outcomes []string
	for _, event := range ordered.Events {
		outcome, rejection := applyFixtureEvent(state, base.Handoff.Records, corpus.ProfileID, domain.ContextNormal, base.Handoff.ProjectionEpoch, event)
		if rejection != nil {
			t.Fatalf("ordered event seq=%d rejected: %v", event.Sequence, rejection)
		}
		outcomes = append(outcomes, outcome)
	}
	committed := state.State().Committed()
	if !reflect.DeepEqual(outcomes, ordered.Expected.Outcomes) || committed.Revision != ordered.Expected.Revision || committed.Sequence != ordered.Expected.Sequence || !reflect.DeepEqual(identityTabIDs(committed), ordered.Expected.IdentityIDs) {
		t.Fatalf("ordered fixture outcome=%v committed=%+v ids=%v", outcomes, committed, identityTabIDs(committed))
	}
	if record, ok := committed.Lookup(11); !ok || record.TitleDisplay != ordered.Expected.RecordTitle {
		t.Fatalf("ordered fixture record=%+v present=%t", record, ok)
	}

	duplicateFixture := reconciliationCaseByID(t, corpus, "FX-PROJECTION-DUPLICATE-DELTA")
	duplicate := newReconciliationState(t, corpus.ProfileID, domain.ContextNormal, base.Handoff.ProjectionEpoch, base.Handoff.EventSequence, base.Handoff.Records, nil)
	first, rejection := applyFixtureEvent(duplicate, base.Handoff.Records, corpus.ProfileID, domain.ContextNormal, base.Handoff.ProjectionEpoch, duplicateFixture.Events[0])
	if rejection != nil {
		t.Fatal(rejection)
	}
	second := ""
	if duplicateFixture.ReplayExactly {
		second, rejection = applyFixtureEvent(duplicate, base.Handoff.Records, corpus.ProfileID, domain.ContextNormal, base.Handoff.ProjectionEpoch, duplicateFixture.Events[0])
		if rejection != nil {
			t.Fatal(rejection)
		}
	}
	committed = duplicate.State().Committed()
	if !reflect.DeepEqual([]string{first, second}, duplicateFixture.Expected.Outcomes) || committed.Revision != duplicateFixture.Expected.Revision || committed.Sequence != duplicateFixture.Expected.Sequence || !reflect.DeepEqual(identityTabIDs(committed), duplicateFixture.Expected.IdentityIDs) {
		t.Fatalf("duplicate fixture outcomes=%q,%q committed=%+v ids=%v", first, second, committed, identityTabIDs(committed))
	}
}

func TestSharedStaleAndDroppedEventFixturesFenceHostState(t *testing.T) {
	corpus, base := loadReconciliationFixtures(t)
	for _, id := range []string{"FX-PROJECTION-STALE-REVISION", "FX-PROJECTION-MISSED-EVENT"} {
		item := reconciliationCaseByID(t, corpus, id)
		state := newReconciliationState(t, corpus.ProfileID, domain.ContextNormal, base.Handoff.ProjectionEpoch, base.Handoff.EventSequence, base.Handoff.Records, nil)
		outcome, rejection := applyFixtureEvent(state, base.Handoff.Records, corpus.ProfileID, domain.ContextNormal, base.Handoff.ProjectionEpoch, item.Events[0])
		if outcome != item.Expected.Outcomes[0] || rejection == nil {
			t.Fatalf("%s outcome=%q rejection=%v", id, outcome, rejection)
		}
		after := state.State().Committed()
		if after.Status != projection.StatusResyncRequired || after.Revision != item.Expected.Revision || after.Sequence != item.Expected.Sequence || !reflect.DeepEqual(identityTabIDs(after), item.Expected.IdentityIDs) {
			t.Fatalf("%s mutated or kept query-ready state: %+v ids=%v", id, after, identityTabIDs(after))
		}
	}
}

func TestSharedReplacementRestartIsolationAndCanonicalFixtures(t *testing.T) {
	corpus, base := loadReconciliationFixtures(t)
	for _, id := range []string{"FX-PROJECTION-ATOMIC-REPLACE", "FX-PROJECTION-RESTART"} {
		item := reconciliationCaseByID(t, corpus, id)
		state := newReconciliationState(t, corpus.ProfileID, domain.ContextNormal, base.Handoff.ProjectionEpoch, base.Handoff.EventSequence, base.Handoff.Records, nil)
		var outcomes []string
		if id == "FX-PROJECTION-RESTART" {
			state.RequireSnapshot()
			outcome, rejection := applyFixtureEvent(state, base.Handoff.Records, corpus.ProfileID, domain.ContextNormal, base.Handoff.ProjectionEpoch, item.Events[0])
			if rejection == nil {
				t.Fatal("old-session event was accepted during recovery")
			}
			outcomes = append(outcomes, outcome)
		}
		snapshot := item.FreshSnapshot
		profile, err := domain.ParseProfileID(corpus.ProfileID)
		if err != nil {
			t.Fatal(err)
		}
		freshRecords := wireRecords(base.Handoff.Records, corpus.ProfileID, domain.ContextNormal, snapshot.Epoch, snapshot.RecordIDs)
		input, rejection := state.BindSnapshot(projection.SnapshotRequest{ProfileID: corpus.ProfileID, ContextKind: domain.ContextNormal, Epoch: snapshot.Epoch, Revision: 1, Sequence: snapshot.Sequence, Records: freshRecords})
		if rejection != nil {
			t.Fatalf("%s bind fresh snapshot: %v", id, rejection)
		}
		outcome, rejection := state.State().CommitSnapshot(input)
		if rejection != nil {
			t.Fatalf("%s commit fresh snapshot: %v", id, rejection)
		}
		outcomes = append(outcomes, string(outcome))
		committed := state.State().Committed()
		if !reflect.DeepEqual(outcomes, item.Expected.Outcomes) || committed.Epoch != item.Expected.Epoch || committed.Revision != item.Expected.Revision || committed.Sequence != item.Expected.Sequence || !reflect.DeepEqual(identityTabIDs(committed), item.Expected.IdentityIDs) {
			t.Fatalf("%s outcomes=%v committed=%+v ids=%v", id, outcomes, committed, identityTabIDs(committed))
		}
		_ = profile
	}

	isolation := reconciliationCaseByID(t, corpus, "FX-PROJECTION-PROFILE-BOUNDARY")
	states := make([]*projection.Reconciler, 0, isolation.Expected.PartitionCount)
	for _, profileID := range isolation.Profiles {
		for _, contextKind := range isolation.Contexts {
			state := newReconciliationState(t, profileID, contextKind, base.Handoff.ProjectionEpoch, base.Handoff.EventSequence, base.Handoff.Records, isolation.SameTabIDs)
			states = append(states, state)
			handoff := state.State().Handoff()
			if int(handoff.Count) != isolation.Expected.RecordsPerPartition || handoff.ProfileID != state.Partition().ProfileID || handoff.ContextKind != contextKind || !reflect.DeepEqual(identityTabIDs(state.State().Committed()), isolation.SameTabIDs) {
				t.Fatalf("profile/context partition escaped its boundary: %+v", handoff)
			}
		}
	}
	if len(states) != isolation.Expected.PartitionCount {
		t.Fatalf("partition count=%d want=%d", len(states), isolation.Expected.PartitionCount)
	}
	bound := states[0]
	before := bound.State().Committed()
	if _, rejection := bound.BindSnapshot(projection.SnapshotRequest{ProfileID: corpus.OtherProfile, ContextKind: bound.Partition().ContextKind, Epoch: base.Handoff.ProjectionEpoch, Revision: 1, Sequence: 2, Records: nil}); rejection == nil || string(rejection.Code) != isolation.Expected.MismatchedProfileOutcome {
		t.Fatalf("mismatched profile outcome=%v", rejection)
	}
	after := bound.State().Committed()
	if after.Epoch != before.Epoch || after.Revision != before.Revision || after.Sequence != before.Sequence || !reflect.DeepEqual(identityTabIDs(after), identityTabIDs(before)) {
		t.Fatalf("mismatched profile mutated partition: before=%+v after=%+v", before, after)
	}

	canonical := reconciliationCaseByID(t, corpus, "FX-PROJECTION-CANONICAL-OUTPUT")
	profile, err := domain.ParseProfileID(corpus.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	canonicalState, rejection := projection.New(projection.Partition{ProfileID: profile, ContextKind: base.Handoff.ContextKind})
	if rejection != nil {
		t.Fatal(rejection)
	}
	canonicalRecords := wireRecordsInOrder(base.Handoff.Records, corpus.ProfileID, base.Handoff.ContextKind, base.Handoff.ProjectionEpoch, canonical.InputRecordOrder)
	input, rejection := canonicalState.BindSnapshot(projection.SnapshotRequest{ProfileID: corpus.ProfileID, ContextKind: base.Handoff.ContextKind, Epoch: base.Handoff.ProjectionEpoch, Revision: 1, Sequence: base.Handoff.EventSequence, Records: canonicalRecords})
	if rejection != nil {
		t.Fatal(rejection)
	}
	if _, rejection := canonicalState.State().CommitSnapshot(input); rejection != nil {
		t.Fatal(rejection)
	}
	if !reflect.DeepEqual(identityTabIDs(canonicalState.State().Committed()), canonical.Expected.IdentityIDs) {
		t.Fatalf("canonical fixture output=%v", identityTabIDs(canonicalState.State().Committed()))
	}
	for _, record := range canonicalState.State().Committed().Records() {
		if record.TabIdentity.TabID == canonical.Expected.OptionalAbsenceTabID && record.WindowLabelDisplay != nil {
			t.Fatal("canonical fixture materialized an absent optional window label")
		}
	}
}

func TestSharedSafeHandoffFixtureContainsNoRawPageValues(t *testing.T) {
	corpus, base := loadReconciliationFixtures(t)
	item := reconciliationCaseByID(t, corpus, "FX-PROJECTION-SAFE-HANDOFF-METADATA")
	records := append([]domain.EligibleTabRecord(nil), base.Handoff.Records...)
	records[0].TitleDisplay = item.RawValues[0]
	records[0].TitleSearch = item.RawValues[1]
	records[0].URLDisplay = "https://" + item.RawValues[2] + ".invalid"
	records[0].URLSearch = "https://" + item.RawValues[2] + ".invalid"
	state := newReconciliationState(t, corpus.ProfileID, domain.ContextNormal, base.Handoff.ProjectionEpoch, base.Handoff.EventSequence, records, nil)
	_, rejection := state.State().ApplyDelta(projection.DeltaInput{Partition: state.Partition(), Epoch: corpus.FreshEpoch, BaseRevision: 1, Revision: 2, SequenceStart: 2, SequenceEnd: 2})
	if rejection == nil || rejection.Cause != projection.CauseMismatchedProjectionEpoch {
		t.Fatalf("stale epoch rejection = %v", rejection)
	}
	handoff := state.State().Handoff()
	if handoff.DigestVersion != item.Expected.DigestVersion || handoff.DurationMillis < 0 || handoff.ErrorClass.KnownCause() != item.Expected.ErrorClassBounded || handoff.Queryable {
		t.Fatalf("safe handoff metadata = %+v", handoff)
	}
	encoded, err := json.Marshal(handoff)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range item.RawValues {
		if strings.Contains(string(encoded), raw) {
			t.Fatalf("safe handoff leaked fixture value %q", raw)
		}
	}
}
func TestSharedRecoverySequenceFixtureMatchesOracleAndIndexHandoff(t *testing.T) {
	corpus, base := loadReconciliationFixtures(t)
	item := reconciliationCaseByID(t, corpus, "FX-PROJECTION-RECOVERY-SEQUENCE")
	state := newReconciliationState(t, corpus.ProfileID, domain.ContextNormal, base.Handoff.ProjectionEpoch, base.Handoff.EventSequence, base.Handoff.Records, nil)
	currentEpoch := base.Handoff.ProjectionEpoch

	for index, cycle := range item.Cycles {
		if cycle.RequireSnapshot {
			state.RequireSnapshot()
		}
		if cycle.StartSnapshot != nil {
			start := cycle.StartSnapshot
			outcome, handoff, rejection := state.ApplySnapshot(projection.SnapshotRequest{
				ProfileID: corpus.ProfileID, ContextKind: domain.ContextNormal, Epoch: start.Epoch,
				Revision: 1, Sequence: start.Sequence,
				Records: wireRecords(base.Handoff.Records, corpus.ProfileID, domain.ContextNormal, start.Epoch, start.RecordIDs),
			})
			if rejection != nil || outcome != projection.SnapshotCommitted || handoff.Epoch != start.Epoch {
				t.Fatalf("cycle %d start snapshot outcome=%q handoff=%+v rejection=%v", index+1, outcome, handoff, rejection)
			}
			currentEpoch = start.Epoch
		}

		accepted, rejection := applyFixtureEvent(state, base.Handoff.Records, corpus.ProfileID, domain.ContextNormal, currentEpoch, cycle.AcceptedEvent)
		if rejection != nil || accepted != cycle.Expected.AcceptedOutcome {
			t.Fatalf("cycle %d accepted delta outcome=%q rejection=%v", index+1, accepted, rejection)
		}
		dropped, rejection := applyFixtureEvent(state, base.Handoff.Records, corpus.ProfileID, domain.ContextNormal, currentEpoch, cycle.DroppedEvent)
		if rejection == nil || dropped != cycle.Expected.DroppedOutcome {
			t.Fatalf("cycle %d dropped delta outcome=%q rejection=%v", index+1, dropped, rejection)
		}
		resync := state.State().Handoff()
		committed := state.State().Committed()
		if !cycle.Expected.ResyncRequired || resync.Status != projection.StatusResyncRequired || resync.Queryable || committed.Revision != cycle.Expected.ResyncRevision || committed.Sequence != cycle.Expected.ResyncSequence || !reflect.DeepEqual(identityTabIDs(committed), cycle.Expected.ResyncIdentityIDs) {
			t.Fatalf("cycle %d did not quarantine the last complete state: handoff=%+v ids=%v", index+1, resync, identityTabIDs(committed))
		}

		recovery := cycle.RecoverySnapshot
		outcome, handoff, rejection := state.ApplySnapshot(projection.SnapshotRequest{
			ProfileID: corpus.ProfileID, ContextKind: domain.ContextNormal, Epoch: recovery.Epoch,
			Revision: 1, Sequence: recovery.Sequence,
			Records: wireRecords(base.Handoff.Records, corpus.ProfileID, domain.ContextNormal, recovery.Epoch, recovery.RecordIDs),
		})
		if rejection != nil || outcome != projection.SnapshotCommitted || handoff.Status.String() != cycle.Expected.HandoffStatus || handoff.Epoch != recovery.Epoch || !reflect.DeepEqual(handoff.Identities, state.State().Committed().Identities()) {
			t.Fatalf("cycle %d recovery handoff outcome=%q handoff=%+v rejection=%v", index+1, outcome, handoff, rejection)
		}
		currentEpoch = recovery.Epoch
		if !reflect.DeepEqual(identityTabIDs(state.State().Committed()), cycle.Expected.IdentityIDs) || handoff.Count != len(cycle.Expected.IdentityIDs) {
			t.Fatalf("cycle %d recovery IDs=%v want=%v", index+1, identityTabIDs(state.State().Committed()), cycle.Expected.IdentityIDs)
		}
	}

	final := state.State().Handoff()
	ids := identityTabIDs(state.State().Committed())
	if final.Epoch != item.Expected.FinalEpoch || final.Revision != item.Expected.FinalRevision || final.Sequence != item.Expected.FinalSequence || final.Status.String() != item.Expected.HandoffStatus || !reflect.DeepEqual(ids, item.EligibleBrowserIDs) || !reflect.DeepEqual(ids, item.Expected.IdentityIDs) || len(final.Identities) != item.Expected.UniqueIdentityCount {
		t.Fatalf("final committed IDs diverge from browser oracle: handoff=%+v ids=%v oracle=%v", final, ids, item.EligibleBrowserIDs)
	}
}
