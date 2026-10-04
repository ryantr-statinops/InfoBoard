package phase06

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var (
	snakeCase    = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	fixtureIDSet = regexp.MustCompile(`^FX-HOST-[A-Z]+(-[A-Z]+)*$`)
	originShape  = regexp.MustCompile(`^chrome-extension://[a-p]{32}$`)
	uuidV4Shape  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// bindingLifecycleStates is the seven-state handoff contract required by IP-06
// section 4, in the runtime's own lowercase vocabulary.
var bindingLifecycleStates = []string{
	"starting", "handshaking", "synchronizing", "ready", "closing", "unavailable", "recovering",
}

var bindingAvailabilityStates = []string{"healthy", "unavailable", "recovering"}

// bindingFailureClass indexes the runtime-owned safe classification surface so
// no assertion restates a literal set.
var bindingFailureClass = func() map[string]bool {
	index := map[string]bool{}
	for _, class := range bindingFailureClasses {
		index[class] = true
	}
	return index
}()

// bindingFailureClasses is the runtime-owned safe classification surface. IP-07
// owns the wire literals, so a phase-06 fixture may only name these classes.
var bindingFailureClasses = []string{
	"configuration_invalid", "registration_invalid", "startup_timeout",
	"handshake_timeout", "handshake_rejected", "protocol_incompatible",
	"profile_mismatch", "snapshot_rejected", "synchronization_timeout",
	"request_timeout", "transport_eof", "transport_truncated",
	"transport_malformed", "transport_broken_pipe", "transport_unavailable",
	"frame_too_large", "queue_overflow", "concurrency_limit",
	"dependency_unavailable", "shutdown", "cancelled", "internal_failure",
}

func expectString(t *testing.T, where string, source map[string]json.RawMessage, key string) string {
	t.Helper()
	raw, ok := source[key]
	if !ok {
		t.Fatalf("%s: missing key %q", where, key)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("%s: key %q is not a string: %v", where, key, err)
	}
	return value
}

func expectBool(t *testing.T, where string, source map[string]json.RawMessage, key string) bool {
	t.Helper()
	raw, ok := source[key]
	if !ok {
		t.Fatalf("%s: missing key %q", where, key)
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("%s: key %q is not a bool: %v", where, key, err)
	}
	return value
}

func expectInt(t *testing.T, where string, source map[string]json.RawMessage, key string) int64 {
	t.Helper()
	raw, ok := source[key]
	if !ok {
		t.Fatalf("%s: missing key %q", where, key)
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("%s: key %q is not an integer: %v", where, key, err)
	}
	return value
}

func expectStrings(t *testing.T, where string, source map[string]json.RawMessage, key string) []string {
	t.Helper()
	raw, ok := source[key]
	if !ok {
		t.Fatalf("%s: missing key %q", where, key)
	}
	var value []string
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("%s: key %q is not a string array: %v", where, key, err)
	}
	return value
}

func expectStringsOrNull(t *testing.T, where string, source map[string]json.RawMessage, key string) (string, bool) {
	t.Helper()
	raw, ok := source[key]
	if !ok {
		t.Fatalf("%s: missing key %q", where, key)
	}
	if string(raw) == "null" {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("%s: key %q is not a string or null: %v", where, key, err)
	}
	return value, true
}

func TestArtifactDeclaresEveryPhaseFixtureExactlyOnce(t *testing.T) {
	corpus := LoadCorpus(t)
	if corpus.SchemaVersion != 1 || corpus.Phase != "IP-06" || corpus.Artifact != "host-lifecycle" {
		t.Fatalf("artifact header: version=%d phase=%q artifact=%q", corpus.SchemaVersion, corpus.Phase, corpus.Artifact)
	}
	if corpus.ContractSource != "host/internal/runtime" {
		t.Fatalf("contract_source = %q, want the runtime package that owns the assertions", corpus.ContractSource)
	}
	if len(corpus.Fixtures) != len(RequiredFixtureIDs) {
		t.Fatalf("fixture count = %d, want %d", len(corpus.Fixtures), len(RequiredFixtureIDs))
	}
	for i, id := range RequiredFixtureIDs {
		got := corpus.Fixtures[i].FixtureID
		if got != id {
			t.Fatalf("fixture %d = %s, want %s", i, got, id)
		}
		if !fixtureIDSet.MatchString(got) {
			t.Fatalf("fixture id %q does not use the FX-HOST-* shape", got)
		}
	}
	seen := map[string]int{}
	for _, fixture := range corpus.Fixtures {
		seen[fixture.FixtureID]++
	}
	for id, count := range seen {
		if count != 1 {
			t.Fatalf("fixture %s appears %d times", id, count)
		}
	}
}

