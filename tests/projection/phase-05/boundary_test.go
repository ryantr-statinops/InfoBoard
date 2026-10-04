package phase05

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/projection"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

// allocatedSequence is the IP-04 event sequence every host snapshot request
// carries. A T01 record payload states no sequence of its own, so this consumer
// declares the first sequence of the epoch; the four mandatory identity fields
// still come from the fixture alone and are never allocated here.
const allocatedSequence = 1

// fence is the partition and lineage one declared case states. An absent
// mandatory field stays absent: a record or handoff payload decodes a field the
// peer never sent as its zero value, which is exactly what the boundary refuses.
type fence struct {
	profileID   string
	contextKind domain.ContextKind
	epoch       string
	revision    uint64
}

// admission is the bounded result of driving one declared case through the host
// boundary, together with the input the reconciler returned for an admitted case.
type admission struct {
	admitted bool
	cause    projection.Cause
	snapshot projection.SnapshotInput
}

// tally counts what the host boundary admitted and refused across one fixture.
type tally struct {
	admitted int
	refused  int
}

func text(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func count(value any) uint64 {
	switch typed := value.(type) {
	case float64:
		if typed >= 0 {
			return uint64(typed)
		}
	case uint64:
		return typed
	}
	return 0
}

// bindPartition opens the host boundary on the profile/context partition a
// fixture binds. An unbound partition is refused by the reconciler instead of
// being repaired, so a fixture that cannot state its own partition fails here.
func bindPartition(t *testing.T, fixture Fixture) *projection.Reconciler {
	t.Helper()
	profile, err := domain.ParseProfileID(fixture.Input.ProfileID)
	if err != nil {
		t.Fatalf("%s: profile_id: %v", fixture.FixtureID, err)
	}
	reconciler, rejection := projection.New(projection.Partition{ProfileID: profile, ContextKind: domain.ContextKind(fixture.Input.ContextKind)})
	if rejection != nil {
		t.Fatalf("%s: the bound partition was refused: %v", fixture.FixtureID, rejection)
	}
	if reconciler.Partition() != (projection.Partition{ProfileID: profile, ContextKind: domain.ContextKind(fixture.Input.ContextKind)}) {
		t.Fatalf("%s: the reconciler bound another partition", fixture.FixtureID)
	}
	return reconciler
}

// declaredFence reads the mandatory fields one case states. A record case
// declares them on the record itself and an IP-04 handoff case declares them on
// the handoff envelope; neither value is defaulted here.
func declaredFence(t *testing.T, item Case) fence {
	t.Helper()
	header := item.Handoff
	if header == nil {
		header = item.Payload
	}
	if header == nil {
		t.Fatalf("%s states neither a record nor a handoff", item.CaseID)
	}
	return fence{
		profileID:   text(header["profile_id"]),
		contextKind: domain.ContextKind(text(header["context_kind"])),
		epoch:       text(header["projection_epoch"]),
		revision:    count(header["projection_revision"]),
	}
}

// recordPayloads returns the record payloads one case carries: the record itself,
// or the records an IP-04 snapshot handoff states.
func recordPayloads(t *testing.T, item Case) []map[string]any {
	t.Helper()
	if item.Handoff == nil {
		if item.Payload == nil {
			t.Fatalf("%s carries no record payload", item.CaseID)
		}
		return []map[string]any{item.Payload}
	}
	rows, listed := item.Handoff["records"].([]any)
	if !listed {
		return nil
	}
	payloads := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		record, ok := row.(map[string]any)
		if !ok {
			t.Fatalf("%s carries a record payload that is not an object", item.CaseID)
		}
		payloads = append(payloads, record)
	}
	return payloads
}

