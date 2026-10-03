package protocol

import (
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	snakeCase   = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	fixtureIDOK = regexp.MustCompile(`^NM-PROTO-[0-9]{3}$`)
	uuidV4      = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	// wireErrorCodes is the binding error vocabulary from
	// docs/plan/refactor/runtime-protocol.md. A fixture may not invent a code.
	wireErrorCodes = []string{
		"INVALID_FRAME", "PROTOCOL_MISMATCH", "PROFILE_MISMATCH", "PAYLOAD_LIMIT",
		"SNAPSHOT_REQUIRED", "REVISION_MISMATCH", "INDEX_REBUILDING", "QUERY_TIMEOUT",
		"PERSISTENCE_DEGRADED", "HOST_SHUTDOWN", "INTERNAL_FAILURE",
	}
)

func declaredErrorCodes() map[string]bool {
	index := map[string]bool{}
	for _, code := range wireErrorCodes {
		index[code] = true
	}
	return index
}

func TestCorpusDeclaresEveryRequiredFixtureExactlyOnce(t *testing.T) {
	corpus := LoadCorpus(t)
	if corpus.SchemaVersion != 1 || corpus.Phase != "IP-07" || corpus.Artifact != "native-messaging-protocol" {
		t.Fatalf("artifact header: version=%d phase=%q artifact=%q", corpus.SchemaVersion, corpus.Phase, corpus.Artifact)
	}
	if corpus.ContractSource != "host/internal/protocol" {
		t.Fatalf("contract_source = %q, want the Go protocol package under test", corpus.ContractSource)
	}
	if len(corpus.Fixtures) != len(RequiredFixtureIDs) {
		t.Fatalf("fixture count = %d, want %d", len(corpus.Fixtures), len(RequiredFixtureIDs))
	}
	for index, id := range RequiredFixtureIDs {
		got := corpus.Fixtures[index].FixtureID
		if got != id {
			t.Fatalf("fixture %d = %s, want %s", index, got, id)
		}
		if !fixtureIDOK.MatchString(got) {
			t.Fatalf("fixture id %q does not use the NM-PROTO-NNN shape", got)
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
	for _, fixture := range corpus.Fixtures {
		if strings.TrimSpace(fixture.Title) == "" {
			t.Fatalf("%s has no title, so its intent would be guessed", fixture.FixtureID)
		}
		if len(fixture.RequirementIDs) == 0 {
			t.Fatalf("%s binds no requirement", fixture.FixtureID)
		}
	}
}

func TestCorpusIsConsumableByBothGoAndNodeConsumers(t *testing.T) {
	corpus := LoadCorpus(t)
	if len(corpus.SharedConsumers) != 2 || corpus.SharedConsumers[0] != "go" || corpus.SharedConsumers[1] != "node" {
		t.Fatalf("shared_consumers = %v, want [go node]", corpus.SharedConsumers)
	}
	raw, err := os.ReadFile(Artifact)
	if err != nil {
		t.Fatal(err)
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatal(err)
	}
	for key := range generic {
		if !snakeCase.MatchString(key) {
			t.Fatalf("top-level key %q is not snake_case, so Go and Node consumers diverge", key)
		}
	}
	entries, _ := generic["fixtures"].([]any)
	for _, entry := range entries {
		fixture, _ := entry.(map[string]any)
		id, _ := fixture["fixture_id"].(string)
		for _, section := range []string{"pre_state", "expected"} {
			values, ok := fixture[section].(map[string]any)
			if !ok {
				t.Fatalf("%s: %s is not an object", id, section)
			}
			for key := range values {
				if !snakeCase.MatchString(key) {
					t.Fatalf("%s: %s key %q is not snake_case", id, section, key)
				}
			}
		}
		frames, ok := fixture["input_frames"].([]any)
		if !ok || len(frames) == 0 {
			t.Fatalf("%s: input_frames must be a non-empty array", id)
		}
		for _, rawFrame := range frames {
			frame, _ := rawFrame.(map[string]any)
			for key := range frame {
				if !snakeCase.MatchString(key) {
					t.Fatalf("%s: input frame key %q is not snake_case", id, key)
				}
			}
			_, declaresBytes := frame["raw_payload_text"]
			_, declaresSemantic := frame["type"]
			if !declaresBytes && !declaresSemantic {
				t.Fatalf("%s: an input frame declares neither raw bytes nor a semantic envelope", id)
			}
			if declaresBytes && declaresSemantic {
				t.Fatalf("%s: an input frame declares both raw bytes and a semantic envelope, so replay is ambiguous", id)
			}
		}
	}
}

func TestTransportAndEnvelopeBoundariesAreDeclared(t *testing.T) {
	corpus := LoadCorpus(t)
	if corpus.FrameEncoding.Owner != "IP-06" {
		t.Fatalf("frame encoding owner = %q, want IP-06", corpus.FrameEncoding.Owner)
	}
	if corpus.FrameEncoding.LengthPrefix != 4 || corpus.FrameEncoding.ByteOrder != "little_endian" {
		t.Fatalf("frame encoding is not the browser-provided Native Messaging framing: %+v", corpus.FrameEncoding)
	}
	boundary := corpus.TransportBoundary
	if boundary.MaxFrameBytes != 1048576 {
		t.Fatalf("max_frame_bytes = %d, want the IP-06 1 MiB transport boundary", boundary.MaxFrameBytes)
	}
	if boundary.MaxJSONPayloadBytes != 262144 {
		t.Fatalf("max_json_payload_bytes = %d, want the IP-07 256 KiB payload cap", boundary.MaxJSONPayloadBytes)
	}
	if boundary.MaxJSONPayloadBytes >= boundary.MaxFrameBytes {
		t.Fatal("the JSON payload cap must stay inside the transport frame boundary")
	}
	if !boundary.LimitBeforeDecode || !boundary.FrameZeroLengthRejected {
		t.Fatal("the payload cap must be declared as enforced before decode")
	}
	if boundary.OverFrameBoundaryOwner != "IP-06" {
		t.Fatalf("over_frame_boundary_owner = %q, want IP-06", boundary.OverFrameBoundaryOwner)
	}
	envelope := corpus.Envelope
	want := []string{"protocol", "type", "request_id", "profile_id", "projection_revision", "payload"}
	if len(envelope.RequiredFields) != len(want) {
		t.Fatalf("required_fields = %v, want exactly %v", envelope.RequiredFields, want)
	}
	for index, field := range want {
		if envelope.RequiredFields[index] != field {
			t.Fatalf("required_fields[%d] = %s, want %s", index, envelope.RequiredFields[index], field)
		}
	}
	if !envelope.UnknownFieldsRejected || !envelope.DuplicateKeysRejected || !envelope.TrailingBytesRejected {
		t.Fatal("strict envelope rejection rules are not all declared")
	}
	if envelope.ProtocolMajor != 1 || envelope.RequestIDMaxBytes != 128 || envelope.ProfileIDMaxBytes != 128 {
		t.Fatalf("envelope bounds drifted: %+v", envelope)
	}
	if envelope.RevisionMinimum != 0 {
		t.Fatalf("projection_revision_min = %d, want 0", envelope.RevisionMinimum)
	}
	if envelope.ProfileIdentitySource != "envelope_profile_id" {
		t.Fatalf("profile_identity_source = %q, want the envelope as sole identity", envelope.ProfileIdentitySource)
	}
}

func TestLimitsAndDeadlinesAreBoundedAndResolvable(t *testing.T) {
	corpus := LoadCorpus(t)
	limits := corpus.Limits
	bounded := map[string]int64{
		"query_max_scalars":      limits.QueryMaxScalars,
		"snapshot_max_records":   limits.SnapshotMaxRecords,
		"delta_max_operations":   limits.DeltaMaxOperations,
		"query_max_results":      limits.QueryMaxResults,
		"title_max_scalars":      limits.TitleMaxScalars,
		"domain_max_bytes":       limits.DomainMaxBytes,
		"url_max_bytes":          limits.URLMaxBytes,
		"capabilities_max_count": limits.CapabilitiesMaxCount,
		"message_type_max_bytes": limits.MessageTypeMaxBytes,
		"snapshot_tab_id_max":    limits.SnapshotTabIDMaximum,
	}
	for name, value := range bounded {
		if value <= 0 {
			t.Fatalf("limit %s = %d, must be a positive bounded value", name, value)
		}
	}
	if limits.QueryMaxScalars != 512 || limits.SnapshotMaxRecords != 10000 || limits.DeltaMaxOperations != 1000 {
		t.Fatalf("message size limits drifted: %+v", limits)
	}
	if limits.QueryMaxResults != 50 || limits.TitleMaxScalars != 512 || limits.DomainMaxBytes != 255 || limits.URLMaxBytes != 2048 {
		t.Fatalf("field and result limits drifted: %+v", limits)
	}
	if limits.CapabilitiesMaxCount != 32 || limits.MessageTypeMaxBytes != 64 {
		t.Fatalf("capability/type limits drifted: %+v", limits)
	}
	for _, name := range []string{
		"frame_read", "hello", "snapshot_apply", "delta_apply",
		"query_host_budget", "query_request_budget", "health",
		"activation_status", "shutdown_drain",
	} {
		budget, ok := corpus.Deadlines.Budget(name)
		if !ok {
			t.Fatalf("deadline %q is declared but not resolvable by consumers", name)
		}
		if budget <= 0 || budget > 5*time.Minute {
			t.Fatalf("deadline %q = %s is not bounded", name, budget)
		}
	}
	if _, ok := corpus.Deadlines.Budget("query_unbounded"); ok {
		t.Fatal("an unbounded deadline resolved, which must never be observable")
	}
	if corpus.Deadlines.QueryHostBudgetMS >= corpus.Deadlines.QueryRequestMS {
		t.Fatalf("the host query budget (%d ms) must stay inside the request budget (%d ms)", corpus.Deadlines.QueryHostBudgetMS, corpus.Deadlines.QueryRequestMS)
	}
}

func TestMessageContractCoversEveryDeclaredTypeAndDirection(t *testing.T) {
	corpus := LoadCorpus(t)
	index := corpus.MessageTypeIndex()
	want := map[string]string{
		"hello":               "extension_to_host",
		"hello_ack":           "host_to_extension",
		"snapshot":            "extension_to_host",
		"delta":               "extension_to_host",
		"sync_ack":            "host_to_extension",
		"resync_required":     "host_to_extension",
		"query":               "extension_to_host",
		"query_result":        "host_to_extension",
		"activation_observed": "extension_to_host",
		"activation_failed":   "extension_to_host",
		"activation_ack":      "host_to_extension",
		"health":              "extension_to_host",
		"health_result":       "host_to_extension",
		"error":               "host_to_extension",
	}
	if len(index) != len(want) || len(corpus.MessageTypes) != len(want) {
		t.Fatalf("declared message types = %d, want %d", len(corpus.MessageTypes), len(want))
	}
	for name, direction := range want {
		entry, ok := index[name]
		if !ok {
			t.Fatalf("message type %q is absent from the contract", name)
		}
		if entry.Direction != direction {
			t.Fatalf("%s direction = %s, want %s", name, entry.Direction, direction)
		}
		if len(entry.PayloadFields) == 0 && name != "health" {
			t.Fatalf("%s declares no payload fields", name)
		}
		for _, field := range entry.PayloadFields {
			if !snakeCase.MatchString(field) {
				t.Fatalf("%s payload field %q is not snake_case", name, field)
			}
		}
	}
	required := map[string]bool{"code": false, "retryable": false, "message_key": false}
	for _, field := range index["error"].PayloadFields {
		if _, tracked := required[field]; tracked {
			required[field] = true
		}
	}
	for field, present := range required {
		if !present {
			t.Fatalf("the error contract must require %q", field)
		}
	}
	optional := map[string]bool{}
	for _, field := range corpus.ErrorOptionalFields {
		optional[field] = true
	}
	for _, field := range []string{"retry_after_ms", "expected_revision", "expected_sequence"} {
		if !optional[field] {
			t.Fatalf("error contract lost its optional field %q", field)
		}
	}
	if len(corpus.DeltaOperations) != 7 {
		t.Fatalf("delta_operations = %v, want the seven declared operations", corpus.DeltaOperations)
	}
	for _, operation := range []string{"create", "update", "move", "group", "pin", "activate", "remove"} {
		found := false
		for _, declared := range corpus.DeltaOperations {
			if declared == operation {
				found = true
			}
		}
		if !found {
			t.Fatalf("delta operation %q is absent from the contract", operation)
		}
	}
}

func TestErrorAndStateVocabulariesMatchTheBindingContract(t *testing.T) {
	corpus := LoadCorpus(t)
	if len(corpus.ErrorCodes) != len(wireErrorCodes) {
		t.Fatalf("declared error codes = %d, want %d", len(corpus.ErrorCodes), len(wireErrorCodes))
	}
	declared := map[string]int{}
	for _, code := range corpus.ErrorCodes {
		declared[code]++
		if !corpus.IsErrorCode(code) {
			t.Fatalf("error code %q is not resolvable by consumers", code)
		}
	}
	for _, code := range wireErrorCodes {
		if declared[code] != 1 {
			t.Fatalf("wire error code %q is declared %d times", code, declared[code])
		}
	}
	for _, reason := range corpus.ResyncReasons {
		if !snakeCase.MatchString(reason) {
			t.Fatalf("resync reason %q is not a safe snake_case literal", reason)
		}
	}
	if len(corpus.ResyncReasons) == 0 {
		t.Fatal("no safe resync reason is declared")
	}
	for _, state := range []string{"unbound", "bound", "ready", "degraded", "incompatible", "disconnected", "shutdown"} {
		if !contains(corpus.SessionStates, state) {
			t.Fatalf("session state %q is absent from the contract", state)
		}
	}
	for _, state := range []string{"Ready", "Degraded", "Reconnecting", "Incompatible", "Shutdown"} {
		if !contains(corpus.ClientStates, state) {
			t.Fatalf("client state %q is absent from the contract", state)
		}
	}
}

func TestFixtureIdentitiesUseOpaqueRegisteredValues(t *testing.T) {
	corpus := LoadCorpus(t)
	identity := corpus.Identity
	if !uuidV4.MatchString(identity.ProfileID) || !uuidV4.MatchString(identity.OtherProfileID) {
		t.Fatal("profile identities must be distinct opaque UUIDs")
	}
	if identity.ProfileID == identity.OtherProfileID {
		t.Fatal("the mismatching profile identity must differ from the bound one")
	}
	if !uuidV4.MatchString(identity.ProjectionEpoch) || identity.ProjectionEpoch == identity.SecondProjectionEpoch {
		t.Fatal("projection epochs must be distinct opaque UUIDs")
	}
	if identity.HostVersion == "" || identity.ExtensionVersion == "" || identity.RankingModelVersion == "" {
		t.Fatalf("versions are not fixture visible: %+v", identity)
	}
	if identity.BrowserFamily == "" || identity.BrowserVersion == "" || identity.ContextKind != "normal" {
		t.Fatalf("browser/context identity is not fixture visible: %+v", identity)
	}
	if len(identity.SupportedProtocolVersions) == 0 || identity.SupportedProtocolVersions[0] != corpus.Envelope.ProtocolMajor {
		t.Fatalf("supported protocol versions = %v, want the declared major %d", identity.SupportedProtocolVersions, corpus.Envelope.ProtocolMajor)
	}
	if int64(len(identity.Capabilities)) > corpus.Limits.CapabilitiesMaxCount {
		t.Fatalf("declared capabilities exceed the declared maximum")
	}
	if len(identity.TabIDs) == 0 || len(identity.ReplacementTabIDs) <= len(identity.TabIDs) {
		t.Fatal("fixture identities must declare a base and a larger replacement tab set")
	}
}

func TestEveryFixtureDeclaresInputsPreStateAndBoundedObservables(t *testing.T) {
	corpus := LoadCorpus(t)
	states := map[string]bool{}
	for _, state := range corpus.SessionStates {
		states[state] = true
	}
	codes := declaredErrorCodes()
	for _, fixture := range corpus.Fixtures {
		where := fixture.FixtureID
		if len(fixture.InputFrames) == 0 {
			t.Fatalf("%s: no input frames declared", where)
		}
		if !states[fixture.PreState.State] {
			t.Fatalf("%s: pre_state %q is outside the session state vocabulary", where, fixture.PreState.State)
		}
		if len(fixture.Expected.EmittedFrames) == 0 {
			t.Fatalf("%s: no expected emitted frames", where)
		}
		if !states[fixture.Expected.State] {
			t.Fatalf("%s: expected state %q is outside the session state vocabulary", where, fixture.Expected.State)
		}
		if fixture.Expected.ClientState != "" && !contains(corpus.ClientStates, fixture.Expected.ClientState) {
			t.Fatalf("%s: expected client state %q is outside the client vocabulary", where, fixture.Expected.ClientState)
		}
		if !fixture.Expected.NoSideEffect {
			t.Fatalf("%s: a fixture must declare its no-side-effect assertion", where)
		}
		if !fixture.Expected.BoundedCompletion {
			t.Fatalf("%s: a fixture must declare bounded completion", where)
		}
		redaction := fixture.Expected.Redaction
		if redaction.CarriesRawTabFields {
			t.Fatalf("%s: raw tab fields must never be an expected observable", where)
		}
		if !redaction.EchoesRequestIDWhenParseable || !redaction.MessageKeyIsClosedVocabulary {
			t.Fatalf("%s: redaction assertions are incomplete: %+v", where, redaction)
		}
		if code, ok := optionalCode(fixture.Expected.ErrorCode); ok && !codes[code] {
			t.Fatalf("%s: expected error code %q is outside the wire vocabulary", where, code)
		}
		for _, frame := range fixture.Expected.EmittedFrames {
			if _, declared := corpus.MessageTypeIndex()[frame.Type]; !declared {
				t.Fatalf("%s: emitted frame type %q is not in the message contract", where, frame.Type)
			}
			if frame.Code != nil && !codes[*frame.Code] {
				t.Fatalf("%s: emitted error code %q is outside the wire vocabulary", where, *frame.Code)
			}
			if frame.EmitCount < 0 {
				t.Fatalf("%s: emit_count = %d, must be non-negative", where, frame.EmitCount)
			}
			if frame.EmitCount > 1 && !frame.IdenticalBytes {
				t.Fatalf("%s: a repeated emit must declare identical bytes", where)
			}
		}
		for _, spec := range fixture.InputFrames {
			if spec.ExpectedErrorCode == nil {
				continue
			}
			if !codes[*spec.ExpectedErrorCode] {
				t.Fatalf("%s: case %q expects %q which is outside the wire vocabulary", where, spec.Case, *spec.ExpectedErrorCode)
			}
		}
	}
}

func TestFixtureBytesMaterializeDeterministicallyAndRespectDeclaredBounds(t *testing.T) {
	corpus := LoadCorpus(t)
	for _, fixture := range corpus.Fixtures {
		where := fixture.FixtureID
		first := MaterializeAll(corpus, fixture)
		second := MaterializeAll(corpus, fixture)
		if len(first) != len(second) {
			t.Fatalf("%s: replay produced %d frames then %d", where, len(first), len(second))
		}
		for index := range first {
			if string(first[index].Bytes) != string(second[index].Bytes) {
				t.Fatalf("%s: frame %d bytes are not deterministic across replays", where, index)
			}
			wire := first[index]
			if len(wire.Bytes) < 4 {
				t.Fatalf("%s: frame %d has no length prefix", where, index)
			}
			declared := int(wire.DeclaredLen)
			delivered := len(wire.Bytes) - 4
			if int64(declared) > corpus.TransportBoundary.MaxFrameBytes && wire.Spec.ExpectedErrorCode == nil {
				t.Fatalf("%s: frame %d is accepted yet exceeds the declared transport boundary", where, index)
			}
			if wire.Withheld > 0 {
				if delivered >= declared {
					t.Fatalf("%s: truncated frame %d delivered %d of %d bytes", where, index, delivered, declared)
				}
				continue
			}
			if delivered != declared {
				t.Fatalf("%s: frame %d declared %d bytes but delivered %d", where, index, declared, delivered)
			}
			if int64(declared) > corpus.TransportBoundary.MaxFrameBytes {
				t.Fatalf("%s: accepted frame %d exceeds the declared transport boundary", where, index)
			}
			if wire.Spec.ExpectedErrorCode != nil {
				continue
			}
			if int64(wire.PayloadBytes) > corpus.TransportBoundary.MaxJSONPayloadBytes {
				t.Fatalf("%s: accepted frame %d payload %d exceeds the declared JSON cap", where, index, wire.PayloadBytes)
			}
			spec := wire.Spec
			shape, err := Shape(wire.Bytes[4:])
			if err != nil {
				t.Fatalf("%s: frame %d is not a single JSON object: %v", where, index, err)
			}
			if spec.Type != "" {
				if shape.Type != spec.Type {
					t.Fatalf("%s: frame %d type = %q, want %q", where, index, shape.Type, spec.Type)
				}
				if shape.Protocol != int64(corpus.Envelope.ProtocolMajor) && spec.Protocol == nil {
					t.Fatalf("%s: frame %d protocol = %d, want %d", where, index, shape.Protocol, corpus.Envelope.ProtocolMajor)
				}
			}
		}
	}
}

func TestPayloadPaddingReproducesExactDeclaredByteCounts(t *testing.T) {
	corpus := LoadCorpus(t)
	boundary := corpus.TransportBoundary
	atLimit := FrameSpec{
		Type: "query", ProfileID: corpus.Identity.ProfileID,
		Payload:        json.RawMessage(`{"query":"alpha","result_limit":10,"current_window_id":1}`),
		PayloadPadding: &Padding{Field: "note", TotalJSONPayloadBytes: boundary.MaxJSONPayloadBytes, Filler: "x"},
	}
	overLimit := atLimit
	over := boundary.MaxJSONPayloadBytes + 1
	overLimit.PayloadPadding = &Padding{Field: "note", TotalJSONPayloadBytes: over, Filler: "x"}

	exact := Materialize(corpus, atLimit)
	if int64(exact.PayloadBytes) != boundary.MaxJSONPayloadBytes {
		t.Fatalf("at-limit payload = %d bytes, want exactly %d", exact.PayloadBytes, boundary.MaxJSONPayloadBytes)
	}
	if int64(exact.DeclaredLen) >= boundary.MaxFrameBytes {
		t.Fatal("the at-limit frame must stay inside the transport boundary")
	}
	if _, err := Shape(exact.Bytes[4:]); err != nil {
		t.Fatalf("the at-limit frame is not a single JSON object: %v", err)
	}
	overWire := Materialize(corpus, overLimit)
	if int64(overWire.PayloadBytes) <= boundary.MaxJSONPayloadBytes {
		t.Fatalf("one-byte-over payload = %d bytes, must exceed %d", overWire.PayloadBytes, boundary.MaxJSONPayloadBytes)
	}
	if int64(overWire.PayloadBytes)-boundary.MaxJSONPayloadBytes != 1 {
		t.Fatalf("over-limit payload = %d bytes, want exactly one byte over %d", overWire.PayloadBytes, boundary.MaxJSONPayloadBytes)
	}
	if int64(overWire.DeclaredLen) >= boundary.MaxFrameBytes {
		t.Fatalf("one-byte-over payload frame = %d bytes, must stay deliverable inside the %d byte transport boundary", overWire.DeclaredLen, boundary.MaxFrameBytes)
	}
}

func TestLimitPrecedenceIsDeclaredAndConsistentWithTheByteBoundaries(t *testing.T) {
	corpus := LoadCorpus(t)
	if len(corpus.LimitPrecedence) == 0 {
		t.Fatal("no limit precedence is declared, so a rejected over-limit frame would be ambiguous")
	}
	seen := map[string]bool{}
	for _, entry := range corpus.LimitPrecedence {
		if seen[entry.Case] {
			t.Fatalf("limit precedence case %q is duplicated", entry.Case)
		}
		seen[entry.Case] = true
		if entry.Owner != "IP-06" && entry.Owner != "IP-07" {
			t.Fatalf("limit case %q owner = %q, want IP-06 or IP-07", entry.Case, entry.Owner)
		}
		if code, ok := optionalCode(entry.Code); ok && !declaredErrorCodes()[code] {
			t.Fatalf("limit case %q code %q is outside the wire vocabulary", entry.Case, code)
		}
		if !entry.ReachableOnWire && strings.TrimSpace(entry.Note) == "" {
			t.Fatalf("limit case %q is declared unreachable without explaining why", entry.Case)
		}
	}
	if !seen["frame_bytes_over_1048576"] || !seen["json_payload_over_262144"] {
		t.Fatal("the transport and payload precedence cases must both be declared")
	}
	if owner := corpus.TransportBoundary.OverFrameBoundaryOwner; owner != "IP-06" {
		t.Fatalf("the over-frame owner must stay IP-06, got %q", owner)
	}
}

func TestSnapshotRecordCapIsDefenceInDepthBehindThePayloadCap(t *testing.T) {
	corpus := LoadCorpus(t)
	// A full eligible-tab record cannot fit inside the JSON payload cap at the
	// declared record ceiling, so the payload cap always binds first. The corpus
	// must state that instead of implying a reachable wire rejection.
	count := corpus.Limits.SnapshotMaxRecords
	wire := Materialize(corpus, FrameSpec{
		Type: "snapshot", ProfileID: corpus.Identity.ProfileID,
		Payload:           json.RawMessage(`{"sequence":1,"tabs":[]}`),
		RepeatRecordCount: &count,
	})
	if int64(wire.PayloadBytes) <= corpus.TransportBoundary.MaxJSONPayloadBytes {
		t.Fatalf("%d records encode to %d bytes, which would make the record cap reachable and contradict the declared precedence",
			count, wire.PayloadBytes)
	}
	declared := false
	for _, entry := range corpus.LimitPrecedence {
		if entry.Case == "snapshot_records_over_10000" {
			declared = true
			if entry.ReachableOnWire {
				t.Fatal("the snapshot record cap is declared wire reachable, but the payload cap binds first")
			}
			if strings.TrimSpace(entry.Note) == "" {
				t.Fatal("the unreachable record cap must explain the binding order")
			}
		}
	}
	if !declared {
		t.Fatal("the snapshot record cap precedence is not declared in the corpus")
	}
}

func TestRecordAndOperationCountsLandOnTheDeclaredBoundaries(t *testing.T) {
	corpus := LoadCorpus(t)
	limits := corpus.Limits
	build := func(count int64, operation bool) WireFrame {
		spec := FrameSpec{Type: "snapshot", ProfileID: corpus.Identity.ProfileID, Payload: json.RawMessage(`{"sequence":1,"tabs":[]}`)}
		if operation {
			spec.Type = "delta"
			spec.Payload = json.RawMessage(`{"base_revision":0,"sequence_start":1,"sequence_end":1,"operations":[]}`)
			spec.RepeatOperationCount = &count
		} else {
			spec.RepeatRecordCount = &count
		}
		wire := Materialize(corpus, spec)
		shape, err := Shape(wire.Bytes[4:])
		if err != nil {
			t.Fatal(err)
		}
		if len(shape.Payload) == 0 {
			t.Fatalf("frame materialized without a payload object")
		}
		if operation {
			ops, ok := shape.Payload["operations"].([]any)
			if !ok {
				t.Fatalf("delta payload has no operations array; keys=%v", shape.Payload)
			}
			if got := int64(len(ops)); got != count {
				t.Fatalf("delta operations = %d, want %d", got, count)
			}
			return wire
		}
		tabs, ok := shape.Payload["tabs"].([]any)
		if !ok {
			t.Fatalf("snapshot payload has no tabs array; keys=%v", shape.Payload)
		}
		if got := int64(len(tabs)); got != count {
			t.Fatalf("snapshot records = %d, want %d", got, count)
		}
		return wire
	}
	if got := build(limits.SnapshotMaxRecords, false); int64(len(got.Bytes)) <= 0 {
		t.Fatal("at-limit snapshot produced no bytes")
	}
	if got := build(limits.SnapshotMaxRecords+1, false); int64(len(got.Bytes)) <= 0 {
		t.Fatal("over-limit snapshot produced no bytes")
	}
	build(limits.DeltaMaxOperations, true)
	build(limits.DeltaMaxOperations+1, true)
}

func TestQueryScalarsLandOnTheDeclaredBoundary(t *testing.T) {
	corpus := LoadCorpus(t)
	atLimit := corpus.Limits.QueryMaxScalars
	over := atLimit + 1
	for _, tc := range []struct {
		scalars int64
		want    int
	}{{atLimit, int(atLimit)}, {over, int(over)}} {
		count := tc.scalars
		wire := Materialize(corpus, FrameSpec{
			Type: "query", ProfileID: corpus.Identity.ProfileID,
			Payload:      json.RawMessage(`{"query":"a","result_limit":10,"current_window_id":1}`),
			QueryScalars: &count,
		})
		shape, err := Shape(wire.Bytes[4:])
		if err != nil {
			t.Fatal(err)
		}
		query, _ := shape.Payload["query"].(string)
		if got := ScalarCount(query); got != tc.want {
			t.Fatalf("query scalars = %d, want %d", got, tc.want)
		}
	}
}

func TestFixtureOutputsNeverCarrySensitiveSentinelValues(t *testing.T) {
	corpus := LoadCorpus(t)
	sentinels := corpus.PrivacySentinels
	needles := []string{sentinels.Title, sentinels.URL, sentinels.Query, sentinels.Token, sentinels.PageMarker}
	for _, needle := range needles {
		if strings.TrimSpace(needle) == "" {
			t.Fatal("privacy sentinel set is incomplete")
		}
	}
	raw, err := os.ReadFile(Artifact)
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
		encoded := strings.ToLower(string(mustJSONBytes(fixture["expected"])))
		for _, needle := range needles {
			if strings.Contains(encoded, strings.ToLower(needle)) {
				t.Fatalf("%s: expected output restates the sensitive sentinel %q", id, needle)
			}
		}
	}
}

func mustJSONBytes(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return raw
}

func optionalCode(code *string) (string, bool) {
	if code == nil || *code == "" {
		return "", false
	}
	return *code, true
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestSharedCatalogKeepsTheIP07ReconnectSeam(t *testing.T) {
	raw, err := os.ReadFile("../../fixtures/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Fixtures   []map[string]any `json:"fixtures"`
		PhaseSeams []struct {
			Phase             string   `json:"phase"`
			FixtureIDs        []string `json:"fixture_ids"`
			OwnershipRelation string   `json:"ownership_relation"`
			FixtureSignal     string   `json:"fixture_signal"`
		} `json:"phase_seams"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Fixtures) != 9 {
		t.Fatalf("shared catalog fixture count = %d, want the fixed 9 IDs", len(catalog.Fixtures))
	}
	seams := 0
	for _, seam := range catalog.PhaseSeams {
		if seam.Phase != "IP-07" {
			continue
		}
		seams++
		if seam.OwnershipRelation != "supporting" || seam.FixtureSignal != "snapshot_requested_after_reconnect" {
			t.Fatalf("IP-07 seam drifted: %+v", seam)
		}
		if !contains(seam.FixtureIDs, "FX-RECONNECT") {
			t.Fatalf("IP-07 seam lost FX-RECONNECT: %v", seam.FixtureIDs)
		}
	}
	if seams != 1 {
		t.Fatalf("IP-07 phase seam count = %d, want 1", seams)
	}
	// NM-PROTO-* are phase-local fixtures. The shared catalog's reconnect fixture
	// stays owned by IP-09 with IP-07 as a supporting seam.
	owned := 0
	for _, fixture := range catalog.Fixtures {
		if fixture["owner_phase"] == "IP-07" {
			owned++
		}
		if fixture["fixture_id"] == "FX-RECONNECT" && fixture["owner_phase"] != "IP-09" {
			t.Fatalf("FX-RECONNECT owner drifted to %v, want IP-09", fixture["owner_phase"])
		}
	}
	if owned != 0 {
		t.Fatalf("shared catalog declares %d fixtures owned by IP-07, want phase-local fixtures only", owned)
	}
}

func TestFixtureEnvelopeShapesAreSelfConsistent(t *testing.T) {
	corpus := LoadCorpus(t)
	for _, fixture := range corpus.Fixtures {
		requestIDs := map[string]bool{}
		for _, spec := range fixture.InputFrames {
			if spec.RequestID == nil {
				continue
			}
			id := *spec.RequestID
			if int64(len(id)) > corpus.Envelope.RequestIDMaxBytes {
				t.Fatalf("%s: request id %q exceeds the declared maximum", fixture.FixtureID, id)
			}
			if id != "" && requestIDs[id] && fixture.FixtureID != "NM-PROTO-009" {
				t.Fatalf("%s: request id %q is reused outside the duplicate-ledger fixture", fixture.FixtureID, id)
			}
			requestIDs[id] = true
			if int64(len(spec.ProfileID)) > corpus.Envelope.ProfileIDMaxBytes {
				t.Fatalf("%s: profile id exceeds the declared maximum", fixture.FixtureID)
			}
			if spec.Type != "" && int64(len(spec.Type)) > corpus.Limits.MessageTypeMaxBytes {
				t.Fatalf("%s: message type %q exceeds the declared maximum", fixture.FixtureID, spec.Type)
			}
			if spec.ProjectionRevision != nil && *spec.ProjectionRevision < corpus.Envelope.RevisionMinimum {
				t.Fatalf("%s: projection revision %d is below the declared minimum", fixture.FixtureID, *spec.ProjectionRevision)
			}
			if spec.Protocol != nil && *spec.Protocol == corpus.Envelope.ProtocolMajor {
				t.Fatalf("%s: a case must not declare a supported protocol version", fixture.FixtureID)
			}
		}
	}
}

func TestDeadlinesAreOrderedByDependency(t *testing.T) {
	corpus := LoadCorpus(t)
	deadlines := corpus.Deadlines
	if deadlines.FrameReadMS <= 0 || deadlines.HelloMS <= 0 {
		t.Fatal("frame read and hello budgets must be positive")
	}
	if deadlines.SnapshotApplyMS <= 0 || deadlines.DeltaApplyMS <= 0 {
		t.Fatal("sync apply budgets must be positive")
	}
	if deadlines.ShutdownDrainMS <= 0 || deadlines.HealthMS <= 0 || deadlines.ActivationMS <= 0 {
		t.Fatal("shutdown, health, and activation budgets must be positive")
	}
	if deadlines.SnapshotApplyMS >= deadlines.HelloMS {
		t.Fatalf("sync apply budget (%d ms) must stay inside the hello budget (%d ms)", deadlines.SnapshotApplyMS, deadlines.HelloMS)
	}
	if deadlines.QueryHostBudgetMS >= deadlines.QueryRequestMS {
		t.Fatalf("host query budget (%d ms) must stay inside the request budget (%d ms)", deadlines.QueryHostBudgetMS, deadlines.QueryRequestMS)
	}
	if _, err := strconv.ParseInt(strings.TrimSpace(corpus.Identity.HostVersion), 10, 64); err == nil {
		t.Fatalf("host version %q must be a semantic version, not a bare number", corpus.Identity.HostVersion)
	}
}
