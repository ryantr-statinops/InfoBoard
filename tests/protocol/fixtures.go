// Package protocol holds the deterministic IP-07 fixture corpus and the byte
// scaffolding needed to replay it. The host runtime owns the real length-prefix
// reader and serialized writer (IP-06), and host/internal/protocol owns every
// framing, envelope, limit, and state decision. Nothing here validates a frame:
// it only builds declared fixture bytes and splits captured stdout so a test can
// assert what the protocol layer emitted.
package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Artifact is the single shared IP-07 fixture corpus. Go host tests, the
// extension client test, and later phase consumers all read this one file.
const Artifact = "../../fixtures/protocol/phase-07.json"

// RequiredFixtureIDs is the IP-07 obligation from IP-07-T09.
var RequiredFixtureIDs = []string{
	"NM-PROTO-001", "NM-PROTO-002", "NM-PROTO-003", "NM-PROTO-004",
	"NM-PROTO-005", "NM-PROTO-006", "NM-PROTO-007", "NM-PROTO-008",
	"NM-PROTO-009", "NM-PROTO-010", "NM-PROTO-011", "NM-PROTO-012",
	"NM-PROTO-013",
}

type FrameEncoding struct {
	Owner        string `json:"owner"`
	LengthPrefix int    `json:"length_prefix_bytes"`
	ByteOrder    string `json:"byte_order"`
	Payload      string `json:"payload"`
	Note         string `json:"note"`
}

type TransportBoundary struct {
	MaxFrameBytes           int64  `json:"max_frame_bytes"`
	MaxJSONPayloadBytes     int64  `json:"max_json_payload_bytes"`
	FrameZeroLengthRejected bool   `json:"frame_zero_length_rejected"`
	LimitBeforeDecode       bool   `json:"payload_limit_enforced_before_decode"`
	OverFrameBoundaryOwner  string `json:"over_frame_boundary_owner"`
}

type Envelope struct {
	RequiredFields        []string `json:"required_fields"`
	UnknownFieldsRejected bool     `json:"unknown_fields_rejected"`
	DuplicateKeysRejected bool     `json:"duplicate_keys_rejected"`
	TrailingBytesRejected bool     `json:"trailing_bytes_rejected"`
	ProtocolMajor         int64    `json:"protocol_major"`
	RequestIDMaxBytes     int64    `json:"request_id_max_bytes"`
	ProfileIDMaxBytes     int64    `json:"profile_id_max_bytes"`
	RevisionMinimum       int64    `json:"projection_revision_min"`
	ProfileIdentitySource string   `json:"profile_identity_source"`
}

type Limits struct {
	QueryMaxScalars      int64 `json:"query_max_scalars"`
	SnapshotMaxRecords   int64 `json:"snapshot_max_records"`
	DeltaMaxOperations   int64 `json:"delta_max_operations"`
	QueryMaxResults      int64 `json:"query_max_results"`
	TitleMaxScalars      int64 `json:"title_max_scalars"`
	DomainMaxBytes       int64 `json:"domain_max_bytes"`
	URLMaxBytes          int64 `json:"url_max_bytes"`
	CapabilitiesMaxCount int64 `json:"capabilities_max_count"`
	MessageTypeMaxBytes  int64 `json:"message_type_max_bytes"`
	SnapshotTabIDMaximum int64 `json:"snapshot_tab_id_max"`
}

type Deadlines struct {
	FrameReadMS       int64 `json:"frame_read"`
	HelloMS           int64 `json:"hello"`
	SnapshotApplyMS   int64 `json:"snapshot_apply"`
	DeltaApplyMS      int64 `json:"delta_apply"`
	QueryHostBudgetMS int64 `json:"query_host_budget"`
	QueryRequestMS    int64 `json:"query_request_budget"`
	HealthMS          int64 `json:"health"`
	ActivationMS      int64 `json:"activation_status"`
	ShutdownDrainMS   int64 `json:"shutdown_drain"`
}