// declaredRecords decodes the record payloads of one case into the IP-02 records
// the artifact declares and lifts them onto the IP-07 wire records the host
// boundary consumes. The wire record carries a tab identity in its own identity
// fields, so a record whose nested tab identity cannot name a field arrives
// without it instead of being repaired from the record header.
func declaredRecords(t *testing.T, item Case) []protocol.TabRecord {
	t.Helper()
	raw, err := json.Marshal(recordPayloads(t, item))
	if err != nil {
		t.Fatalf("%s: marshal records: %v", item.CaseID, err)
	}
	var rows []domain.EligibleTabRecord
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("%s: decode records: %v", item.CaseID, err)
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
	return records
}

// declare drives one declared case through the real host boundary: its mandatory
// fence through BindHandoff and, when the case states record payloads, those
// records through BindSnapshot. Delta ordering, snapshot authority, and commit are
// later IP-05 tasks, so a case that states no records is a fence-only input here.
func declare(t *testing.T, reconciler *projection.Reconciler, item Case) admission {
	t.Helper()
	declared := declaredFence(t, item)
	if _, rejection := reconciler.BindHandoff(projection.HandoffRequest{
		ProfileID:   declared.profileID,
		ContextKind: declared.contextKind,
		Epoch:       declared.epoch,
		Revision:    declared.revision,
	}); rejection != nil {
		return admission{cause: rejection.Cause}
	}
	if recordPayloads(t, item) == nil {
		return admission{admitted: true}
	}
	input, rejection := reconciler.BindSnapshot(projection.SnapshotRequest{
		ProfileID:   declared.profileID,
		ContextKind: declared.contextKind,
		Epoch:       declared.epoch,
		Revision:    declared.revision,
		Sequence:    allocatedSequence,
		Records:     declaredRecords(t, item),
	})
	if rejection != nil {
		return admission{cause: rejection.Cause}
	}
	return admission{admitted: true, snapshot: input}
}

// assertRefused proves a refused case stays bounded and inert: the cause comes
// from the closed tab-free vocabulary, the bound partition keeps no committed
// lineage or identity, and the partition is not queryable.
func assertRefused(t *testing.T, reconciler *projection.Reconciler, fixture Fixture, item Case, result admission) {
	t.Helper()
	if result.admitted {
		t.Fatalf("%s/%s: the boundary admitted an input that cannot state its identity", fixture.FixtureID, item.CaseID)
	}
	if !result.cause.KnownCause() {
		t.Fatalf("%s/%s: cause %q is outside the closed cause vocabulary", fixture.FixtureID, item.CaseID, result.cause)
	}
	committed := reconciler.State().Committed()
	if committed.Status != projection.StatusUninitialized || committed.Epoch != "" || committed.Revision != 0 || committed.Count() != 0 {
		t.Fatalf("%s/%s: a refused input left committed state %s epoch=%q revision=%d count=%d",
			fixture.FixtureID, item.CaseID, committed.Status, committed.Epoch, committed.Revision, committed.Count())
	}
	handoff := reconciler.State().Handoff()
	if handoff.Queryable || handoff.Status != projection.StatusUninitialized || handoff.Count != 0 {
		t.Fatalf("%s/%s: a refused input published handoff status=%s queryable=%t count=%d",
			fixture.FixtureID, item.CaseID, handoff.Status, handoff.Queryable, handoff.Count)
	}
}

