package phase05

// This file is the IP-05-T02 half of the shared IP-05 fixture corpus: the
// snapshot-authority artifact the Go host consumer, the Node producer consumer,
// and the Python runner all read. The host consumer drives every declared case
// through the real boundary in host/internal/projection and the Node consumer
// drives the same cases through the real producer boundary in
// extension/src/projection/reconciler.ts, so this loader only reads and shapes the
// artifact and never restates an authority rule. Nothing here asserts canonical
// ordering or digest, revision ordering, delta reduction, atomic replacement, or
// recovery: those are IP-05-T03 through IP-05-T07 and IP-05-T12 with their own
// gates.

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SnapshotAuthorityArtifact is the single shared IP-05-T02 fixture corpus.
const SnapshotAuthorityArtifact = "../../../fixtures/projection/phase-05/snapshot-authority.json"

// SnapshotAuthorityFixtureID is the stable ID IP-05-T01 reserved and IP-05-T02
// claims for the acquisition and validation clauses of the scenario.
const SnapshotAuthorityFixtureID = "FX-PROJECTION-SNAPSHOT-AUTHORITY"

// BoundaryArtifact is the IP-05-T01 corpus that reserved the fixture ID and owns
// the partition, fence, and handoff field contract every case here restates.
const BoundaryArtifact = "../../../fixtures/projection/phase-05/phase-05.json"

const (
	// outcomeAuthoritative is the only acquisition outcome that may carry records.
	outcomeAuthoritative = "AUTHORITATIVE"

	// hostCallSnapshot binds a snapshot payload; hostCallHandoff binds a published
	// status, window, or lifecycle handoff. Only the snapshot call can return
	// records, so only the snapshot call can return authority.
	hostCallSnapshot = "snapshot"
	hostCallHandoff  = "handoff"

	validationAdmitted = "ADMITTED"
	validationRefused  = "REFUSED"

	contextNormal  = "normal"
	contextPrivate = "private"
)

// AuthorityCondition is one declared condition a handoff must satisfy before a
// snapshot may be authority for its partition.
type AuthorityCondition struct {
	ID   string `json:"id"`
	Rule string `json:"rule"`
}

// RecordPrecondition is one IP-02/IP-04 record requirement a snapshot payload must
// satisfy before the boundary treats it as a read of the current profile.
type RecordPrecondition struct {
	Field string `json:"field"`
	Value any    `json:"value"`
	Owner string `json:"owner"`
	Rule  string `json:"rule"`
}

// DeferredClause is one part of the reserved scenario this slice deliberately does
// not decide, with the task that owns it.
type DeferredClause struct {
	Clause    string `json:"clause"`
	OwnerTask string `json:"owner_task"`
	Reason    string `json:"reason"`
}

// Claim records which reserved fixture ID this artifact claims and which artifact
// reserved it, so no two IP-05 tasks can decide the same scenario.
type Claim struct {
	BoundaryArtifact   string   `json:"boundary_artifact"`
	ReservedFixtureIDs []string `json:"reserved_fixture_ids"`
	ClaimNote          string   `json:"claim_note"`
}

// Authority is the declared acquisition contract: the bounded outcome vocabulary,
// the conditions a read must satisfy, the IP-04 resync markers that withhold
// authority, and the clauses later IP-05 tasks own.
type Authority struct {
	OutcomeVocabulary      []string             `json:"outcome_vocabulary"`
	AuthoritativeOutcome   string               `json:"authoritative_outcome"`
	MaxSnapshotRecords     int                  `json:"max_snapshot_records"`
	RequiredConditions     []AuthorityCondition `json:"required_conditions"`
	RecordPreconditions    []RecordPrecondition `json:"record_preconditions"`
	ResyncSignals          map[string]string    `json:"resync_signals"`
	ProducerSignalNote     string               `json:"producer_signal_note"`
	HostFieldNote          string               `json:"host_field_note"`
	HostLimitNote          string               `json:"host_limit_note"`
	RejectReasonVocabulary []string             `json:"reject_reason_vocabulary"`
	DeferredClauses        []DeferredClause     `json:"deferred_clauses"`
}

