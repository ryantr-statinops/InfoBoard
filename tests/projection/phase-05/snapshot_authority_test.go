package phase05

// This file is the Go consumer of the shared IP-05-T02 snapshot-authority corpus.
// Every declared case is driven through the real host boundary in
// host/internal/projection: a complete fenced snapshot of the known current
// partition binds as authority with exactly the declared IP-02 records, a
// deliberately incomplete or unpartitioned handoff never binds authority, and one
// invalid record refuses the whole snapshot with no partial output. The consumer
// asserts only what the real boundary reports; it never restates an authority rule,
// and the producer-side acquisition outcome is the Node consumer's evidence.

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

// The bounded acquisition vocabulary this corpus declares, mapped onto the corpus
// observation that counts each outcome.
var authorityOutcomeSignals = map[string]string{
	"AUTHORITATIVE":     "authoritative_cases",
	"SNAPSHOT_REQUIRED": "snapshot_required_cases",
	"PROFILE_MISMATCH":  "profile_mismatch_cases",
}

// authorityFence is the partition and lineage one declared handoff states. An
// absent mandatory field stays absent, so the boundary reads it as the zero value
// it would receive from a peer that never sent it.
type authorityFence struct {
	profileID   string
	contextKind domain.ContextKind
	epoch       string
	revision    uint64
	sequence    uint64
}

// hostAuthority is the bounded result of driving one declared case through the real
// host boundary. Only a snapshot call can return records, so only a snapshot call can
// return authority.
type hostAuthority struct {
	called    bool
	input     projection.SnapshotInput
	handoff   projection.Handoff
	rejection *projection.Rejection
	records   int
}

// bindAuthority opens the host boundary on one profile/context partition of the
// corpus. An unbound partition is refused by the reconciler instead of repaired, so
// a corpus that cannot state its own partition fails here.
func bindAuthority(t *testing.T, corpus AuthorityCorpus, contextKind string) *projection.Reconciler {
	t.Helper()
	profile, err := domain.ParseProfileID(corpus.Input.ProfileID)
	if err != nil {
		t.Fatalf("profile_id: %v", err)
	}
	partition := projection.Partition{ProfileID: profile, ContextKind: domain.ContextKind(contextKind)}
	reconciler, rejection := projection.New(partition)
	if rejection != nil {
		t.Fatalf("context %s: the bound partition was refused: %v", contextKind, rejection)
	}
	if reconciler.Partition() != partition {
		t.Fatalf("context %s: the reconciler bound another partition", contextKind)
	}
	return reconciler
}

// authorityFenceOf reads the mandatory fields one declared handoff states.
func authorityFenceOf(t *testing.T, item AuthorityCase) authorityFence {
	t.Helper()
	if item.Handoff == nil {
		t.Fatalf("%s states no handoff", item.CaseID)
	}
	return authorityFence{
		profileID:   text(item.Handoff["profile_id"]),
		contextKind: domain.ContextKind(text(item.Handoff["context_kind"])),
		epoch:       text(item.Handoff["projection_epoch"]),
		revision:    count(item.Handoff["projection_revision"]),
		sequence:    count(item.Handoff["event_sequence"]),
	}
}

// declaredDomainRecords decodes the record payloads one case declares into the IP-02
// records the host boundary must return unchanged. The comparison target is the
// declared data itself, so a mapped record is proven to be the browser record rather
// than a repaired one.
func declaredDomainRecords(t *testing.T, item AuthorityCase) []domain.EligibleTabRecord {
	t.Helper()
	payloads := item.Records()
	if payloads == nil {
		return nil
	}
	raw, err := json.Marshal(payloads)
	if err != nil {
		t.Fatalf("%s: marshal records: %v", item.CaseID, err)
	}
	var rows []domain.EligibleTabRecord
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("%s: decode records: %v", item.CaseID, err)
	}
	return rows
}

// declaredWireRecords lifts the declared records onto the IP-07 wire records and
// routes them through the real IP-07 codec, so a refused record is proven to be one
// the wire contract accepted and the projection boundary still refused.
func declaredWireRecords(t *testing.T, item AuthorityCase) []protocol.TabRecord {
	t.Helper()
	rows := declaredDomainRecords(t, item)
	if rows == nil {
		return nil
	}
	records := make([]protocol.TabRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, protocol.TabRecord{
			ProfileID:          string(row.TabIdentity.ProfileID),
			ContextKind:        row.TabIdentity.ContextKind,
			TabID:              row.TabIdentity.TabID,
			WindowID:           row.WindowID,
			GroupID:            row.GroupID,
			TitleDisplay:       row.TitleDisplay,
			TitleSearch:        row.TitleSearch,
			URLSearch:          row.URLSearch,
			URLDisplay:         row.URLDisplay,
			DomainDisplay:      row.DomainDisplay,
			DomainSearch:       row.DomainSearch,
			WindowLabelDisplay: row.WindowLabelDisplay,
			WindowLabelSearch:  row.WindowLabelSearch,
			GroupLabelDisplay:  row.GroupLabelDisplay,
			GroupLabelSearch:   row.GroupLabelSearch,
			Pinned:             row.Pinned,
			Active:             row.Active,
			Eligible:           row.Eligible,
			ObservedAt:         row.ObservedAt,
			ProjectionEpoch:    row.ProjectionEpoch,
			ProjectionRevision: row.ProjectionRevision,
		})
	}
	payload, failure := protocol.PayloadOf(protocol.Snapshot{Tabs: records, Sequence: authorityFenceOf(t, item).sequence})
	if failure != nil {
		t.Fatalf("%s: encode the IP-07 snapshot payload: %v", item.CaseID, failure)
	}
	if item.RecordExpansion != nil {
		// The IP-07 decoder owns its own snapshot_records bound and refuses an
		// over-limit payload before the projection boundary is reached, so an oversized
		// read is stopped at the wire and never arrives. This consumer proves that gate
		// and then hands the same records to the boundary, because the boundary's own
		// pre-staging gate is the rule IP-05-T02 owns.
		if _, failure := protocol.DecodeSnapshot(payload, protocol.DefaultLimits()); failure == nil {
			t.Fatalf("%s: the IP-07 decoder accepted %d records past its bound", item.CaseID, len(records))
		} else if failure.Code != protocol.CodePayloadLimit {
			t.Fatalf("%s: the IP-07 decoder refused an over-limit payload with %q, want %q",
				item.CaseID, failure.Code, protocol.CodePayloadLimit)
		}
		return records
	}
	snapshot, failure := protocol.DecodeSnapshot(payload, protocol.DefaultLimits())
	if failure != nil {
		t.Fatalf("%s: the IP-07 decoder refused a payload this case needs: %v", item.CaseID, failure)
	}
	return snapshot.Tabs
}