// assertAdmitted proves an accepted case was mapped rather than repaired: the
// reconciler returns the record under the bound partition identity and the
// declared lineage, so no value was invented for the peer.
func assertAdmitted(t *testing.T, reconciler *projection.Reconciler, fixture Fixture, item Case, result admission) {
	t.Helper()
	declared := declaredFence(t, item)
	partition := reconciler.Partition()
	if result.snapshot.Partition != partition {
		t.Fatalf("%s/%s: the record was bound to partition %+v, want %+v", fixture.FixtureID, item.CaseID, result.snapshot.Partition, partition)
	}
	if result.snapshot.Epoch != declared.epoch || result.snapshot.Revision != declared.revision {
		t.Fatalf("%s/%s: lineage epoch=%q revision=%d, want the declared epoch=%q revision=%d",
			fixture.FixtureID, item.CaseID, result.snapshot.Epoch, result.snapshot.Revision, declared.epoch, declared.revision)
	}
	payloads := recordPayloads(t, item)
	if len(result.snapshot.Records) != len(payloads) {
		t.Fatalf("%s/%s: the boundary mapped %d records, want %d", fixture.FixtureID, item.CaseID, len(result.snapshot.Records), len(payloads))
	}
	for index, record := range result.snapshot.Records {
		identity, _ := payloads[index]["tab_identity"].(map[string]any)
		if want := partition.Identity(int64(count(identity["tab_id"]))); record.TabIdentity != want {
			t.Fatalf("%s/%s: record identity %+v, want the declared %+v", fixture.FixtureID, item.CaseID, record.TabIdentity, want)
		}
		if record.ProfileID != partition.ProfileID || record.ContextKind != partition.ContextKind {
			t.Fatalf("%s/%s: record was mapped under %s/%s, want %s/%s",
				fixture.FixtureID, item.CaseID, record.ProfileID, record.ContextKind, partition.ProfileID, partition.ContextKind)
		}
	}
}

// missingField reports the mandatory field a case declares as missing and proves
// the case really does not state it: the partition and fence fields live on the
// record or handoff header, the tab identity fields inside the nested identity.
func missingField(t *testing.T, item Case) (string, bool) {
	t.Helper()
	if item.Cause == nil {
		t.Fatalf("%s declares no boundary cause", item.CaseID)
	}
	name := strings.TrimPrefix(*item.Cause, causeMissing)
	header := item.Handoff
	if header == nil {
		header = item.Payload
	}
	if header == nil {
		t.Fatalf("%s states neither a record nor a handoff", item.CaseID)
	}
	holder, field := header, name
	if nested, found := strings.CutPrefix(name, "tab_identity_"); found {
		identity, ok := header["tab_identity"].(map[string]any)
		if !ok {
			return "tab_identity." + nested, true
		}
		holder, field = identity, nested
	}
	_, stated := holder[field]
	return field, !stated
}

// mandatoryCauses is the closed set of causes that name an absent mandatory
// partition, fence, or tab identity field. IP-05-T01 refuses a record for no other
// reason, so a cause outside this set belongs to a later IP-05 task.
func mandatoryCauses(boundary Boundary) map[string]bool {
	causes := map[string]bool{}
	for _, field := range boundary.MandatoryPartitionFields {
		causes[causeMissing+field] = true
	}
	for _, field := range boundary.MandatoryFenceFields {
		causes[causeMissing+field] = true
	}
	for _, field := range boundary.TabIdentityRequiredFields {
		causes[causeMissing+"tab_identity_"+field] = true
	}
	return causes
}