// ProducerExpectation is what the real producer acquisition must report for one
// case. record_count is the number of records the acquisition published, so a
// non-authoritative acquisition is required to report none.
type ProducerExpectation struct {
	Outcome         string  `json:"outcome"`
	Reason          *string `json:"reason"`
	Authoritative   bool    `json:"authoritative"`
	ResyncRequired  bool    `json:"resync_required"`
	RecordCount     int     `json:"record_count"`
	ExplicitlyEmpty bool    `json:"explicitly_empty"`
	Retained        bool    `json:"retained"`
}

// HostExpectation is what the real host boundary must report for one case. Cause,
// Code, Resync, and Index are the bounded rejection fields; Index is the refused
// record position or -1 when the whole input failed its fence.
type HostExpectation struct {
	Call       string  `json:"call"`
	Validation string  `json:"validation"`
	Cause      *string `json:"cause"`
	Code       *string `json:"code"`
	Resync     *string `json:"resync"`
	Index      *int    `json:"index"`
	Records    int     `json:"records"`
}

// AuthorityExpected is the per-boundary declared result of one case.
type AuthorityExpected struct {
	Producer ProducerExpectation `json:"producer"`
	Host     HostExpectation     `json:"host"`
}

// AuthorityInput binds the corpus to one profile, context, and lineage, plus the
// two foreign identities the refusal cases need.
type AuthorityInput struct {
	ProfileID        string `json:"profile_id"`
	ContextKind      string `json:"context_kind"`
	ProjectionEpoch  string `json:"projection_epoch"`
	UnboundProfileID string `json:"unbound_profile_id"`
	ForeignEpoch     string `json:"foreign_epoch"`
}

// RecordExpansion declares a bounded record list the artifact cannot write as
// fixture text. The template is one record of the declared fence and only the
// browser tab id is substituted, so both consumers materialize the same
// deterministic list and hand it to the real boundary. An over-limit read is the
// declared use: the list exists only so the size gate is what refuses it.
type RecordExpansion struct {
	Template      map[string]any `json:"template"`
	Count         int            `json:"count"`
	IdentityPath  string         `json:"identity_path"`
	IdentityStart int            `json:"identity_start"`
	IdentityStep  int            `json:"identity_step"`
	ExpansionNote string         `json:"expansion_note"`
}

// AuthorityCase is one IP-04 handoff with the authority outcome both real
// boundaries must report for it.
type AuthorityCase struct {
	CaseID            string            `json:"case_id"`
	AcquisitionSignal string            `json:"acquisition_signal"`
	PartitionContext  string            `json:"partition_context"`
	Handoff           map[string]any    `json:"handoff"`
	RecordExpansion   *RecordExpansion  `json:"record_expansion"`
	Acquisition       string            `json:"acquisition"`
	Expected          AuthorityExpected `json:"expected"`
	Note              string            `json:"note"`
}

// ProducerRetained declares how many records a partition keeps acquired after the
// authoritative case and after the first refused case: a failed read must not erase
// what a completed read acquired.
type ProducerRetained struct {
	AuthoritativeCase int `json:"authoritative_case"`
	AfterFirstRefusal int `json:"after_first_refusal"`
}

// CommittedState declares the committed lineage every case must leave behind,
// because authority is not commit and IP-05-T07 owns publication.
type CommittedState struct {
	Status    string `json:"status"`
	Epoch     string `json:"epoch"`
	Revision  uint64 `json:"revision"`
	Count     int    `json:"count"`
	Queryable bool   `json:"queryable"`
}

// Invariants are the declared constant observations of the whole corpus.
type Invariants struct {
	ProducerRetainedRecords ProducerRetained `json:"producer_retained_records"`
	HostCommittedState      CommittedState   `json:"host_committed_state"`
	Note                    string           `json:"note"`
}

// AuthorityExpectedBlock is the declared observable evidence of the corpus.
type AuthorityExpectedBlock struct {
	Observations      []Observation `json:"observations"`
	Invariants        Invariants    `json:"invariants"`
	PrivacyAssertions []string      `json:"privacy_assertions"`
}