// bind drives one declared case through the real host boundary. A declared snapshot
// call binds the snapshot payload; any other declared call binds the published
// handoff, which can never return records.
func bind(t *testing.T, reconciler *projection.Reconciler, item AuthorityCase) hostAuthority {
	t.Helper()
	declared := authorityFenceOf(t, item)
	result := hostAuthority{called: true}
	if item.HostSnapshotCall() {
		input, rejection := reconciler.BindSnapshot(projection.SnapshotRequest{
			ProfileID:   declared.profileID,
			ContextKind: declared.contextKind,
			Epoch:       declared.epoch,
			Revision:    declared.revision,
			Sequence:    declared.sequence,
			Records:     declaredWireRecords(t, item),
		})
		result.input, result.rejection = input, rejection
		result.records = len(input.Records)
		return result
	}
	handoff, rejection := reconciler.BindHandoff(projection.HandoffRequest{
		ProfileID:   declared.profileID,
		ContextKind: declared.contextKind,
		Epoch:       declared.epoch,
		Revision:    declared.revision,
	})
	result.handoff, result.rejection = handoff, rejection
	result.records = handoff.Count
	return result
}

// assertRefusal proves the boundary refused a case with exactly the bounded result
// the corpus declares: the cause, the wire code, the resync marker, and the refused
// position all come from the real rejection, and the failure text carries no record
// value.
func assertRefusal(t *testing.T, item AuthorityCase, result hostAuthority) {
	t.Helper()
	declared := item.Expected.Host
	if !result.called || result.rejection == nil {
		t.Fatalf("%s: the boundary admitted a case declared %s", item.CaseID, declared.Validation)
	}
	if declared.Validation != validationRefused {
		t.Fatalf("%s: the case declares %s, so a refusal proves nothing", item.CaseID, declared.Validation)
	}
	rejection := result.rejection
	if !rejection.Cause.KnownCause() {
		t.Fatalf("%s: cause %q is outside the closed cause vocabulary", item.CaseID, rejection.Cause)
	}
	if declared.Cause == nil || string(rejection.Cause) != *declared.Cause {
		t.Fatalf("%s: cause %q, want the declared %v", item.CaseID, rejection.Cause, declared.Cause)
	}
	if declared.Code == nil || string(rejection.Code) != *declared.Code {
		t.Fatalf("%s: code %q, want the declared %v", item.CaseID, rejection.Code, declared.Code)
	}
	observed := ""
	if rejection.Resync != "" {
		observed = string(rejection.Resync)
		if !rejection.Resync.KnownReason() {
			t.Fatalf("%s: resync %q is outside the closed resync vocabulary", item.CaseID, rejection.Resync)
		}
	}
	declaredResync := ""
	if declared.Resync != nil {
		declaredResync = *declared.Resync
	}
	if observed != declaredResync {
		t.Fatalf("%s: resync %q, want the declared %q", item.CaseID, observed, declaredResync)
	}
	if rejection.ResyncRequired() != (declaredResync != "") {
		t.Fatalf("%s: ResyncRequired()=%t disagrees with the declared resync %q", item.CaseID, rejection.ResyncRequired(), declaredResync)
	}
	if declared.Index == nil || rejection.Index != *declared.Index {
		t.Fatalf("%s: refused position %d, want the declared %v", item.CaseID, rejection.Index, declared.Index)
	}
	// The rendered failure is the only text the runtime may log, so it must carry
	// closed-vocabulary values only.
	if rendered := rejection.Error(); rendered != "projection rejected: "+string(rejection.Code)+" ("+string(rejection.Cause)+")" {
		t.Fatalf("%s: rendered rejection %q carries more than its code and cause", item.CaseID, rendered)
	}
	for _, value := range declaredRecordValues(item) {
		if strings.Contains(rejection.Error(), value) {
			t.Fatalf("%s: the rendered rejection leaks a declared record value", item.CaseID)
		}
	}
}