// replay drives every declared case of one fixture through the host boundary and
// asserts only the IP-05-T01 contract: a case that cannot state its mandatory
// identity is refused with a bounded cause and leaves no committed state, and an
// accepted case carries the bound partition identity and declared lineage.
func replay(t *testing.T, corpus Corpus, fixture Fixture) tally {
	t.Helper()
	reconciler := bindPartition(t, fixture)
	observed := tally{}
	for _, item := range fixture.Input.Cases {
		result := declare(t, reconciler, item)
		switch item.Stage {
		case stageBoundary:
			assertRefused(t, reconciler, fixture, item, result)
			if item.ExpectedOutcome != corpus.Boundary.FailClosedOutcome {
				t.Fatalf("%s/%s: expected outcome %q, want the declared fail-closed outcome %q",
					fixture.FixtureID, item.CaseID, item.ExpectedOutcome, corpus.Boundary.FailClosedOutcome)
			}
			if item.Apply != applyNone {
				t.Fatalf("%s/%s: a refused boundary case must not reach the projection stage", fixture.FixtureID, item.CaseID)
			}
			if !item.SyntheticFallbackProbe {
				t.Fatalf("%s/%s: a refused boundary case must probe the synthetic fallback", fixture.FixtureID, item.CaseID)
			}
			// The artifact's own probe: the same case with every mandatory value
			// blanked and an invalid tab id is refused as well, so no zero-valued
			// identity fallback completes a boundary failure.
			probe := corpus.Boundary.SyntheticCase(item)
			assertRefused(t, reconciler, fixture, item, declare(t, reconciler, probe))
			observed.refused++
		case stageProjection:
			if item.Cause != nil {
				t.Fatalf("%s/%s: a projection-stage case must not declare a boundary cause", fixture.FixtureID, item.CaseID)
			}
			if item.Apply != applyRecord {
				t.Fatalf("%s/%s: projection-stage apply = %q, want %q", fixture.FixtureID, item.CaseID, item.Apply, applyRecord)
			}
			usable := contains(corpus.Boundary.AcceptedOutcomes, item.ExpectedOutcome)
			if usable != result.admitted {
				t.Fatalf("%s/%s: the artifact declares this record %s, the boundary reported admitted=%t cause=%q",
					fixture.FixtureID, item.CaseID, item.ExpectedOutcome, result.admitted, result.cause)
			}
			if usable {
				// A usable record must clear the boundary mapped, not repaired: it
				// comes back under the bound partition identity and declared lineage.
				assertAdmitted(t, reconciler, fixture, item, result)
				observed.admitted++
				continue
			}
			// The artifact declares this record fail-closed, so the boundary must
			// refuse it rather than complete a placeholder identity. IP-02 owns the
			// outcome value the refusal carries.
			if item.ExpectedOutcome != corpus.Boundary.FailClosedOutcome {
				t.Fatalf("%s/%s: expected outcome %q, want the declared fail-closed outcome %q",
					fixture.FixtureID, item.CaseID, item.ExpectedOutcome, corpus.Boundary.FailClosedOutcome)
			}
			assertRefused(t, reconciler, fixture, item, result)
			observed.refused++
		default:
			t.Fatalf("%s/%s: stage %q is not declared by the artifact", fixture.FixtureID, item.CaseID, item.Stage)
		}
	}
	return observed
}