// Budget resolves a declared deadline name to a duration so assertions never
// restate a numeric constant.
func (deadlines Deadlines) Budget(name string) (time.Duration, bool) {
	for key, value := range map[string]int64{
		"frame_read":           deadlines.FrameReadMS,
		"hello":                deadlines.HelloMS,
		"snapshot_apply":       deadlines.SnapshotApplyMS,
		"delta_apply":          deadlines.DeltaApplyMS,
		"query_host_budget":    deadlines.QueryHostBudgetMS,
		"query_request_budget": deadlines.QueryRequestMS,
		"health":               deadlines.HealthMS,
		"activation_status":    deadlines.ActivationMS,
		"shutdown_drain":       deadlines.ShutdownDrainMS,
	} {
		if key == name {
			return time.Duration(value) * time.Millisecond, true
		}
	}
	return 0, false
}

type MessageType struct {
	Type          string   `json:"type"`
	Direction     string   `json:"direction"`
	PayloadFields []string `json:"payload_fields"`
}

type LimitPrecedence struct {
	Case            string  `json:"case"`
	Owner           string  `json:"owner"`
	Code            *string `json:"code"`
	ReachableOnWire bool    `json:"reachable_on_the_wire"`
	Note            string  `json:"note"`
}

type PreState struct {
	State                   string  `json:"state"`
	BoundProfileID          *string `json:"bound_profile_id"`
	ProjectionRevision      int64   `json:"projection_revision"`
	ProjectionSequence      int64   `json:"projection_sequence"`
	IndexedTabIDs           []int64 `json:"indexed_tab_ids"`
	IndexState              string  `json:"index_state"`
	StorageState            string  `json:"storage_state"`
	PreviousSessionID       string  `json:"previous_session_id"`
	PreviousProjectionEpoch string  `json:"previous_projection_epoch"`
}

type Redaction struct {
	CarriesRawTabFields          bool `json:"carries_raw_tab_fields"`
	EchoesRequestIDWhenParseable bool `json:"echoes_request_id_when_parseable"`
	MessageKeyIsClosedVocabulary bool `json:"message_key_is_closed_vocabulary"`
}

type Padding struct {
	Field                 string `json:"field"`
	TotalJSONPayloadBytes int64  `json:"total_json_payload_bytes"`
	Filler                string `json:"filler"`
}

type FrameSpec struct {
	Case                 string          `json:"case"`
	RequestID            *string         `json:"request_id"`
	Type                 string          `json:"type"`
	Protocol             *int64          `json:"protocol"`
	ProfileID            string          `json:"profile_id"`
	ProjectionRevision   *int64          `json:"projection_revision"`
	Payload              json.RawMessage `json:"payload"`
	RawPayloadText       *string         `json:"raw_payload_text"`
	PayloadPadding       *Padding        `json:"payload_padding"`
	RepeatRecordCount    *int64          `json:"repeat_record_count"`
	RepeatOperationCount *int64          `json:"repeat_operation_count"`
	QueryScalars         *int64          `json:"query_scalars"`
	RepeatCount          int64           `json:"repeat_count"`
	TruncateBytes        *int64          `json:"truncate_bytes"`
	HoldResponse         bool            `json:"hold_response"`
	ThenEOF              bool            `json:"then_eof"`
	BlockBeyondDeadline  string          `json:"block_beyond_deadline"`
	ExpectedErrorCode    *string         `json:"expected_error_code"`
}

type EmittedFrame struct {
	Type               string   `json:"type"`
	Code               *string  `json:"code"`
	PayloadFields      []string `json:"payload_fields"`
	EmitCount          int64    `json:"emit_count"`
	IdenticalBytes     bool     `json:"identical_bytes"`
	StorageState       string   `json:"storage_state"`
	DegradedAnnotation string   `json:"degraded_annotation"`
}

