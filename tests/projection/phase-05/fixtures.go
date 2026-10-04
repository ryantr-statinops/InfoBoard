// Package phase05 holds the shared IP-05-T01 projection boundary corpus and the
// bounded probe the artifact declares for a synthetic identity fallback. The Go
// consumer drives every case through the real host reconciler in
// host/internal/projection and the Node consumer drives the same cases through the
// real extension boundary in extension/src/projection/reconciler.ts; neither
// consumer restates the boundary rules, so the loader here only reads and shapes
// the artifact. Nothing here asserts snapshot authority, canonical ordering,
// revision sequencing, duplicate or gap handling, profile isolation, atomic
// replacement, or recovery: those are later IP-05 tasks with their own gates.
package phase05

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Artifact is the single shared IP-05 fixture corpus. The Go host consumer, the
// extension consumer, and the Python runner all read this one file; no consumer
// may restate the boundary in its own format.
const Artifact = "../../../fixtures/projection/phase-05/phase-05.json"

// RequiredFixtureIDs is the IP-05-T01 obligation from phase-05 section 6.
var RequiredFixtureIDs = []string{
	"FX-PROJECTION-BOUNDARY-HANDOFF-IDENTITY",
	"FX-PROJECTION-BOUNDARY-REQUIRED-FIELDS",
	"FX-PROJECTION-BOUNDARY-SYNTHETIC-IDENTITY",
}

const (
	stageProjection = "projection"
	stageBoundary   = "boundary"

	applyRecord = "record"
	applyNone   = "none"

	causeMissing = "missing_"
)

// HandoffBoundary declares which IP-04 observer handoff fields must reach the
// projection boundary before a handoff may become projection input.
type HandoffBoundary struct {
	Owner                 string              `json:"owner"`
	Consumer              string              `json:"consumer"`
	HandoffKinds          []string            `json:"handoff_kinds"`
	MessageKinds          []string            `json:"message_kinds"`
	HandoffKindMap        map[string]string   `json:"handoff_kind_map"`
	RequiredHandoffFields map[string][]string `json:"required_handoff_fields"`
	OptionalHandoffFields []string            `json:"optional_handoff_fields"`
}

// Boundary is the declared partition and fence contract for projection input.
type Boundary struct {
	MandatoryPartitionFields     []string          `json:"mandatory_partition_fields"`
	MandatoryFenceFields         []string          `json:"mandatory_fence_fields"`
	TabIdentityRequiredFields    []string          `json:"tab_identity_required_fields"`
	SnapshotRecordRequiredFields []string          `json:"snapshot_record_required_fields"`
	OptionalRecordFields         []string          `json:"optional_record_fields"`
	IdentitySources              map[string]string `json:"identity_sources"`
	AcceptedOutcomes             []string          `json:"accepted_outcomes"`
	FailClosedOutcome            string            `json:"fail_closed_outcome"`
	SyntheticFallbackAllowed     bool              `json:"synthetic_identity_fallback_allowed"`
	IP04HandoffBoundary          HandoffBoundary   `json:"ip04_handoff_boundary"`
}

// CaseState is the observable projection state after one case.
type CaseState struct {
	Epoch         *string `json:"epoch"`
	Revision      uint64  `json:"revision"`
	IdentityCount int     `json:"identity_count"`
}

// Case is one boundary input plus its declared boundary and projection outcome.
type Case struct {
	CaseID                 string         `json:"case_id"`
	Stage                  string         `json:"stage"`
	Cause                  *string        `json:"cause"`
	Apply                  string         `json:"apply"`
	Payload                map[string]any `json:"payload"`
	Handoff                map[string]any `json:"handoff"`
	MessageKind            string         `json:"message_kind"`
	BoundaryCheck          string         `json:"boundary_check"`
	SyntheticFallbackProbe bool           `json:"synthetic_fallback_probe"`
	ExpectedOutcome        string         `json:"expected_outcome"`
	ExpectedState          CaseState      `json:"expected_state"`
}

// FixtureInput binds one boundary scenario to one profile, context, and lineage.
type FixtureInput struct {
	ProfileID       string `json:"profile_id"`
	ContextKind     string `json:"context_kind"`
	ProjectionEpoch string `json:"projection_epoch"`
	Cases           []Case `json:"cases"`
}

// Observation is one declared expected signal and value.
type Observation struct {
	Signal string `json:"signal"`
	Value  any    `json:"value"`
}

// Expected declares the observable evidence and privacy boundary of a fixture.
type Expected struct {
	Observations      []Observation `json:"observations"`
	PrivacyAssertions []string      `json:"privacy_assertions"`
}