// AuthorityCorpus is the shared IP-05-T02 artifact root.
type AuthorityCorpus struct {
	SchemaVersion   int                    `json:"schema_version"`
	Phase           string                 `json:"phase"`
	Artifact        string                 `json:"artifact"`
	OwnerTask       string                 `json:"owner_task"`
	FixtureID       string                 `json:"fixture_id"`
	RequirementIDs  []string               `json:"requirement_ids"`
	SharedConsumers []string               `json:"shared_consumers"`
	ContractSource  []string               `json:"contract_source"`
	Note            string                 `json:"note"`
	OutcomeNote     string                 `json:"outcome_note"`
	Claims          Claim                  `json:"claims"`
	Authority       Authority              `json:"authority"`
	Input           AuthorityInput         `json:"input"`
	Cases           []AuthorityCase        `json:"cases"`
	Expected        AuthorityExpectedBlock `json:"expected"`
}

// LoadSnapshotAuthority reads the shared IP-05-T02 artifact exactly once per
// consumer.
func LoadSnapshotAuthority(t Fataler) AuthorityCorpus {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(SnapshotAuthorityArtifact))
	if err != nil {
		t.Fatalf("read %s: %v", SnapshotAuthorityArtifact, err)
	}
	var corpus AuthorityCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatalf("decode %s: %v", SnapshotAuthorityArtifact, err)
	}
	return corpus
}

// AuthorityCaseByID returns one declared case by its stable ID.
func (corpus AuthorityCorpus) AuthorityCaseByID(t Fataler, id string) AuthorityCase {
	t.Helper()
	for _, item := range corpus.Cases {
		if item.CaseID == id {
			return item
		}
	}
	t.Fatalf("case %s is absent from %s", id, SnapshotAuthorityArtifact)
	return AuthorityCase{}
}

// ContractPaths resolves the source paths the artifact names, so a consumer can
// prove the declared outcomes are stated against the merged source boundary.
func (corpus AuthorityCorpus) ContractPaths() []string {
	paths := make([]string, 0, len(corpus.ContractSource))
	for _, source := range corpus.ContractSource {
		paths = append(paths, filepath.Clean(filepath.Join("..", "..", "..", source)))
	}
	return paths
}

// Observation returns one declared corpus observation by signal name.
func (block AuthorityExpectedBlock) Observation(signal string) (Observation, bool) {
	for _, observation := range block.Observations {
		if observation.Signal == signal {
			return observation, true
		}
	}
	return Observation{}, false
}

// Authoritative reports whether the declared acquisition may carry records.
func (item AuthorityCase) Authoritative() bool {
	return item.Acquisition == outcomeAuthoritative
}

// HostSnapshotCall reports whether the declared host call binds a snapshot payload,
// which is the only host call that can return records.
func (item AuthorityCase) HostSnapshotCall() bool {
	return item.Expected.Host.Call == hostCallSnapshot
}

// Records returns the record payloads one declared handoff carries. A handoff with
// no declared record list returns nil, so an absent list and an explicit empty list
// stay distinguishable without either being repaired. A case that declares a bounded
// record_expansion materializes that list instead, because an over-limit read is the
// one record count the artifact cannot write out literally.
func (item AuthorityCase) Records() []map[string]any {
	if item.RecordExpansion != nil {
		return item.RecordExpansion.materialize()
	}
	rows, listed := item.Handoff["records"].([]any)
	if !listed {
		return nil
	}
	payloads := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if record, ok := row.(map[string]any); ok {
			payloads = append(payloads, record)
		}
	}
	return payloads
}

// materialize expands one declared record template into the declared count. Only the
// field the artifact names is substituted, every other value is copied from the
// template, and each record is an independent copy so a consumer cannot stage one
// record's values into another.
func (expansion *RecordExpansion) materialize() []map[string]any {
	records := make([]map[string]any, 0, expansion.Count)
	for index := 0; index < expansion.Count; index++ {
		record := make(map[string]any, len(expansion.Template))
		for field, value := range expansion.Template {
			record[field] = value
		}
		identity, ok := record["tab_identity"].(map[string]any)
		if !ok {
			continue
		}
		expanded := make(map[string]any, len(identity))
		for field, value := range identity {
			expanded[field] = value
		}
		expanded["tab_id"] = float64(expansion.IdentityStart + index*expansion.IdentityStep)
		record["tab_identity"] = expanded
		records = append(records, record)
	}
	return records
}