// declaredRecordValues collects the declared text fields of a case, so a bounded
// failure can be proven to carry no title, URL, or domain value.
func declaredRecordValues(item AuthorityCase) []string {
	seen := map[string]bool{}
	values := []string{}
	for _, record := range item.Records() {
		for _, field := range []string{"title_display", "title_search", "url_display", "url_search", "domain_display", "domain_search"} {
			value := text(record[field])
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			values = append(values, value)
		}
	}
	return values
}

// assertSnapshotAuthority proves a declared authority case was bound and mapped: the
// partition, lineage, and record set are exactly what the handoff declared, so no
// field, identity, or revision was repaired on the way in.
func assertSnapshotAuthority(t *testing.T, reconciler *projection.Reconciler, item AuthorityCase, result hostAuthority) {
	t.Helper()
	declared := authorityFenceOf(t, item)
	if !result.called || result.rejection != nil {
		t.Fatalf("%s: the boundary refused an authoritative snapshot: %v", item.CaseID, result.rejection)
	}
	if item.Expected.Host.Validation != validationAdmitted {
		t.Fatalf("%s: the case declares %s, so an admission proves nothing", item.CaseID, item.Expected.Host.Validation)
	}
	input := result.input
	if input.Partition != reconciler.Partition() {
		t.Fatalf("%s: the snapshot was bound to partition %+v, want %+v", item.CaseID, input.Partition, reconciler.Partition())
	}
	if input.Epoch != declared.epoch || input.Revision != declared.revision || input.Sequence != declared.sequence {
		t.Fatalf("%s: lineage epoch=%q revision=%d sequence=%d, want the declared epoch=%q revision=%d sequence=%d",
			item.CaseID, input.Epoch, input.Revision, input.Sequence, declared.epoch, declared.revision, declared.sequence)
	}
	want := declaredDomainRecords(t, item)
	if len(input.Records) != len(want) || item.Expected.Host.Records != len(input.Records) {
		t.Fatalf("%s: the boundary bound %d records, want the declared %d", item.CaseID, len(input.Records), len(want))
	}
	for index, record := range input.Records {
		if !reflect.DeepEqual(record, want[index]) {
			t.Fatalf("%s: the boundary mapped %+v, want the declared %+v", item.CaseID, record, want[index])
		}
		if record.ProfileID != reconciler.Partition().ProfileID || record.ContextKind != reconciler.Partition().ContextKind {
			t.Fatalf("%s: a record was bound under %s/%s, want the partition identity", item.CaseID, record.ProfileID, record.ContextKind)
		}
	}
}

// assertHandoffNotAuthority proves a declared non-snapshot call bound nothing that
// could be served: the published view stays uninitialized, empty, and unqueryable.
func assertHandoffNotAuthority(t *testing.T, item AuthorityCase, result hostAuthority) {
	t.Helper()
	if !result.called || result.rejection != nil {
		t.Fatalf("%s: the boundary refused a handoff the case declares admitted: %v", item.CaseID, result.rejection)
	}
	handoff := result.handoff
	if handoff.Queryable || handoff.Count != 0 || len(handoff.Identities) != 0 {
		t.Fatalf("%s: a non-snapshot handoff published count=%d identities=%v queryable=%t",
			item.CaseID, handoff.Count, handoff.Identities, handoff.Queryable)
	}
	if handoff.Status != projection.StatusUninitialized || handoff.Epoch != "" || handoff.Revision != 0 {
		t.Fatalf("%s: a non-snapshot handoff published status=%s epoch=%q revision=%d",
			item.CaseID, handoff.Status, handoff.Epoch, handoff.Revision)
	}
}

// assertNoCommittedLineage proves the boundary published nothing: authority is not
// commit, so a validated snapshot and every refusal leave the partition exactly as
// uninitialized as they found it.
func assertNoCommittedLineage(t *testing.T, reconciler *projection.Reconciler, corpus AuthorityCorpus, item AuthorityCase) {
	t.Helper()
	committed := reconciler.State().Committed()
	observed := CommittedState{
		Status:    string(committed.Status),
		Epoch:     committed.Epoch,
		Revision:  committed.Revision,
		Count:     committed.Count(),
		Queryable: reconciler.State().Handoff().Queryable,
	}
	if observed != corpus.Expected.Invariants.HostCommittedState {
		t.Fatalf("%s: the corpus declares committed %+v, the boundary published %+v",
			item.CaseID, corpus.Expected.Invariants.HostCommittedState, observed)
	}
	if counters := reconciler.State().Counters(); counters != (projection.Counters{}) {
		t.Fatalf("%s: the boundary moved projection counters %+v", item.CaseID, counters)
	}
	if reconciler.State().Partition() != reconciler.Partition() {
		t.Fatalf("%s: the partition identity changed", item.CaseID)
	}
}

// driveCase proves the declared host result of one case and returns what the real
// boundary observed.
func driveCase(t *testing.T, corpus AuthorityCorpus, reconciler *projection.Reconciler, item AuthorityCase) hostAuthority {
	t.Helper()
	result := bind(t, reconciler, item)
	if result.rejection != nil {
		assertRefusal(t, item, result)
	} else if item.HostSnapshotCall() {
		assertSnapshotAuthority(t, reconciler, item, result)
	} else {
		assertHandoffNotAuthority(t, item, result)
	}
	assertNoCommittedLineage(t, reconciler, corpus, item)
	return result
}