type Expected struct {
	EmittedFrames                      []EmittedFrame `json:"emitted_frames"`
	State                              string         `json:"state"`
	ClientState                        string         `json:"client_state"`
	ErrorCode                          *string        `json:"error_code"`
	Retryable                          bool           `json:"retryable"`
	Availability                       string         `json:"availability"`
	DegradedCode                       string         `json:"degraded_code"`
	LexicalResultsAvailable            bool           `json:"lexical_results_available"`
	TabDataAccepted                    bool           `json:"tab_data_accepted"`
	ProjectionRevision                 int64          `json:"projection_revision"`
	ProjectionSequence                 int64          `json:"projection_sequence"`
	BoundProfileID                     string         `json:"bound_profile_id"`
	IndexedTabIDs                      []int64        `json:"indexed_tab_ids"`
	IndexedIDsEqualSnapshotIDs         bool           `json:"indexed_ids_equal_snapshot_ids"`
	RequestLedgerSize                  int64          `json:"request_ledger_size"`
	SideEffectsExecuted                int64          `json:"side_effects_executed"`
	DuplicateLedgerHits                int64          `json:"duplicate_ledger_hits"`
	ResyncReasons                      []string       `json:"resync_reasons"`
	StaleDeltaRejected                 int64          `json:"stale_delta_rejected"`
	OutOfOrderOperationsVisible        int64          `json:"out_of_order_operations_visible"`
	OverLimitRecordsIndexed            int64          `json:"over_limit_records_indexed"`
	CrossProfileRecordsRejected        int64          `json:"cross_profile_records_rejected"`
	CrossProfileMutations              int64          `json:"cross_profile_mutations"`
	StaleResultsServed                 int64          `json:"stale_results_served"`
	PartialResultsReturned             int64          `json:"partial_results_returned"`
	PartialApplication                 bool           `json:"partial_application"`
	RejectedBeforeAllocation           bool           `json:"rejected_before_allocation"`
	FramesAfterDisconnectAccepted      int64          `json:"frames_after_disconnect_accepted"`
	FramesAfterCloseAccepted           int64          `json:"frames_after_close_accepted"`
	SecondPayloadApplied               bool           `json:"second_payload_applied"`
	SessionClosedAfterError            bool           `json:"session_closed_after_error"`
	SessionIdentityCleared             bool           `json:"session_identity_cleared"`
	InFlightFailed                     int64          `json:"in_flight_failed"`
	RequiresNewHello                   bool           `json:"requires_new_hello"`
	RequiresNewSnapshot                bool           `json:"requires_new_snapshot"`
	ReachedReadyWithoutReinstall       bool           `json:"reached_ready_without_reinstall"`
	SessionIDDiffersFromPrevious       bool           `json:"session_id_differs_from_previous"`
	ProjectionEpochDiffersFromPrevious bool           `json:"projection_epoch_differs_from_previous"`
	RecoveryAction                     string         `json:"recovery_action"`
	NoSideEffect                       bool           `json:"no_side_effect"`
	Redaction                          Redaction      `json:"redaction"`
	BoundedCompletion                  bool           `json:"bounded_completion"`
}

type ProtocolFixture struct {
	FixtureID      string      `json:"fixture_id"`
	Title          string      `json:"title"`
	RequirementIDs []string    `json:"requirement_ids"`
	PreState       PreState    `json:"pre_state"`
	InputFrames    []FrameSpec `json:"input_frames"`
	Expected       Expected    `json:"expected"`
}

type Identity struct {
	ProfileID                 string   `json:"profile_id"`
	OtherProfileID            string   `json:"other_profile_id"`
	ProjectionEpoch           string   `json:"projection_epoch"`
	SecondProjectionEpoch     string   `json:"second_projection_epoch"`
	HostVersion               string   `json:"host_version"`
	ExtensionVersion          string   `json:"extension_version"`
	BrowserFamily             string   `json:"browser_family"`
	BrowserVersion            string   `json:"browser_version"`
	RankingModelVersion       string   `json:"ranking_model_version"`
	ContextKind               string   `json:"context_kind"`
	Capabilities              []string `json:"capabilities"`
	SupportedProtocolVersions []int64  `json:"supported_protocol_versions"`
	TabIDs                    []int64  `json:"tab_ids"`
	ReplacementTabIDs         []int64  `json:"replacement_tab_ids"`
}