func TestArtifactIsConsumableByBothGoAndNodeConsumers(t *testing.T) {
	corpus := LoadCorpus(t)
	if len(corpus.SharedConsumers) != 2 || corpus.SharedConsumers[0] != "go" || corpus.SharedConsumers[1] != "node" {
		t.Fatalf("shared_consumers = %v, want [go node]", corpus.SharedConsumers)
	}
	if corpus.FrameEncoding.LengthPrefixBytes != 4 || corpus.FrameEncoding.ByteOrder != "little_endian" || corpus.FrameEncoding.Payload != "utf8_json" {
		t.Fatalf("frame encoding is not the browser-provided Native Messaging framing: %+v", corpus.FrameEncoding)
	}
	raw, err := os.ReadFile(filepath.Clean(Artifact))
	if err != nil {
		t.Fatal(err)
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatal(err)
	}
	for key := range generic {
		if !snakeCase.MatchString(key) {
			t.Fatalf("top-level key %q is not snake_case, so Node and Go consumers would diverge", key)
		}
	}
	fixtures, _ := generic["fixtures"].([]any)
	for _, entry := range fixtures {
		fixture, _ := entry.(map[string]any)
		for _, section := range []string{"scenario", "expected"} {
			values, ok := fixture[section].(map[string]any)
			if !ok {
				t.Fatalf("%v: %s is not an object", fixture["fixture_id"], section)
			}
			for key := range values {
				if !snakeCase.MatchString(key) {
					t.Fatalf("%v: %s key %q is not snake_case", fixture["fixture_id"], section, key)
				}
			}
		}
	}
}

func TestArtifactLimitsAreBoundedAndFixtureVisible(t *testing.T) {
	limits := LoadCorpus(t).Limits
	for name, value := range limits.ZeroedLimits() {
		if value <= 0 {
			t.Fatalf("limit %s = %d, must be a positive fixture-visible budget", name, value)
		}
	}
	if limits.MaxFrameBytes != 1<<20 {
		t.Fatalf("max_frame_bytes = %d, want the bounded 1 MiB envelope", limits.MaxFrameBytes)
	}
	if limits.MaxQueuedFrames > 4096 || limits.MaxConcurrentRequests > 256 {
		t.Fatalf("queue/concurrency limits are not bounded: queued=%d concurrent=%d", limits.MaxQueuedFrames, limits.MaxConcurrentRequests)
	}
	if limits.MaxDiagnosticRecords > 4096 || limits.MaxDiagnosticBytes > 1<<20 {
		t.Fatalf("diagnostic retention is not bounded: records=%d bytes=%d", limits.MaxDiagnosticRecords, limits.MaxDiagnosticBytes)
	}
	for _, pair := range [][2]int64{
		{limits.HandshakeBudgetMS, limits.StartupBudgetMS},
		{limits.RequestBudgetMS, limits.StartupBudgetMS},
		{limits.ShutdownBudgetMS, limits.StartupBudgetMS},
	} {
		if pair[0] > pair[1] {
			t.Fatalf("budget %d ms exceeds the startup budget %d ms", pair[0], pair[1])
		}
	}
	if limits.HandshakeBudget() <= 0 || limits.SynchronizationBudget() <= 0 {
		t.Fatal("handshake and synchronization budgets must both be observable")
	}
	for _, name := range []string{
		"startup_budget_ms", "handshake_budget_ms", "synchronization_budget_ms",
		"request_budget_ms", "shutdown_budget_ms",
	} {
		if _, ok := limits.Budget(name); !ok {
			t.Fatalf("budget %q is declared in the artifact but not resolvable by consumers", name)
		}
	}
	if _, ok := limits.Budget("unbounded_budget_ms"); ok {
		t.Fatal("unbounded budget resolved, which must never be observable")
	}
	for _, budget := range []time.Duration{limits.StartupBudget(), limits.ShutdownBudget()} {
		if budget > 5*time.Minute {
			t.Fatalf("budget %s exceeds the runtime absolute maximum", budget)
		}
	}
}