func TestProjection_RevisionBoundaryCorpusContract(t *testing.T) {
	corpus := LoadCorpus(t)
	if corpus.SchemaVersion != 1 || corpus.Phase != "IP-05" || corpus.Artifact != "projection-boundary" {
		t.Fatalf("artifact header: version=%d phase=%q artifact=%q", corpus.SchemaVersion, corpus.Phase, corpus.Artifact)
	}
	if corpus.OwnerTask != "IP-05-T01" {
		t.Fatalf("owner_task = %q, want the IP-05-T01 slice", corpus.OwnerTask)
	}
	if len(corpus.SharedConsumers) != 2 || corpus.SharedConsumers[0] != "go" || corpus.SharedConsumers[1] != "node" {
		t.Fatalf("shared_consumers = %v, want [go node]", corpus.SharedConsumers)
	}
	for _, path := range corpus.DeclaredPaths() {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("contract source %s is absent: %v", path, err)
		}
	}
	if len(corpus.Fixtures) != len(RequiredFixtureIDs) {
		t.Fatalf("fixture count = %d, want %d", len(corpus.Fixtures), len(RequiredFixtureIDs))
	}
	declared := map[string]bool{}
	for _, id := range RequiredFixtureIDs {
		declared[id] = true
	}
	seen := map[string]int{}
	for _, fixture := range corpus.Fixtures {
		seen[fixture.FixtureID]++
		if !declared[fixture.FixtureID] {
			t.Fatalf("fixture %s is outside the IP-05-T01 obligation", fixture.FixtureID)
		}
		if fixture.OwnerTask != corpus.OwnerTask || len(fixture.RequirementIDs) == 0 || strings.TrimSpace(fixture.Title) == "" {
			t.Fatalf("%s has no owner task, requirement, or title", fixture.FixtureID)
		}
		if len(fixture.Expected.PrivacyAssertions) == 0 {
			t.Fatalf("%s declares no privacy assertion", fixture.FixtureID)
		}
		if len(fixture.Input.Cases) == 0 {
			t.Fatalf("%s declares no case", fixture.FixtureID)
		}
		for _, requirement := range fixture.RequirementIDs {
			if !contains(corpus.RequirementIDs, requirement) {
				t.Fatalf("%s references requirement %s outside the artifact", fixture.FixtureID, requirement)
			}
		}
	}
	for _, id := range RequiredFixtureIDs {
		if seen[id] != 1 {
			t.Fatalf("fixture %s appears %d times", id, seen[id])
		}
	}
	for _, reserved := range corpus.ReservedIDs {
		if seen[reserved] != 0 {
			t.Fatalf("fixture %s is reserved for a later IP-05 task", reserved)
		}
	}
	boundary := corpus.Boundary
	if boundary.SyntheticFallbackAllowed {
		t.Fatal("the artifact permits a synthetic identity fallback")
	}
	if len(boundary.AcceptedOutcomes) == 0 || strings.TrimSpace(boundary.FailClosedOutcome) == "" {
		t.Fatal("the artifact declares no accepted outcome or fail-closed outcome")
	}
	mandatory := append(append([]string{}, boundary.MandatoryPartitionFields...), boundary.MandatoryFenceFields...)
	if len(mandatory) != len(boundary.MandatoryPartitionFields)+len(boundary.MandatoryFenceFields) {
		t.Fatal("a mandatory field is declared twice")
	}
	for _, field := range mandatory {
		if !contains(boundary.SnapshotRecordRequiredFields, field) {
			t.Fatalf("mandatory field %s is absent from the record requirement set", field)
		}
		if _, sourced := boundary.IdentitySources[field]; !sourced {
			t.Fatalf("mandatory field %s declares no identity source", field)
		}
		if contains(boundary.MandatoryPartitionFields, field) {
			if _, allocated := (FixtureInput{}).Allocated(field); !allocated {
				t.Fatalf("mandatory partition field %s has no bound identity value to compare against", field)
			}
		}
	}
	if len(boundary.IdentitySources) != len(mandatory) {
		t.Fatal("an identity source is declared for a field that is not mandatory")
	}
	for _, field := range boundary.MandatoryPartitionFields {
		if !contains(boundary.TabIdentityRequiredFields, field) {
			t.Fatalf("tab identity does not require the mandatory partition field %s", field)
		}
	}
	handoff := boundary.IP04HandoffBoundary
	if handoff.Owner != "IP-04" || handoff.Consumer != "IP-05" {
		t.Fatalf("handoff boundary owner=%q consumer=%q", handoff.Owner, handoff.Consumer)
	}
	for _, kind := range []string{"snapshot", "delta"} {
		fields := handoff.RequiredHandoffFields[kind]
		for _, field := range mandatory {
			if !contains(fields, field) {
				t.Fatalf("%s handoff does not require the mandatory field %s", kind, field)
			}
		}
	}
	if len(handoff.RequiredHandoffFields["status"]) != 1 || !contains(handoff.RequiredHandoffFields["status"], "kind") {
		t.Fatal("a status handoff must require only its kind; every other identity field is unavailable before allocation")
	}
	for _, kind := range handoff.HandoffKinds {
		mapped, known := handoff.HandoffKindMap[kind]
		if !known {
			t.Fatalf("handoff kind %s has no declared projection message kind", kind)
		}
		if !contains(handoff.MessageKinds, mapped) {
			t.Fatalf("handoff kind %s maps to %s, which is not a declared message kind", kind, mapped)
		}
	}
	for _, kind := range handoff.MessageKinds {
		if len(handoff.RequiredHandoffFields[kind]) == 0 {
			t.Fatalf("message kind %s declares no required handoff field", kind)
		}
	}
}