// assertNoPartialInput proves a refused snapshot returned nothing at all, so the
// valid records that preceded the invalid one cannot reach a later commit.
func assertNoPartialInput(t *testing.T, item AuthorityCase, result hostAuthority) {
	t.Helper()
	input := result.input
	if len(input.Records) != 0 || input.Epoch != "" || input.Revision != 0 || input.Sequence != 0 {
		t.Fatalf("%s: a refused snapshot returned epoch=%q revision=%d sequence=%d records=%d",
			item.CaseID, input.Epoch, input.Revision, input.Sequence, len(input.Records))
	}
	if input.Partition != (projection.Partition{}) {
		t.Fatalf("%s: a refused snapshot returned partition %+v", item.CaseID, input.Partition)
	}
}

// authorityDeclaredCount compares one declared corpus observation with what a
// consumer observed from a real boundary.
func authorityDeclaredCount(t *testing.T, corpus AuthorityCorpus, signal string, observed int) {
	t.Helper()
	declaration, ok := corpus.Expected.Observation(signal)
	if !ok {
		t.Fatalf("the corpus declares no %s observation", signal)
	}
	if int(count(declaration.Value)) != observed {
		t.Fatalf("%s = %v, want the observed %d", signal, declaration.Value, observed)
	}
}

// TestProjection_SnapshotAuthorityCorpusContract proves the artifact is the IP-05-T02
// slice, that it claims the reserved fixture ID without taking the boundary contract
// away from IP-05-T01, and that it defers every clause a later IP-05 task owns.
func TestProjection_SnapshotAuthorityCorpusContract(t *testing.T) {
	corpus := LoadSnapshotAuthority(t)
	if corpus.SchemaVersion != 1 || corpus.Phase != "IP-05" || corpus.Artifact != "projection-snapshot-authority" {
		t.Fatalf("artifact header: version=%d phase=%q artifact=%q", corpus.SchemaVersion, corpus.Phase, corpus.Artifact)
	}
	if corpus.OwnerTask != "IP-05-T02" {
		t.Fatalf("owner_task = %q, want the IP-05-T02 slice", corpus.OwnerTask)
	}
	if corpus.FixtureID != SnapshotAuthorityFixtureID {
		t.Fatalf("fixture_id = %q, want %q", corpus.FixtureID, SnapshotAuthorityFixtureID)
	}
	if len(corpus.SharedConsumers) != 2 || corpus.SharedConsumers[0] != "go" || corpus.SharedConsumers[1] != "node" {
		t.Fatalf("shared_consumers = %v, want [go node]", corpus.SharedConsumers)
	}
	if len(corpus.RequirementIDs) == 0 || strings.TrimSpace(corpus.Note) == "" || strings.TrimSpace(corpus.OutcomeNote) == "" {
		t.Fatal("the corpus declares no requirement, note, or outcome note")
	}
	for _, path := range corpus.ContractPaths() {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("contract source %s is absent: %v", path, err)
		}
	}
	if len(corpus.Expected.PrivacyAssertions) == 0 {
		t.Fatal("the corpus declares no privacy assertion")
	}
	if len(corpus.Cases) == 0 {
		t.Fatal("the corpus declares no case")
	}
	if len(corpus.Expected.Invariants.HostCommittedState.Status) == 0 ||
		strings.TrimSpace(corpus.Expected.Invariants.Note) == "" {
		t.Fatal("the corpus declares no committed-state invariant or note")
	}
	// The claimed ID must be reserved by the boundary artifact and implemented by
	// none of its fixtures, so no two IP-05 tasks decide the same scenario.
	if len(corpus.Claims.ReservedFixtureIDs) != 1 || corpus.Claims.ReservedFixtureIDs[0] != corpus.FixtureID {
		t.Fatalf("claims = %v, want exactly %q", corpus.Claims.ReservedFixtureIDs, corpus.FixtureID)
	}
	if strings.TrimSpace(corpus.Claims.ClaimNote) == "" {
		t.Fatal("the corpus states no claim note")
	}
	boundary := LoadCorpus(t)
	if filepath.Clean(filepath.Join("..", "..", "..", corpus.Claims.BoundaryArtifact)) != filepath.Clean(BoundaryArtifact) {
		t.Fatalf("the claim names boundary artifact %q, want %q", corpus.Claims.BoundaryArtifact, BoundaryArtifact)
	}
	if !contains(boundary.ReservedIDs, corpus.FixtureID) {
		t.Fatalf("%s is not reserved by the boundary artifact", corpus.FixtureID)
	}
	for _, item := range boundary.Fixtures {
		if item.FixtureID == corpus.FixtureID {
			t.Fatalf("the boundary artifact already implements %s", corpus.FixtureID)
		}
	}

	authority := corpus.Authority
	if authority.AuthoritativeOutcome != outcomeAuthoritative {
		t.Fatalf("authoritative outcome = %q, want %q", authority.AuthoritativeOutcome, outcomeAuthoritative)
	}
	if len(authority.OutcomeVocabulary) != len(authorityOutcomeSignals) || !contains(authority.OutcomeVocabulary, outcomeAuthoritative) {
		t.Fatalf("outcome vocabulary %v does not declare the bounded acquisition vocabulary", authority.OutcomeVocabulary)
	}
	for outcome := range authorityOutcomeSignals {
		if !contains(authority.OutcomeVocabulary, outcome) {
			t.Fatalf("outcome vocabulary %v omits %s", authority.OutcomeVocabulary, outcome)
		}
	}
	if len(authority.RequiredConditions) == 0 || len(authority.RecordPreconditions) == 0 || len(authority.RejectReasonVocabulary) == 0 {
		t.Fatal("the corpus declares no authority condition, record precondition, or reason vocabulary")
	}
	for _, condition := range authority.RequiredConditions {
		if condition.ID == "" || strings.TrimSpace(condition.Rule) == "" {
			t.Fatalf("condition %+v declares no id or rule", condition)
		}
	}
	for _, precondition := range authority.RecordPreconditions {
		if precondition.Field == "" || precondition.Owner == "" || strings.TrimSpace(precondition.Rule) == "" {
			t.Fatalf("record precondition %+v declares no field, owner, or rule", precondition)
		}
	}
	if len(authority.ResyncSignals) == 0 {
		t.Fatal("the corpus declares no resync signal")
	}
	for signal, effect := range authority.ResyncSignals {
		if signal == "" || strings.TrimSpace(effect) == "" {
			t.Fatalf("resync signal %q declares no effect", signal)
		}
	}
	if strings.TrimSpace(authority.ProducerSignalNote) == "" || strings.TrimSpace(authority.HostFieldNote) == "" {
		t.Fatal("the corpus states no producer signal note or host field note")
	}
	for _, deferred := range authority.DeferredClauses {
		if !strings.HasPrefix(deferred.OwnerTask, "IP-05-T") || deferred.OwnerTask == "IP-05-T02" {
			t.Fatalf("deferred clause %q names owner %q, want a later IP-05 task", deferred.Clause, deferred.OwnerTask)
		}
		if strings.TrimSpace(deferred.Reason) == "" {
			t.Fatalf("deferred clause %q declares no reason", deferred.Clause)
		}
	}
	// Every later task that owns a clause of this scenario must be declared, so the
	// reserved ID cannot be narrowed silently.
	for _, owner := range []string{"IP-05-T03", "IP-05-T04", "IP-05-T05", "IP-05-T07", "IP-05-T09", "IP-05-T12"} {
		declared := false
		for _, deferred := range authority.DeferredClauses {
			if deferred.OwnerTask == owner {
				declared = true
			}
		}
		if !declared {
			t.Fatalf("the corpus defers no clause to %s", owner)
		}
	}
	for _, value := range []string{corpus.Input.ProfileID, corpus.Input.UnboundProfileID, corpus.Input.ProjectionEpoch, corpus.Input.ForeignEpoch} {
		if _, err := domain.ParseProfileID(value); err != nil {
			t.Fatalf("%q is not an allocated identity: %v", value, err)
		}
	}
	if corpus.Input.UnboundProfileID == corpus.Input.ProfileID || corpus.Input.ForeignEpoch == corpus.Input.ProjectionEpoch {
		t.Fatal("the corpus declares no foreign identity to refuse")
	}
	if corpus.Input.ContextKind != contextNormal && corpus.Input.ContextKind != contextPrivate {
		t.Fatalf("input context_kind = %q, want a declared context", corpus.Input.ContextKind)
	}

	signals := map[string]bool{}
	seen := map[string]int{}
	for _, item := range corpus.Cases {
		seen[item.CaseID]++
		if item.CaseID == "" || strings.TrimSpace(item.Note) == "" || item.Handoff == nil {
			t.Fatalf("case %+v declares no id, handoff, or note", item)
		}
		if item.PartitionContext != contextNormal && item.PartitionContext != contextPrivate {
			t.Fatalf("%s: partition_context = %q, want a declared context", item.CaseID, item.PartitionContext)
		}
		if item.Authoritative() && item.PartitionContext != corpus.Input.ContextKind {
			t.Fatalf("%s: authority is declared for a context the corpus does not bind", item.CaseID)
		}
		if !contains(authority.OutcomeVocabulary, item.Acquisition) {
			t.Fatalf("%s: acquisition %q is outside the declared vocabulary", item.CaseID, item.Acquisition)
		}
		signals[item.AcquisitionSignal] = true
		declared := item.Expected.Producer
		if declared.Outcome != item.Acquisition || declared.Authoritative != item.Authoritative() {
			t.Fatalf("%s: producer outcome %q disagrees with the declared acquisition %q", item.CaseID, declared.Outcome, item.Acquisition)
		}
		if item.Authoritative() {
			if declared.Reason != nil || declared.ResyncRequired {
				t.Fatalf("%s: an authoritative acquisition declares a reason or resync requirement", item.CaseID)
			}
			if declared.RecordCount != len(item.Records()) || declared.ExplicitlyEmpty != (declared.RecordCount == 0) {
				t.Fatalf("%s: an authoritative acquisition declares record_count=%d explicitly_empty=%t for %d records",
					item.CaseID, declared.RecordCount, declared.ExplicitlyEmpty, len(item.Records()))
			}
			if !declared.Retained || !item.HostSnapshotCall() {
				t.Fatalf("%s: an authoritative acquisition must be retained through a snapshot call", item.CaseID)
			}
			continue
		}
		// The whole point of a failed, partial, denied, or unavailable read: it
		// publishes no record at all, so it can never be read as an empty projection.
		if declared.Reason == nil || !contains(authority.RejectReasonVocabulary, *declared.Reason) {
			t.Fatalf("%s: a non-authoritative acquisition declares no reason inside the declared vocabulary", item.CaseID)
		}
		if !declared.ResyncRequired || declared.RecordCount != 0 || declared.ExplicitlyEmpty || declared.Retained {
			t.Fatalf("%s: a non-authoritative acquisition must require a fresh read and publish no record", item.CaseID)
		}
		host := item.Expected.Host
		if host.Call != hostCallSnapshot && host.Call != hostCallHandoff {
			t.Fatalf("%s: host call %q is not a declared boundary call", item.CaseID, host.Call)
		}
		if host.Call == hostCallHandoff && item.Handoff["kind"] == hostCallSnapshot {
			t.Fatalf("%s: a snapshot handoff must bind the snapshot call", item.CaseID)
		}
		if host.Validation == validationRefused {
			if host.Cause == nil || host.Code == nil || host.Index == nil || host.Records != 0 {
				t.Fatalf("%s: a refused call declares no cause, code, position, or records", item.CaseID)
			}
			continue
		}
		if host.Cause != nil || host.Code != nil || host.Index != nil {
			t.Fatalf("%s: an admitted call declares a cause, code, or position", item.CaseID)
		}
		if host.Records != len(item.Records()) {
			t.Fatalf("%s: an admitted call declares %d records for %d declared", item.CaseID, host.Records, len(item.Records()))
		}
	}
	for id, occurrences := range seen {
		if occurrences != 1 {
			t.Fatalf("case %s appears %d times", id, occurrences)
		}
	}
	// The corpus must cover both authority clauses and every failure class the task
	// names, or the acceptance evidence is incomplete.
	for _, signal := range []string{"completed", "denied", "failed", "unsupported", "incomplete", "disconnect"} {
		if !signals[signal] {
			t.Fatalf("the corpus declares no %s acquisition signal", signal)
		}
	}
	if !signals["completed"] {
		t.Fatal("the corpus declares no completed read")
	}
	declaredReasons := map[string]bool{}
	for _, item := range corpus.Cases {
		if reason := item.Expected.Producer.Reason; reason != nil {
			declaredReasons[*reason] = true
		}
	}
	for _, reason := range []string{
		"missing_records", "missing_context_kind", "missing_projection_epoch", "snapshot_resync_required",
		"non_snapshot_handoff", "unpartitioned_handoff", "invalid_record", "partition_profile_mismatch",
	} {
		if !declaredReasons[reason] {
			t.Fatalf("the corpus proves no refusal with reason %q", reason)
		}
	}
	// Every declared outcome must be represented, so the evidence cannot claim
	// authority coverage while proving none.
	tallies := map[string]int{}
	for _, item := range corpus.Cases {
		tallies[item.Acquisition]++
	}
	for outcome, signal := range authorityOutcomeSignals {
		if tallies[outcome] == 0 {
			t.Fatalf("the corpus declares no %s case", outcome)
		}
		authorityDeclaredCount(t, corpus, signal, tallies[outcome])
	}
}

