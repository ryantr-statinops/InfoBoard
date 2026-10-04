package phase06

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Artifact is the single shared IP-06 fixture artifact. Go and Node consumers load
// the same file; no phase consumer may restate the corpus in its own format.
const Artifact = "../../../fixtures/host/phase-06/phase-06.json"

type Limits struct {
	MaxFrameBytes           int64 `json:"max_frame_bytes"`
	MaxQueuedFrames         int64 `json:"max_queued_frames"`
	MaxConcurrentRequests   int64 `json:"max_concurrent_requests"`
	MaxDiagnosticRecords    int64 `json:"max_diagnostic_records"`
	MaxDiagnosticBytes      int64 `json:"max_diagnostic_bytes"`
	StartupBudgetMS         int64 `json:"startup_budget_ms"`
	HandshakeBudgetMS       int64 `json:"handshake_budget_ms"`
	SynchronizationBudgetMS int64 `json:"synchronization_budget_ms"`
	RequestBudgetMS         int64 `json:"request_budget_ms"`
	ShutdownBudgetMS        int64 `json:"shutdown_budget_ms"`
}

type FrameEncoding struct {
	LengthPrefixBytes int    `json:"length_prefix_bytes"`
	ByteOrder         string `json:"byte_order"`
	Payload           string `json:"payload"`
	Note              string `json:"note"`
}

type PrivacySentinels struct {
	Title      string `json:"title"`
	URL        string `json:"url"`
	Query      string `json:"query"`
	Token      string `json:"token"`
	PageMarker string `json:"page_marker"`
}

type Identity struct {
	Origin                  string   `json:"origin"`
	AllowedOrigins          []string `json:"allowed_origins"`
	UnregisteredOrigin      string   `json:"unregistered_origin"`
	ProfileID               string   `json:"profile_id"`
	OtherProfileID          string   `json:"other_profile_id"`
	ProjectionEpoch         string   `json:"projection_epoch"`
	Session2ProjectionEpoch string   `json:"session_2_projection_epoch"`
	HostVersion             string   `json:"host_version"`
	ProtocolVersion         int      `json:"protocol_version"`
}

type HostFixture struct {
	FixtureID      string                     `json:"fixture_id"`
	RequirementIDs []string                   `json:"requirement_ids"`
	Scenario       map[string]json.RawMessage `json:"scenario"`
	Expected       map[string]json.RawMessage `json:"expected"`
}

type Corpus struct {
	SchemaVersion      int              `json:"schema_version"`
	Phase              string           `json:"phase"`
	Artifact           string           `json:"artifact"`
	RequirementIDs     []string         `json:"requirement_ids"`
	SharedConsumers    []string         `json:"shared_consumers"`
	ContractSource     string           `json:"contract_source"`
	FrameEncoding      FrameEncoding    `json:"frame_encoding"`
	Limits             Limits           `json:"limits"`
	LifecycleStates    []string         `json:"lifecycle_states"`
	AvailabilityStates []string         `json:"availability_states"`
	FailureClasses     []string         `json:"failure_classes"`
	WireErrorOwnership string           `json:"wire_error_ownership"`
	Identity           Identity         `json:"identity"`
	PrivacySentinels   PrivacySentinels `json:"privacy_sentinels"`
	Fixtures           []HostFixture    `json:"fixtures"`
}

// RequiredFixtureIDs is the IP-06 fixture obligation from the phase plan.
var RequiredFixtureIDs = []string{
	"FX-HOST-START-HEALTHY",
	"FX-HOST-ABSENT",
	"FX-HOST-RECOVERING",
	"FX-HOST-STDIO-OWNERSHIP",
	"FX-HOST-TIMEOUT",
	"FX-HOST-CANCEL-DISCONNECT",
	"FX-HOST-RESOURCE-LIMIT",
	"FX-HOST-NO-LOOPBACK",
}

// Fataler is the reporting seam the corpus loader needs from a test handle.
type Fataler interface {
	Helper()
	Fatalf(format string, args ...any)
}

// LoadCorpus reads the shared IP-06 artifact exactly once per consumer.
func LoadCorpus(t Fataler) Corpus {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(Artifact))
	if err != nil {
		t.Fatalf("read %s: %v", Artifact, err)
	}
	var corpus Corpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatalf("decode %s: %v", Artifact, err)
	}
	return corpus
}

// Fixture returns one fixture by its stable ID.
func (corpus Corpus) Fixture(t Fataler, id string) HostFixture {
	t.Helper()
	for _, fixture := range corpus.Fixtures {
		if fixture.FixtureID == id {
			return fixture
		}
	}
	t.Fatalf("fixture %s is absent from %s", id, Artifact)
	return HostFixture{}
}

func (limits Limits) StartupBudget() time.Duration {
	return time.Duration(limits.StartupBudgetMS) * time.Millisecond
}

func (limits Limits) HandshakeBudget() time.Duration {
	return time.Duration(limits.HandshakeBudgetMS) * time.Millisecond
}

func (limits Limits) SynchronizationBudget() time.Duration {
	return time.Duration(limits.SynchronizationBudgetMS) * time.Millisecond
}

func (limits Limits) RequestBudget() time.Duration {
	return time.Duration(limits.RequestBudgetMS) * time.Millisecond
}

func (limits Limits) ShutdownBudget() time.Duration {
	return time.Duration(limits.ShutdownBudgetMS) * time.Millisecond
}

// Budget resolves a fixture-visible budget name to its duration so assertions
// never restate a numeric constant.
func (limits Limits) Budget(name string) (time.Duration, bool) {
	switch name {
	case "startup_budget_ms":
		return limits.StartupBudget(), true
	case "handshake_budget_ms":
		return limits.HandshakeBudget(), true
	case "synchronization_budget_ms":
		return limits.SynchronizationBudget(), true
	case "request_budget_ms":
		return limits.RequestBudget(), true
	case "shutdown_budget_ms":
		return limits.ShutdownBudget(), true
	default:
		return 0, false
	}
}

// ZeroedLimits returns the declared limit set with every bounded value cleared.
// A host configuration built from it must be rejected before any allocation.
func (limits Limits) ZeroedLimits() map[string]int64 {
	return map[string]int64{
		"max_frame_bytes":           limits.MaxFrameBytes,
		"max_queued_frames":         limits.MaxQueuedFrames,
		"max_concurrent_requests":   limits.MaxConcurrentRequests,
		"max_diagnostic_records":    limits.MaxDiagnosticRecords,
		"max_diagnostic_bytes":      limits.MaxDiagnosticBytes,
		"startup_budget_ms":         limits.StartupBudgetMS,
		"handshake_budget_ms":       limits.HandshakeBudgetMS,
		"synchronization_budget_ms": limits.SynchronizationBudgetMS,
		"request_budget_ms":         limits.RequestBudgetMS,
		"shutdown_budget_ms":        limits.ShutdownBudgetMS,
	}
}