// TestProjection_RevisionMandatoryIdentityFields drives the record fixture that
// declares one case per mandatory field through the host reconciler. Each case
// really omits the field its cause names, the reconciler refuses it with that
// field's bounded cause, and the accepted contrast states every mandatory field
// and is mapped onto the bound partition identity unchanged.
func TestProjection_RevisionMandatoryIdentityFields(t *testing.T) {
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, "FX-PROJECTION-BOUNDARY-REQUIRED-FIELDS")
	replay(t, corpus, fixture)
	reconciler := bindPartition(t, fixture)
	allowed := mandatoryCauses(corpus.Boundary)
	covered := map[string]bool{}
	refused := 0
	for _, item := range fixture.Input.Cases {
		if item.Stage != stageBoundary {
			continue
		}
		refused++
		field, absent := missingField(t, item)
		if !absent {
			t.Fatalf("%s: cause %q names field %q, which the case states", item.CaseID, *item.Cause, field)
		}
		if item.Cause == nil || !allowed[*item.Cause] {
			t.Fatalf("%s: cause is not a missing mandatory identity field", item.CaseID)
		}
		// The reconciler refuses the case with the cause the artifact declares for
		// the field the case really omits.
		if result := declare(t, reconciler, item); result.admitted || string(result.cause) != *item.Cause {
			t.Fatalf("%s: the reconciler reported admitted=%t cause=%q, want the declared cause %q",
				item.CaseID, result.admitted, result.cause, *item.Cause)
		}
		covered[*item.Cause] = true
	}
	if refused != len(fixture.Input.Cases)-1 {
		t.Fatalf("the fixture declares %d refused cases, want every case but the complete record", refused)
	}
	// Every mandatory partition, fence, and tab identity field must have a case that
	// proves the boundary refuses a record omitting it.
	for _, field := range corpus.Boundary.TabIdentityRequiredFields {
		if !covered[causeMissing+"tab_identity_"+field] {
			t.Fatalf("the fixture proves no refusal for tab identity field %s", field)
		}
	}
	for _, field := range append(append([]string{}, corpus.Boundary.MandatoryPartitionFields...), corpus.Boundary.MandatoryFenceFields...) {
		if !covered[causeMissing+field] {
			t.Fatalf("the fixture proves no refusal for mandatory field %s", field)
		}
	}
}