type PrivacySentinels struct {
	Title      string `json:"title"`
	URL        string `json:"url"`
	Query      string `json:"query"`
	Token      string `json:"token"`
	PageMarker string `json:"page_marker"`
}

type Corpus struct {
	SchemaVersion       int               `json:"schema_version"`
	Phase               string            `json:"phase"`
	Artifact            string            `json:"artifact"`
	RequirementIDs      []string          `json:"requirement_ids"`
	SharedConsumers     []string          `json:"shared_consumers"`
	ContractSource      string            `json:"contract_source"`
	FrameEncoding       FrameEncoding     `json:"frame_encoding"`
	TransportBoundary   TransportBoundary `json:"transport_boundary"`
	Envelope            Envelope          `json:"envelope"`
	Limits              Limits            `json:"limits"`
	Deadlines           Deadlines         `json:"deadlines_ms"`
	ErrorCodes          []string          `json:"error_codes"`
	ResyncReasons       []string          `json:"resync_reasons"`
	MessageTypes        []MessageType     `json:"message_types"`
	ErrorOptionalFields []string          `json:"error_optional_fields"`
	SessionStates       []string          `json:"session_states"`
	ClientStates        []string          `json:"client_states"`
	DeltaOperations     []string          `json:"delta_operations"`
	LimitPrecedence     []LimitPrecedence `json:"limit_precedence"`
	Identity            Identity          `json:"identity"`
	PrivacySentinels    PrivacySentinels  `json:"privacy_sentinels"`
	Fixtures            []ProtocolFixture `json:"fixtures"`
}

// Fataler is the reporting seam the corpus loader needs from a test handle.
type Fataler interface {
	Helper()
	Fatalf(format string, args ...any)
}

// LoadCorpus reads the shared IP-07 artifact.
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
func (corpus Corpus) Fixture(t Fataler, id string) ProtocolFixture {
	t.Helper()
	for _, fixture := range corpus.Fixtures {
		if fixture.FixtureID == id {
			return fixture
		}
	}
	t.Fatalf("fixture %s is absent from %s", id, Artifact)
	return ProtocolFixture{}
}

// MessageTypeIndex returns the declared message contract keyed by type name.
func (corpus Corpus) MessageTypeIndex() map[string]MessageType {
	index := map[string]MessageType{}
	for _, entry := range corpus.MessageTypes {
		index[entry.Type] = entry
	}
	return index
}

// IsErrorCode reports whether code belongs to the exact wire error vocabulary.
func (corpus Corpus) IsErrorCode(code string) bool {
	for _, known := range corpus.ErrorCodes {
		if known == code {
			return true
		}
	}
	return false
}

// WireFrame is one fixture frame materialized onto the browser-provided stream.
// Payload stays raw so no consumer re-encodes or reorders declared bytes.
type WireFrame struct {
	Spec         FrameSpec
	DeclaredLen  uint32
	PayloadBytes int
	Bytes        []byte
	Withheld     int
	RepeatCount  int
}