// TestProjection_SnapshotAuthorityAcceptsCompletePopulatedSnapshot proves the
// authority clause: a completed read of the known current partition with valid
// records binds with exactly the declared records, and its record set is the browser
// read itself.
func TestProjection_SnapshotAuthorityAcceptsCompletePopulatedSnapshot(t *testing.T) {
	corpus := LoadSnapshotAuthority(t)
	item := corpus.AuthorityCaseByID(t, "complete_populated_snapshot")
	reconciler := bindAuthority(t, corpus, item.PartitionContext)
	result := driveCase(t, corpus, reconciler, item)
	if !item.Authoritative() || result.rejection != nil {
		t.Fatalf("%s is not declared authoritative or was refused", item.CaseID)
	}
	records := result.input.Records
	declared := corpus.Expected.Invariants.ProducerRetainedRecords.AuthoritativeCase
	if len(records) != declared {
		t.Fatalf("the corpus declares %d authoritative records, the boundary bound %d", declared, len(records))
	}
	identities := map[int64]bool{}
	for _, record := range records {
		if identities[record.TabIdentity.TabID] {
			t.Fatalf("%s: the bound snapshot repeats tab identity %d", item.CaseID, record.TabIdentity.TabID)
		}
		identities[record.TabIdentity.TabID] = true
	}
	for _, payload := range item.Records() {
		identity, ok := payload["tab_identity"].(map[string]any)
		if !ok {
			t.Fatalf("%s: a declared record states no tab identity", item.CaseID)
		}
		if !identities[int64(count(identity["tab_id"]))] {
			t.Fatalf("%s: the bound snapshot is missing the declared tab %v", item.CaseID, identity["tab_id"])
		}
	}
}