func TestArtifactStatesAndClassesMatchTheRuntimeContract(t *testing.T) {
	corpus := LoadCorpus(t)
	if len(corpus.LifecycleStates) != len(bindingLifecycleStates) {
		t.Fatalf("lifecycle_states = %v, want %v", corpus.LifecycleStates, bindingLifecycleStates)
	}
	for i, state := range bindingLifecycleStates {
		if corpus.LifecycleStates[i] != state {
			t.Fatalf("lifecycle_states[%d] = %s, want %s", i, corpus.LifecycleStates[i], state)
		}
	}
	if len(corpus.AvailabilityStates) != len(bindingAvailabilityStates) {
		t.Fatalf("availability_states = %v, want %v", corpus.AvailabilityStates, bindingAvailabilityStates)
	}
	for i, state := range bindingAvailabilityStates {
		if corpus.AvailabilityStates[i] != state {
			t.Fatalf("availability_states[%d] = %s, want %s", i, corpus.AvailabilityStates[i], state)
		}
	}
	if len(corpus.FailureClasses) != len(bindingFailureClasses) {
		t.Fatalf("failure_classes count = %d, want %d", len(corpus.FailureClasses), len(bindingFailureClasses))
	}
	known := map[string]bool{}
	for _, class := range bindingFailureClasses {
		known[class] = true
	}
	declared := map[string]int{}
	for _, class := range corpus.FailureClasses {
		if !known[class] {
			t.Fatalf("failure class %q is outside the runtime vocabulary", class)
		}
		declared[class]++
	}
	for class := range known {
		if declared[class] == 0 {
			t.Fatalf("runtime failure class %q is missing from the artifact", class)
		}
	}
	if !strings.Contains(corpus.WireErrorOwnership, "IP-07") {
		t.Fatalf("wire_error_ownership must leave the wire mapping to IP-07, got %q", corpus.WireErrorOwnership)
	}
}

func TestArtifactIdentityUsesOpaqueRegisteredValues(t *testing.T) {
	identity := LoadCorpus(t).Identity
	if !originShape.MatchString(identity.Origin) {
		t.Fatalf("origin %q is not a registered extension origin", identity.Origin)
	}
	if !originShape.MatchString(identity.UnregisteredOrigin) || identity.UnregisteredOrigin == identity.Origin {
		t.Fatalf("unregistered_origin %q must be well formed and distinct from the registered origin", identity.UnregisteredOrigin)
	}
	if len(identity.AllowedOrigins) != 1 || identity.AllowedOrigins[0] != identity.Origin {
		t.Fatalf("allowed_origins = %v, want exactly the registered origin", identity.AllowedOrigins)
	}
	if !uuidV4Shape.MatchString(identity.ProfileID) || identity.ProfileID == identity.OtherProfileID {
		t.Fatalf("profile identities must be distinct opaque UUIDs: %q %q", identity.ProfileID, identity.OtherProfileID)
	}
	if !uuidV4Shape.MatchString(identity.ProjectionEpoch) || !uuidV4Shape.MatchString(identity.Session2ProjectionEpoch) {
		t.Fatal("projection epochs must be opaque UUIDs")
	}
	if identity.ProjectionEpoch == identity.Session2ProjectionEpoch {
		t.Fatal("a fresh session must not reuse the previous projection epoch")
	}
	if identity.ProtocolVersion != 1 || identity.HostVersion == "" {
		t.Fatalf("protocol/host identity is not fixture visible: %+v", identity)
	}
}

func TestEveryFixtureBindsRequirementPrivacyAndBoundedSignals(t *testing.T) {
	corpus := LoadCorpus(t)
	for _, fixture := range corpus.Fixtures {
		where := fixture.FixtureID
		if len(fixture.RequirementIDs) == 0 {
			t.Fatalf("%s: no owning requirement", where)
		}
		for _, id := range fixture.RequirementIDs {
			if id != "FR-012" && id != "NFR-006" {
				t.Fatalf("%s: requirement %q is not owned by IP-06", where, id)
			}
		}
		if len(fixture.Scenario) == 0 || len(fixture.Expected) == 0 {
			t.Fatalf("%s: scenario and expected observations are both required", where)
		}
		if !expectBool(t, where, fixture.Expected, "bounded_completion") {
			t.Fatalf("%s: bounded completion is not an asserted observable", where)
		}
		if !expectBool(t, where, fixture.Expected, "stdout_contains_only_frames") {
			t.Fatalf("%s: stdout frame exclusivity is not asserted", where)
		}
		if !expectBool(t, where, fixture.Expected, "diagnostics_absent_from_stdout") {
			t.Fatalf("%s: diagnostics must be asserted absent from stdout", where)
		}
		if !expectBool(t, where, fixture.Expected, "sentinels_absent_from_diagnostics") {
			t.Fatalf("%s: privacy sentinel absence is not asserted", where)
		}
		if raw, declared := fixture.Expected["failure_class"]; declared && string(raw) != "null" {
			var class string
			if err := json.Unmarshal(raw, &class); err != nil || !bindingFailureClass[class] {
				t.Fatalf("%s: failure_class %q is outside the runtime vocabulary", where, class)
			}
		}
		hasFailureSignal := false
		for key := range fixture.Expected {
			if strings.Contains(key, "failure_class") {
				hasFailureSignal = true
			}
		}
		if !hasFailureSignal {
			t.Fatalf("%s: no failure_class signal is declared for any case", where)
		}
		if mutations := expectInt(t, where, fixture.Expected, "tab_mutations"); mutations != 0 {
			t.Fatalf("%s: host lifecycle must never mutate browser tabs, got %d", where, mutations)
		}
	}
}