// Materialize builds the deterministic declared bytes for one fixture frame.
// It applies only fixture-shaping rules from the artifact: exact payload padding,
// generated record/operation counts, exact query scalar counts, and declared
// truncation. Every limit decision remains the protocol layer's job.
func Materialize(corpus Corpus, spec FrameSpec) WireFrame {
	var payload []byte
	if spec.RawPayloadText != nil {
		payload = []byte(*spec.RawPayloadText)
	}
	if payload == nil {
		object := map[string]any{}
		if len(spec.Payload) > 0 {
			if err := json.Unmarshal(spec.Payload, &object); err != nil {
				panic(fmt.Sprintf("fixture payload is not an object: %v", err))
			}
		}
		applyPadding(object, spec.PayloadPadding)
		applyQueryScalars(object, spec.QueryScalars)
		applyRecordCount(object, spec.RepeatRecordCount)
		applyOperationCount(object, spec.RepeatOperationCount)
		encoded, err := json.Marshal(object)
		if err != nil {
			panic(fmt.Sprintf("fixture payload is not encodable: %v", err))
		}
		payload = encoded
	}

	// A declared semantic frame is materialized as the full six-field envelope
	// the protocol contract requires; a declared raw frame stays verbatim.
	envelope := payload
	if spec.RawPayloadText == nil {
		protocol := corpus.Envelope.ProtocolMajor
		if spec.Protocol != nil {
			protocol = *spec.Protocol
		}
		revision := corpus.Envelope.RevisionMinimum
		if spec.ProjectionRevision != nil {
			revision = *spec.ProjectionRevision
		}
		requestID := ""
		if spec.RequestID != nil {
			requestID = *spec.RequestID
		}
		object := map[string]any{}
		if err := json.Unmarshal(payload, &object); err != nil {
			panic(fmt.Sprintf("fixture payload is not an object: %v", err))
		}
		envelopeObject := map[string]any{
			"protocol":            protocol,
			"type":                spec.Type,
			"request_id":          requestID,
			"profile_id":          spec.ProfileID,
			"projection_revision": revision,
			"payload":             object,
		}
		encoded, err := json.Marshal(envelopeObject)
		if err != nil {
			panic(fmt.Sprintf("fixture envelope is not encodable: %v", err))
		}
		envelope = encoded
	}

	frame := encodeFrame(envelope)
	withheld := 0
	if spec.TruncateBytes != nil && *spec.TruncateBytes > 0 {
		limit := len(frame) - int(*spec.TruncateBytes)
		if limit < 4 {
			limit = 4
		}
		withheld = len(frame) - limit
		frame = frame[:limit]
	}
	repeats := spec.RepeatCount
	if repeats < 1 {
		repeats = 1
	}
	return WireFrame{
		Spec:         spec,
		DeclaredLen:  uint32(len(envelope)),
		PayloadBytes: len(payload),
		Bytes:        frame,
		Withheld:     withheld,
		RepeatCount:  int(repeats),
	}
}

// MaterializeAll builds every declared input frame of a fixture in order.
func MaterializeAll(corpus Corpus, fixture ProtocolFixture) []WireFrame {
	frames := make([]WireFrame, 0, len(fixture.InputFrames))
	for _, spec := range fixture.InputFrames {
		wire := Materialize(corpus, spec)
		for repeat := 0; repeat < wire.RepeatCount; repeat++ {
			frames = append(frames, wire)
		}
	}
	return frames
}

// encodeFrame applies the browser-provided length prefix. This is byte
// scaffolding for declared fixture input, not a second stream reader.
func encodeFrame(payload []byte) []byte {
	frame := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(frame[:4], uint32(len(payload)))
	copy(frame[4:], payload)
	return frame
}

func applyPadding(object map[string]any, padding *Padding) {
	if padding == nil {
		return
	}
	object[padding.Field] = ""
	filler := padding.Filler
	if filler == "" {
		filler = "x"
	}
	if len(filler) != 1 || filler[0] > 0x7f {
		panic("fixture padding filler must be one ASCII byte so the encoded length is monotonic")
	}
	encoded := func(size int) int {
		object[padding.Field] = strings.Repeat(filler, size)
		raw, err := json.Marshal(object)
		if err != nil {
			panic(err)
		}
		return len(raw)
	}
	high := int(padding.TotalJSONPayloadBytes) + 1
	if int64(encoded(0)) > padding.TotalJSONPayloadBytes {
		// The unpadded object already exceeds the target; leave it exact so the
		// protocol layer decides whether it is over the declared limit.
		return
	}
	if int64(encoded(high)) < padding.TotalJSONPayloadBytes {
		object[padding.Field] = strings.Repeat(filler, high)
		return
	}
	low := 0
	for low < high {
		middle := (low + high) / 2
		if int64(encoded(middle)) < padding.TotalJSONPayloadBytes {
			low = middle + 1
		} else {
			high = middle
		}
	}
	object[padding.Field] = strings.Repeat(filler, low)
}