// TestProjection_RevisionRejectsSyntheticIdentityFallback proves a placeholder,
// blank, or zero identity is refused rather than completed by a substitute. Every
// value the fixture declares outside its allocated partition cannot even open a
// partition, every refused case leaves the committed state uninitialized, and the
// accepted contrast carries the allocated identity verbatim.
func TestProjection_RevisionRejectsSyntheticIdentityFallback(t *testing.T) {
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, "FX-PROJECTION-BOUNDARY-SYNTHETIC-IDENTITY")
	observed := replay(t, corpus, fixture)
	reconciler := bindPartition(t, fixture)
	profile, err := domain.ParseProfileID(fixture.Input.ProfileID)
	if err != nil {
		t.Fatalf("%s: profile_id: %v", fixture.FixtureID, err)
	}
	context := domain.ContextKind(fixture.Input.ContextKind)
	profiles, contexts := placeholders(fixture)
	if len(profiles) == 0 || len(contexts) == 0 {
		t.Fatalf("%s declares no placeholder profile and context to refuse", fixture.FixtureID)
	}
	// A partition without a real profile id or with an unknown context is refused
	// rather than repaired, so a placeholder value never becomes a host partition.
	for _, partition := range append([]projection.Partition{{}}, declaredPartitions(profiles, contexts, profile, context)...) {
		if _, rejection := projection.New(partition); rejection == nil || rejection.Cause != projection.CausePartitionUnbound {
			t.Fatalf("partition %+v was accepted instead of refused", partition)
		}
	}
	// Each identity defect class this fixture declares must reach the boundary as a
	// bounded cause naming a mandatory field, never as a repaired identity.
	causes := map[projection.Cause]bool{}
	refused := 0
	for _, item := range fixture.Input.Cases {
		result := declare(t, reconciler, item)
		if result.admitted {
			// Only the complete record clears the host boundary: every value the
			// artifact probes, including a zero revision, is refused here instead of
			// being completed, so no probed identity can be admitted.
			if item.SyntheticFallbackProbe {
				t.Fatalf("%s: an admitted record must not be probed as a fallback", item.CaseID)
			}
			assertAdmitted(t, reconciler, fixture, item, result)
			continue
		}
		refused++
		assertRefused(t, reconciler, fixture, item, result)
		causes[result.cause] = true
	}
	// A blank tab id is the required-fields fixture's case; this fixture probes the
	// fallback by blanking every mandatory value at once.
	for _, cause := range []projection.Cause{
		projection.CauseMissingProfileID,
		projection.CauseMissingContextKind,
		projection.CauseMissingProjectionEpoch,
		projection.CauseMissingProjectionRevision,
		projection.CauseMissingTabIdentityProfileID,
		projection.CauseMissingTabIdentityContextKind,
	} {
		if !causes[cause] {
			t.Fatalf("the fixture proves no refusal with cause %q", cause)
		}
	}
	if refused != observed.refused {
		t.Fatalf("refused %d cases, want the %d the replay observed", refused, observed.refused)
	}
	// The accepted contrast must carry the allocated identity verbatim; otherwise
	// it would be the fallback the other cases refuse.
	for _, item := range fixture.Input.Cases {
		if item.Stage != stageProjection || item.SyntheticFallbackProbe {
			continue
		}
		declared := declaredFence(t, item)
		if declared.profileID != fixture.Input.ProfileID || string(declared.contextKind) != fixture.Input.ContextKind || declared.epoch != fixture.Input.ProjectionEpoch {
			t.Fatalf("%s: the accepted record does not carry the allocated identity", item.CaseID)
		}
	}
}

// TestProjection_RevisionHandoffRequiresAllocatedIdentity drives the IP-04 handoff
// fixture through the host boundary. A handoff that cannot state profile_id,
// context_kind, projection_epoch, and projection_revision is refused with the cause
// the artifact declares for the field it omits, that field is genuinely absent,
// and the accepted handoff states every allocated value and carries its records.
func TestProjection_RevisionHandoffRequiresAllocatedIdentity(t *testing.T) {
	corpus := LoadCorpus(t)
	fixture := corpus.Fixture(t, "FX-PROJECTION-BOUNDARY-HANDOFF-IDENTITY")
	replay(t, corpus, fixture)
	reconciler := bindPartition(t, fixture)
	mapped := corpus.Boundary.IP04HandoffBoundary.HandoffKindMap
	for _, item := range fixture.Input.Cases {
		kind, known := mapped[text(item.Handoff["kind"])]
		if !known {
			t.Fatalf("%s: handoff kind %q is not an IP-04 kind", item.CaseID, text(item.Handoff["kind"]))
		}
		if item.MessageKind != kind {
			t.Fatalf("%s: message_kind = %q, want %q for handoff kind %q", item.CaseID, item.MessageKind, kind, text(item.Handoff["kind"]))
		}
		result := declare(t, reconciler, item)
		if item.Stage == stageProjection {
			if !result.admitted {
				t.Fatalf("%s: an accepted handoff was refused with cause %q", item.CaseID, result.cause)
			}
			for _, field := range corpus.Boundary.MandatoryPartitionFields {
				allocated, ok := fixture.Input.Allocated(field)
				if !ok {
					t.Fatalf("%s: mandatory field %s has no bound identity value", item.CaseID, field)
				}
				if text(item.Handoff[field]) != allocated {
					t.Fatalf("%s: handoff %s = %q, want the allocated value %q", item.CaseID, field, text(item.Handoff[field]), allocated)
				}
			}
			for _, field := range corpus.Boundary.MandatoryFenceFields {
				if _, stated := item.Handoff[field]; !stated {
					t.Fatalf("%s: an accepted handoff must state %s", item.CaseID, field)
				}
			}
			if _, stated := item.Handoff["records"]; !stated {
				t.Fatalf("%s: an accepted handoff must carry its record payload", item.CaseID)
			}
			continue
		}
		// The host fence resolves profile, context, epoch, then revision, which is
		// the order the artifact declares its required handoff fields in, so the
		// refusal names exactly the field the handoff failed to state.
		if result.admitted || string(result.cause) != *item.Cause {
			t.Fatalf("%s: the reconciler reported admitted=%t cause=%q, want the declared cause %q",
				item.CaseID, result.admitted, result.cause, *item.Cause)
		}
		field, absent := missingField(t, item)
		if !absent {
			t.Fatalf("%s: cause %q names field %q, which the handoff states", item.CaseID, *item.Cause, field)
		}
	}
}