// TestProjection_SnapshotAuthorityAcceptsExplicitEmptySnapshot proves the
// intentional-empty clause: a completed read that declares an empty record list is
// authority for the partition, so the projection can converge on no eligible tab
// instead of waiting forever.
func TestProjection_SnapshotAuthorityAcceptsExplicitEmptySnapshot(t *testing.T) {
	corpus := LoadSnapshotAuthority(t)
	item := corpus.AuthorityCaseByID(t, "successful_explicit_empty_snapshot")
	if item.Records() == nil {
		t.Fatal("the case must declare its record list explicitly, even when it is empty")
	}
	if len(item.Records()) != 0 || !item.Authoritative() {
		t.Fatalf("%s must declare an authoritative empty read, got %d records", item.CaseID, len(item.Records()))
	}
	if !item.Expected.Producer.ExplicitlyEmpty {
		t.Fatal("the case must declare that its read is explicitly empty")
	}
	reconciler := bindAuthority(t, corpus, item.PartitionContext)
	result := driveCase(t, corpus, reconciler, item)
	if result.rejection != nil {
		t.Fatalf("%s: a completed empty read was refused: %v", item.CaseID, result.rejection)
	}
	if len(result.input.Records) != 0 {
		t.Fatalf("%s: an empty read bound %d records", item.CaseID, len(result.input.Records))
	}
	if result.input.Epoch != corpus.Input.ProjectionEpoch || result.input.Revision == 0 {
		t.Fatalf("%s: an empty read lost its fence: epoch=%q revision=%d", item.CaseID, result.input.Epoch, result.input.Revision)
	}
	explicitlyEmpty := 0
	for _, entry := range corpus.Cases {
		if entry.Expected.Producer.ExplicitlyEmpty {
			explicitlyEmpty++
		}
	}
	authorityDeclaredCount(t, corpus, "explicitly_empty_cases", explicitlyEmpty)
	// A snapshot that declares no record list at all is a different case: only an
	// explicit list may mean the browser reported no eligible tab.
	absent := corpus.AuthorityCaseByID(t, "snapshot_without_record_list")
	if absent.Records() != nil {
		t.Fatal("the no-record-list case must declare no records field")
	}
	if absent.Authoritative() || absent.Expected.Producer.Reason == nil || *absent.Expected.Producer.Reason != "missing_records" {
		t.Fatalf("%s must prove that an undeclared record list is not an empty read", absent.CaseID)
	}
}