func applyQueryScalars(object map[string]any, scalars *int64) {
	if scalars == nil {
		return
	}
	object["query"] = strings.Repeat("a", int(*scalars))
}

func applyRecordCount(object map[string]any, count *int64) {
	if count == nil || *count <= 0 {
		return
	}
	tabs := make([]any, 0, *count)
	for index := int64(0); index < *count; index++ {
		tabs = append(tabs, map[string]any{
			"tab_id":         index,
			"window_id":      1,
			"title_display":  "T" + strconv.FormatInt(index, 10),
			"url_display":    "t" + strconv.FormatInt(index, 10) + ".example",
			"domain_display": "t" + strconv.FormatInt(index, 10) + ".example",
			"pinned":         false,
			"active":         false,
			"group_id":       nil,
			"context_kind":   "normal",
		})
	}
	object["tabs"] = tabs
}

func applyOperationCount(object map[string]any, count *int64) {
	if count == nil || *count <= 0 {
		return
	}
	operations := make([]any, 0, *count)
	for index := int64(0); index < *count; index++ {
		operations = append(operations, map[string]any{"operation": "remove", "tab_id": index})
	}
	object["operations"] = operations
}

// CapturedFrame is one frame split out of captured stdout by byte offset only.
type CapturedFrame struct {
	Offset  int
	Size    int
	Payload []byte
}

// SplitCaptured splits a captured stdout stream on the declared length prefix so
// a test can count and compare frames. It applies no validity rule; every
// fail-closed outcome must come from the protocol layer.
func SplitCaptured(raw []byte) ([]CapturedFrame, int) {
	frames := make([]CapturedFrame, 0, 4)
	offset := 0
	for offset+4 <= len(raw) {
		size := int(binary.LittleEndian.Uint32(raw[offset : offset+4]))
		if size < 0 || offset+4+size > len(raw) {
			break
		}
		payload := make([]byte, size)
		copy(payload, raw[offset+4:offset+4+size])
		frames = append(frames, CapturedFrame{Offset: offset, Size: 4 + size, Payload: payload})
		offset += 4 + size
	}
	return frames, len(raw) - offset
}

// EnvelopeShape is the minimal structural read a fixture needs to name the frame
// it observed. Field values are not validated here.
type EnvelopeShape struct {
	Protocol           int64          `json:"protocol"`
	Type               string         `json:"type"`
	RequestID          string         `json:"request_id"`
	ProfileID          string         `json:"profile_id"`
	ProjectionRevision int64          `json:"projection_revision"`
	Payload            map[string]any `json:"payload"`
}

// Shape reads the declared envelope keys of a captured payload.
func Shape(payload []byte) (EnvelopeShape, error) {
	var shape EnvelopeShape
	if err := json.Unmarshal(payload, &shape); err != nil {
		return shape, err
	}
	return shape, nil
}

// ErrorCode extracts the declared error code from a captured error frame payload.
func ErrorCode(payload []byte) string {
	var envelope struct {
		Payload struct {
			Code string `json:"code"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return ""
	}
	return envelope.Payload.Code
}

// SortedIDs returns a stable copy of an identity slice for order-free assertions.
func SortedIDs(ids []int64) []int64 {
	out := append([]int64(nil), ids...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// ScalarCount reports the Unicode scalar count of a value for bounded-field checks.
func ScalarCount(value string) int { return utf8.RuneCountInString(value) }