// TestProjection_RevisionAdmitsOnlyDeclaredInputs compares what the host boundary
// admitted and refused with the bounded evidence the artifact declares. The
// artifact's outcome_note makes an IP-02 projection outcome the authoritative
// boundary result for a record input, so the declared accepted and rejected case
// counts must match the reconciler exactly. The final lineage and identity count
// are IP-02 reducer outcomes owned by later IP-05 tasks and stay with the
// fixture runner.
func TestProjection_RevisionAdmitsOnlyDeclaredInputs(t *testing.T) {
	corpus := LoadCorpus(t)
	for _, id := range RequiredFixtureIDs {
		fixture := corpus.Fixture(t, id)
		observed := replay(t, corpus, fixture)
		declaredCount(t, fixture, "accepted_cases", observed.admitted)
		declaredCount(t, fixture, "rejected_cases", observed.refused)
	}
}

func declaredCount(t *testing.T, fixture Fixture, signal string, observed int) {
	t.Helper()
	declaration, ok := fixture.Observation(signal)
	if !ok {
		t.Fatalf("%s declares no %s observation", fixture.FixtureID, signal)
	}
	value := count(declaration.Value)
	if int(value) != observed {
		t.Fatalf("%s %s = %d, want %v", fixture.FixtureID, signal, observed, declaration.Value)
	}
}

// placeholders collects the profile and context values the fixture declares
// outside its allocated partition, which are the placeholder identities IP-05-T01
// refuses instead of substituting.
func placeholders(fixture Fixture) ([]string, []string) {
	profiles, contexts := map[string]bool{}, map[string]bool{}
	for _, item := range fixture.Input.Cases {
		header := item.Payload
		if item.Handoff != nil {
			header = item.Handoff
		}
		if header == nil {
			continue
		}
		if value := text(header["profile_id"]); value != "" && value != fixture.Input.ProfileID {
			profiles[value] = true
		}
		if value := text(header["context_kind"]); value != "" && value != fixture.Input.ContextKind {
			contexts[value] = true
		}
	}
	return sorted(profiles), sorted(contexts)
}

func declaredPartitions(profiles []string, contexts []string, profile domain.ProfileID, context domain.ContextKind) []projection.Partition {
	partitions := make([]projection.Partition, 0, len(profiles)+len(contexts))
	for _, value := range profiles {
		partitions = append(partitions, projection.Partition{ProfileID: domain.ProfileID(value), ContextKind: context})
	}
	for _, value := range contexts {
		partitions = append(partitions, projection.Partition{ProfileID: profile, ContextKind: domain.ContextKind(value)})
	}
	return partitions
}

func sorted(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