// TestProjection_SnapshotAuthorityWithholdsAuthorityOnIncompleteAcquisition proves
// the withheld clause: a denied, failed, partial, unavailable, or resync-required
// handoff binds nothing the host could serve as a projection, and a snapshot that
// contradicts the bound partition is refused as a mismatch instead of an empty read.
func TestProjection_SnapshotAuthorityWithholdsAuthorityOnIncompleteAcquisition(t *testing.T) {
	corpus := LoadSnapshotAuthority(t)
	withheld := 0
	for _, item := range corpus.Cases {
		if item.Authoritative() {
			continue
		}
		withheld++
		reconciler := bindAuthority(t, corpus, item.PartitionContext)
		result := driveCase(t, corpus, reconciler, item)
		if result.records != 0 {
			t.Fatalf("%s: a non-authoritative read published %d records", item.CaseID, result.records)
		}
		if result.rejection != nil {
			assertNoPartialInput(t, item, result)
		}
	}
	if withheld == 0 {
		t.Fatal("the corpus withholds no authority case")
	}
}

// TestProjection_SnapshotAuthorityRejectsInvalidRecordsWithoutPartialOutput proves
// the invalid-record clause: one unusable record refuses the whole snapshot at its
// own position, and neither the boundary nor its rejection can publish the valid
// records that preceded it.
func TestProjection_SnapshotAuthorityRejectsInvalidRecordsWithoutPartialOutput(t *testing.T) {
	corpus := LoadSnapshotAuthority(t)
	covered := map[string]bool{}
	refusals := 0
	for _, item := range corpus.Cases {
		if !strings.HasPrefix(item.CaseID, "invalid_record_") {
			continue
		}
		refusals++
		records := item.Records()
		if len(records) < 2 {
			t.Fatalf("%s must declare a valid record before the invalid one", item.CaseID)
		}
		// The refusal has to come from the later record: the first record is a
		// complete IP-02 record of the declared fence, so a whole-input failure could
		// not be attributed to it.
		first := records[0]
		identity, ok := first["tab_identity"].(map[string]any)
		if !ok || first["eligible"] != true || first["projection_epoch"] != corpus.Input.ProjectionEpoch {
			t.Fatalf("%s: the leading record is not a complete record of the declared fence", item.CaseID)
		}
		if _, stated := identity["tab_id"]; !stated {
			t.Fatalf("%s: the leading record states no tab id", item.CaseID)
		}
		if text(first["projection_epoch"]) != text(item.Handoff["projection_epoch"]) {
			t.Fatalf("%s: the leading record does not repeat the handoff epoch", item.CaseID)
		}
		reconciler := bindAuthority(t, corpus, item.PartitionContext)
		result := driveCase(t, corpus, reconciler, item)
		if result.rejection == nil {
			t.Fatalf("%s: an invalid record was accepted", item.CaseID)
		}
		assertNoPartialInput(t, item, result)
		if result.rejection.Index != 1 {
			t.Fatalf("%s: refused position %d, want the invalid second record", item.CaseID, result.rejection.Index)
		}
		switch result.rejection.Cause {
		case projection.CauseMismatchedProjectionEpoch, projection.CauseMissingTabIdentityTabID,
			projection.CauseInvalidRecord, projection.CauseMismatchedProfileID:
			covered[string(result.rejection.Cause)] = true
		default:
			t.Fatalf("%s: cause %q is not an IP-02 record-contract refusal", item.CaseID, result.rejection.Cause)
		}
	}
	if refusals == 0 {
		t.Fatal("the corpus proves no invalid-record refusal")
	}
	// Every invalid-record class this corpus declares must reach the real boundary as
	// its own bounded cause.
	for _, cause := range []projection.Cause{
		projection.CauseMismatchedProjectionEpoch,
		projection.CauseMissingTabIdentityTabID,
		projection.CauseInvalidRecord,
		projection.CauseMismatchedProfileID,
	} {
		if !covered[string(cause)] {
			t.Fatalf("the corpus proves no refusal with cause %q", cause)
		}
	}
}