// Fixture is one reusable projection boundary scenario.
type Fixture struct {
	FixtureID      string       `json:"fixture_id"`
	OwnerTask      string       `json:"owner_task"`
	Category       string       `json:"category"`
	RequirementIDs []string     `json:"requirement_ids"`
	Title          string       `json:"title"`
	Input          FixtureInput `json:"input"`
	Expected       Expected     `json:"expected"`
}

// Corpus is the shared artifact root.
type Corpus struct {
	SchemaVersion   int       `json:"schema_version"`
	Phase           string    `json:"phase"`
	Artifact        string    `json:"artifact"`
	OwnerTask       string    `json:"owner_task"`
	RequirementIDs  []string  `json:"requirement_ids"`
	SharedConsumers []string  `json:"shared_consumers"`
	ContractSource  []string  `json:"contract_source"`
	Note            string    `json:"note"`
	OutcomeNote     string    `json:"outcome_note"`
	Boundary        Boundary  `json:"boundary"`
	ReservedIDs     []string  `json:"reserved_fixture_ids"`
	Fixtures        []Fixture `json:"fixtures"`
}

// Observation returns one declared expected observation by signal name.
func (fixture Fixture) Observation(signal string) (Observation, bool) {
	for _, observation := range fixture.Expected.Observations {
		if observation.Signal == signal {
			return observation, true
		}
	}
	return Observation{}, false
}

// Fataler is the reporting seam the corpus loader needs from a test handle.
type Fataler interface {
	Helper()
	Fatalf(format string, args ...any)
}

// LoadCorpus reads the shared IP-05 artifact exactly once per consumer.
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
func (corpus Corpus) Fixture(t Fataler, id string) Fixture {
	t.Helper()
	for _, fixture := range corpus.Fixtures {
		if fixture.FixtureID == id {
			return fixture
		}
	}
	t.Fatalf("fixture %s is absent from %s", id, Artifact)
	return Fixture{}
}

// DeclaredPaths resolves the contract source paths the artifact names, so a
// consumer can prove the boundary is stated against merged source types.
func (corpus Corpus) DeclaredPaths() []string {
	paths := make([]string, 0, len(corpus.ContractSource))
	for _, source := range corpus.ContractSource {
		paths = append(paths, filepath.Clean(filepath.Join("..", "..", "..", source)))
	}
	return paths
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// Allocated returns the identity value a fixture binds to a declared mandatory
// field, so a consumer compares against allocated identity instead of a literal.
func (input FixtureInput) Allocated(field string) (string, bool) {
	switch field {
	case "profile_id":
		return input.ProfileID, true
	case "context_kind":
		return input.ContextKind, true
	case "projection_epoch":
		return input.ProjectionEpoch, true
	}
	return "", false
}

// SyntheticCase blanks every mandatory partition and fence value of a case and
// substitutes an invalid numeric tab id, so a consumer can drive the artifact's
// declared synthetic-fallback probe through the real boundary rather than through
// a second copy of the boundary rules. The probe changes fixture data only; it
// never decides whether an input is admissible.
func (boundary Boundary) SyntheticCase(item Case) Case {
	partition, fence := boundary.blanking()
	probe := item
	if item.Handoff != nil {
		value, _ := blank(item.Handoff, partition, fence).(map[string]any)
		probe.Handoff = value
	}
	if item.Payload != nil {
		value, _ := blank(item.Payload, partition, fence).(map[string]any)
		probe.Payload = value
	}
	return probe
}

func (boundary Boundary) blanking() (map[string]bool, map[string]bool) {
	partition := map[string]bool{}
	for _, field := range boundary.MandatoryPartitionFields {
		partition[field] = true
	}
	fence := map[string]bool{}
	for _, field := range boundary.MandatoryFenceFields {
		fence[field] = true
	}
	return partition, fence
}

func blank(value any, partition, fence map[string]bool) any {
	switch typed := value.(type) {
	case map[string]any:
		result := map[string]any{}
		for key, item := range typed {
			switch {
			case key == "tab_id":
				// A synthetic identity substitutes an invalid numeric sentinel,
				// never a browser tab id.
				result[key] = float64(-1)
			case partition[key], fence[key]:
				// Blank a mandatory value without changing its type, so the probe
				// still reaches the boundary as a decodable input.
				if _, text := item.(string); text {
					result[key] = ""
				} else {
					result[key] = float64(0)
				}
			default:
				result[key] = blank(item, partition, fence)
			}
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, blank(item, partition, fence))
		}
		return result
	default:
		return value
	}
}