func TestEveryScenarioUsesTheFixtureIdentityAndRegisteredVocabulary(t *testing.T) {
	corpus := LoadCorpus(t)
	knownStates := map[string]bool{}
	for _, state := range bindingLifecycleStates {
		knownStates[state] = true
	}
	_ = knownStates
	knownClasses := bindingFailureClass
	for _, fixture := range corpus.Fixtures {
		where := fixture.FixtureID
		origin := expectString(t, where, fixture.Scenario, "invocation_origin")
		if !originShape.MatchString(origin) {
			t.Fatalf("%s: invocation_origin %q is not a browser-provided extension origin", where, origin)
		}
		allowed := expectStrings(t, where, fixture.Scenario, "allowed_origins")
		if len(allowed) > 1 {
			t.Fatalf("%s: allowed_origins must stay minimal, got %v", where, allowed)
		}
		for _, value := range allowed {
			if !originShape.MatchString(value) {
				t.Fatalf("%s: allowed origin %q is not a registered extension origin", where, value)
			}
		}
		registered := expectBool(t, where, fixture.Scenario, "registered_host")
		if registered && (len(allowed) == 0 || allowed[0] != origin) {
			t.Fatalf("%s: a registered host scenario must allow its own invocation origin", where)
		}
		if !registered && len(allowed) != 0 {
			t.Fatalf("%s: an unregistered host scenario must not carry an allowlist", where)
		}
		if version := expectInt(t, where, fixture.Scenario, "protocol_version"); version != int64(corpus.Identity.ProtocolVersion) {
			t.Fatalf("%s: protocol_version = %d, want the artifact protocol version", where, version)
		}
		if hostVersion := expectString(t, where, fixture.Scenario, "host_version"); hostVersion != corpus.Identity.HostVersion {
			t.Fatalf("%s: host_version = %q, want the artifact host version", where, hostVersion)
		}
		for key, raw := range fixture.Expected {
			value := strings.TrimSpace(string(raw))
			if strings.HasPrefix(key, "lifecycle_path") {
				for _, state := range expectStrings(t, where, fixture.Expected, key) {
					if !knownStates[state] {
						t.Fatalf("%s: lifecycle state %q is outside the runtime vocabulary", where, state)
					}
				}
			}
			if strings.HasSuffix(key, "availability") || key == "availability" {
				if !strings.HasPrefix(value, `"`) {
					continue
				}
				var state string
				if err := json.Unmarshal(raw, &state); err != nil {
					t.Fatalf("%s: %s is not a string: %v", where, key, err)
				}
				if state != "healthy" && state != "unavailable" && state != "recovering" {
					t.Fatalf("%s: availability %q is outside the public vocabulary", where, state)
				}
			}
			if strings.Contains(key, "failure_class") && value != "null" {
				var class string
				if err := json.Unmarshal(raw, &class); err == nil && !knownClasses[class] {
					t.Fatalf("%s: failure class %q is outside the runtime vocabulary", where, class)
				}
			}
		}
	}
}