// TestProjection_SnapshotAuthorityRefusesOverLimitRecordsBeforeStaging proves the
// host size gate. An over-limit snapshot is refused at the boundary's own record-count
// gate before any record is mapped and before the fence is resolved, so it can neither
// publish a truncated projection nor disturb what a validated read left behind. The
// IP-07 wire gate is proven by the shared record loader, so this test compares the
// declared expansion against the bound the host really enforces and drives both sides
// of it.
func TestProjection_SnapshotAuthorityRefusesOverLimitRecordsBeforeStaging(t *testing.T) {
	corpus := LoadSnapshotAuthority(t)
	if corpus.Authority.MaxSnapshotRecords != protocol.MaxSnapshotRecords {
		t.Fatalf("the corpus declares the bound %d, the host enforces %d",
			corpus.Authority.MaxSnapshotRecords, protocol.MaxSnapshotRecords)
	}
	item := corpus.AuthorityCaseByID(t, "snapshot_over_record_limit")
	expansion := item.RecordExpansion
	if expansion == nil {
		t.Fatalf("%s declares no bounded record expansion", item.CaseID)
	}
	if expansion.IdentityPath != "tab_identity.tab_id" {
		t.Fatalf("%s substitutes %q, which is not the browser tab id", item.CaseID, expansion.IdentityPath)
	}
	if strings.TrimSpace(expansion.ExpansionNote) == "" {
		t.Fatalf("%s states no expansion note", item.CaseID)
	}
	records := item.Records()
	if len(records) <= protocol.MaxSnapshotRecords {
		t.Fatalf("%s materializes %d records, which is not past the real bound %d",
			item.CaseID, len(records), protocol.MaxSnapshotRecords)
	}
	reconciler := bindAuthority(t, corpus, item.PartitionContext)
	result := driveCase(t, corpus, reconciler, item)
	if result.rejection == nil || result.rejection.Cause != projection.CauseBoundsExceeded {
		t.Fatalf("%s: cause %v, want the bounded %q", item.CaseID, result.rejection.Cause, projection.CauseBoundsExceeded)
	}
	assertNoPartialInput(t, item, result)
	// Pre-staging evidence: a record refusal names the declared revision and sequence
	// and a record position, while the size gate refuses the whole input before the
	// fence is resolved, so it carries neither a lineage nor a position.
	if result.rejection.Index != -1 || result.rejection.ExpectedRevision != 0 || result.rejection.ExpectedSequence != 0 {
		t.Fatalf("%s: refused position=%d revision=%d sequence=%d, want a whole-input refusal before the fence",
			item.CaseID, result.rejection.Index, result.rejection.ExpectedRevision, result.rejection.ExpectedSequence)
	}
	// The bound is exact rather than incidental: the very same read at the declared
	// record count is still a completed read of the same fence, so the refusal above is
	// the size gate and not a size this boundary cannot map.
	declared := authorityFenceOf(t, item)
	atLimit, rejection := reconciler.BindSnapshot(projection.SnapshotRequest{
		ProfileID:   declared.profileID,
		ContextKind: declared.contextKind,
		Epoch:       declared.epoch,
		Revision:    declared.revision,
		Sequence:    declared.sequence,
		Records:     declaredWireRecords(t, item)[:protocol.MaxSnapshotRecords],
	})
	if rejection != nil {
		t.Fatalf("%s: a read of exactly %d records was refused: %v", item.CaseID, protocol.MaxSnapshotRecords, rejection)
	}
	if len(atLimit.Records) != protocol.MaxSnapshotRecords {
		t.Fatalf("%s: a read of exactly the bound mapped %d records", item.CaseID, len(atLimit.Records))
	}
	assertNoCommittedLineage(t, reconciler, corpus, item)
}

// TestProjection_SnapshotAuthorityEvidenceMatchesBoundary proves the declared
// evidence equals what the real host boundary produced, and that no case left a
// committed lineage or a partial record set behind.
func TestProjection_SnapshotAuthorityEvidenceMatchesBoundary(t *testing.T) {
	corpus := LoadSnapshotAuthority(t)
	admitted, refused, records, queryable, partial := 0, 0, 0, 0, 0
	for _, item := range corpus.Cases {
		reconciler := bindAuthority(t, corpus, item.PartitionContext)
		result := driveCase(t, corpus, reconciler, item)
		if result.rejection != nil {
			refused++
			if len(result.input.Records) != 0 {
				partial++
			}
			continue
		}
		admitted++
		records += result.records
		if reconciler.State().Handoff().Queryable {
			queryable++
		}
	}
	authorityDeclaredCount(t, corpus, "authoritative_record_count", records)
	authorityDeclaredCount(t, corpus, "non_authoritative_record_count", 0)
	authorityDeclaredCount(t, corpus, "partial_outputs", partial)
	authorityDeclaredCount(t, corpus, "committed_queryable_partitions", queryable)
	if queryable != 0 {
		t.Fatalf("%d partitions became queryable before IP-05-T07 commits a snapshot", queryable)
	}
	if partial != 0 {
		t.Fatalf("%d refused snapshots published records", partial)
	}
	if admitted+refused != len(corpus.Cases) {
		t.Fatalf("the boundary reported %d admitted and %d refused for %d cases", admitted, refused, len(corpus.Cases))
	}
	final, ok := corpus.Expected.Observation("final_outcome")
	if !ok {
		t.Fatal("the corpus declares no final_outcome observation")
	}
	if text(final.Value) != corpus.Cases[len(corpus.Cases)-1].Acquisition {
		t.Fatalf("final_outcome = %v, want %q", final.Value, corpus.Cases[len(corpus.Cases)-1].Acquisition)
	}
}