func TestFixtureInputsStayInsideTheDeclaredEnvelope(t *testing.T) {
	corpus := LoadCorpus(t)
	for _, fixture := range corpus.Fixtures {
		where := fixture.FixtureID
		for key, raw := range fixture.Scenario {
			if !strings.HasSuffix(key, "tab_ids") {
				continue
			}
			var ids []int64
			if err := json.Unmarshal(raw, &ids); err != nil {
				t.Fatalf("%s: %s is not an integer array: %v", where, key, err)
			}
			if len(ids) > 1000 {
				t.Fatalf("%s: %s has %d tabs, above the 1000-tab envelope", where, key, len(ids))
			}
			seen := map[int64]bool{}
			for _, id := range ids {
				if id < 0 || seen[id] {
					t.Fatalf("%s: tab id %d is negative or duplicated", where, id)
				}
				seen[id] = true
			}
		}
		if _, declared := fixture.Expected["rejected_inputs"]; declared {
			cases := expectSlice(t, where, fixture.Scenario, "rejected_inputs")
			rejections := expectInt(t, where, fixture.Expected, "rejected_inputs")
			if int64(len(cases)) != rejections {
				t.Fatalf("%s: declared %d rejections but listed %d cases", where, rejections, len(cases))
			}
			seen := map[string]bool{}
			for _, testCase := range cases {
				name := expectString(t, where, testCase, "case")
				if seen[name] {
					t.Fatalf("%s: rejected case %q is duplicated", where, name)
				}
				seen[name] = true
				if class := expectString(t, where, testCase, "observed_failure_class"); !bindingFailureClass[class] {
					t.Fatalf("%s: case %s observed class %q is outside the runtime vocabulary", where, name, class)
				}
			}
		}
	}
}

func expectSlice(t *testing.T, where string, source map[string]json.RawMessage, key string) []map[string]json.RawMessage {
	t.Helper()
	raw, ok := source[key]
	if !ok {
		t.Fatalf("%s: missing key %q", where, key)
	}
	var value []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("%s: key %q is not an object array: %v", where, key, err)
	}
	return value
}

func TestFixtureOutputsNeverCarrySensitiveSentinelValues(t *testing.T) {
	corpus := LoadCorpus(t)
	sentinels := corpus.PrivacySentinels
	values := []string{sentinels.Title, sentinels.URL, sentinels.Query, sentinels.Token, sentinels.PageMarker}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			t.Fatal("privacy sentinel set is incomplete")
		}
	}
	raw, err := os.ReadFile(filepath.Clean(Artifact))
	if err != nil {
		t.Fatal(err)
	}
	var generic struct {
		Fixtures []map[string]any `json:"fixtures"`
	}
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range generic.Fixtures {
		id, _ := fixture["fixture_id"].(string)
		for section, value := range fixture {
			if section == "scenario" {
				continue
			}
			encoded := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(mustJSON(value)), " ", ""))
			for _, sentinel := range values {
				if strings.Contains(encoded, strings.ToLower(sentinel)) {
					t.Fatalf("%s: %s restates the sensitive sentinel %q", id, section, sentinel)
				}
			}
		}
	}
}

func mustJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func TestSharedCatalogStillRoutesIP06ThroughTheHostFailureSeam(t *testing.T) {
	raw, err := os.ReadFile(filepath.Clean("../../../fixtures/catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Fixtures []struct {
			FixtureID  string `json:"fixture_id"`
			OwnerPhase string `json:"owner_phase"`
			Category   string `json:"category"`
		} `json:"fixtures"`
		PhaseSeams []struct {
			Phase             string `json:"phase"`
			OwnershipRelation string `json:"ownership_relation"`
			FixtureSignal     string `json:"fixture_signal"`
			AcceptanceSignal  string `json:"acceptance_signal"`
		} `json:"phase_seams"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Fixtures) != 9 {
		t.Fatalf("shared catalog fixture count = %d, want the fixed 9 IDs", len(catalog.Fixtures))
	}
	seam := 0
	for _, entry := range catalog.PhaseSeams {
		if entry.Phase != "IP-06" {
			continue
		}
		seam++
		if entry.OwnershipRelation != "supporting" || entry.FixtureSignal != "surface_status" {
			t.Fatalf("IP-06 seam drifted: %+v", entry)
		}
		if entry.AcceptanceSignal == "" {
			t.Fatal("IP-06 seam lost its acceptance signal")
		}
	}
	if seam != 1 {
		t.Fatalf("IP-06 phase seam count = %d, want 1", seam)
	}
	for _, fixture := range catalog.Fixtures {
		if fixture.OwnerPhase == "IP-06" && (fixture.FixtureID != "FX-HOST-DOWN" || fixture.Category != "host") {
			t.Fatalf("shared catalog host fixture drifted: %+v", fixture)
		}
	}
}
