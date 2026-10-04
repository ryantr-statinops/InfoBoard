package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// BrowserFamily is the supported Chromium desktop family enum.
type BrowserFamily string

const (
	BrowserChrome BrowserFamily = "chrome"
	BrowserEdge   BrowserFamily = "edge"
)

// KnownBrowser reports whether a family is part of the supported enum.
func (family BrowserFamily) KnownBrowser() bool {
	return family == BrowserChrome || family == BrowserEdge
}

// TabRecord is the wire shape of one eligible open-tab record. Its field names
// match the IP-02 domain projection so the host maps without redefinition, and
// it carries no page content, cookie, or storage data.
type TabRecord struct {
	ProfileID          string             `json:"profile_id"`
	ContextKind        domain.ContextKind `json:"context_kind"`
	TabID              int64              `json:"tab_id"`
	WindowID           int64              `json:"window_id"`
	GroupID            *int64             `json:"group_id,omitempty"`
	TitleDisplay       string             `json:"title_display"`
	TitleSearch        string             `json:"title_search"`
	URLDisplay         string             `json:"url_display"`
	URLSearch          string             `json:"url_search"`
	DomainDisplay      string             `json:"domain_display"`
	DomainSearch       string             `json:"domain_search"`
	WindowLabelDisplay *string            `json:"window_label_display,omitempty"`
	WindowLabelSearch  *string            `json:"window_label_search,omitempty"`
	GroupLabelDisplay  *string            `json:"group_label_display,omitempty"`
	GroupLabelSearch   *string            `json:"group_label_search,omitempty"`
	Pinned             bool               `json:"pinned"`
	Active             bool               `json:"active"`
	Eligible           bool               `json:"eligible"`
	ObservedAt         int64              `json:"observed_at"`
	ProjectionEpoch    string             `json:"projection_epoch"`
	ProjectionRevision uint64             `json:"projection_revision"`
}

// Hello is the extension to host handshake payload. The envelope profile_id is
// the sole profile identity, so this payload never repeats it.
type Hello struct {
	ExtensionVersion   string             `json:"extension_version"`
	BrowserFamily      BrowserFamily      `json:"browser_family"`
	BrowserVersion     string             `json:"browser_version"`
	ContextKind        domain.ContextKind `json:"context_kind"`
	Capabilities       []string           `json:"capabilities"`
	SupportedProtocols []uint64           `json:"supported_protocols"`
}

// SupportsCurrentProtocol reports whether the client offered the current major
// version. A client that did not is incompatible, not merely degraded.
func (hello Hello) SupportsCurrentProtocol() bool {
	for _, version := range hello.SupportedProtocols {
		if version == CurrentVersion {
			return true
		}
	}
	return false
}

// DecodeHello validates the handshake payload before any session state changes.
func DecodeHello(payload []byte, limits Limits) (Hello, *Error) {
	var hello Hello
	failure := DecodePayloadObject(payload, map[string]fieldDecoder{
		"extension_version":           decodeStringField(&hello.ExtensionVersion, MaxOpaqueValueBytes, false),
		"browser_family":              decodeStringField((*string)(&hello.BrowserFamily), MaxOpaqueValueBytes, false),
		"browser_version":             decodeStringField(&hello.BrowserVersion, MaxOpaqueValueBytes, false),
		"context_kind":                decodeStringField((*string)(&hello.ContextKind), MaxOpaqueValueBytes, false),
		"capabilities":                decodeStringSliceField(&hello.Capabilities, limits.MaxCapabilities, MaxOpaqueValueBytes, false),
		"supported_protocol_versions": decodeUint64SliceField(&hello.SupportedProtocols, 8, false),
	})
	if failure != nil {
		return Hello{}, failure
	}
	if !hello.BrowserFamily.KnownBrowser() {
		return Hello{}, NewError(CodeProtocolMismatch)
	}
	if hello.ContextKind != domain.ContextNormal && hello.ContextKind != domain.ContextPrivate {
		return Hello{}, NewError(CodeInvalidFrame)
	}
	if len(hello.SupportedProtocols) == 0 {
		return Hello{}, NewError(CodeProtocolMismatch)
	}
	for _, version := range hello.SupportedProtocols {
		if version == 0 || version > uint64(CurrentVersion)+1 {
			return Hello{}, NewError(CodeProtocolMismatch)
		}
	}
	if !hello.SupportsCurrentProtocol() {
		return Hello{}, NewError(CodeProtocolMismatch)
	}
	return hello, nil
}

// HelloAck is the host acknowledgement payload.
type HelloAck struct {
	Protocol            int    `json:"protocol"`
	HostVersion         string `json:"host_version"`
	Limits              Limits `json:"limits"`
	RankingModelVersion string `json:"ranking_model_version"`
	Persistence         string `json:"persistence"`
}

// Snapshot is the authoritative extension to host projection payload. The
// envelope revision is the full projection revision.
type Snapshot struct {
	Tabs     []TabRecord `json:"tabs"`
	Sequence uint64      `json:"sequence"`
}

// DecodeSnapshot validates the authoritative snapshot before it is applied.
func DecodeSnapshot(payload []byte, limits Limits) (Snapshot, *Error) {
	var snapshot Snapshot
	var tabs json.RawMessage
	failure := DecodePayloadObject(payload, map[string]fieldDecoder{
		"tabs":     decodeRawArrayField(&tabs, limits.MaxSnapshotRecords, "snapshot_records"),
		"sequence": decodeUint64Field(&snapshot.Sequence, false),
	})
	if failure != nil {
		return Snapshot{}, failure
	}
	records, failure := decodeTabArray(tabs, limits)
	if failure != nil {
		return Snapshot{}, failure
	}
	snapshot.Tabs = records
	return snapshot, nil
}

func decodeTabArray(raw json.RawMessage, limits Limits) ([]TabRecord, *Error) {
	records, failure := decodeStrictArray[TabRecord](raw)
	if failure != nil {
		return nil, failure
	}
	for index := range records {
		if failure := records[index].Validate(limits); failure != nil {
			return nil, failure
		}
	}
	return records, nil
}

// Validate enforces the bounded record contract before any allocation-heavy
// indexing work.
func (record TabRecord) Validate(limits Limits) *Error {
	if !nonEmptyOpaque(record.ProfileID, limits.MaxProfileIDBytes) {
		return NewError(CodeInvalidFrame)
	}
	if record.ContextKind != domain.ContextNormal && record.ContextKind != domain.ContextPrivate {
		return NewError(CodeInvalidFrame)
	}
	if record.TabID < 0 || record.WindowID < 0 || record.GroupID != nil && *record.GroupID < 0 {
		return NewError(CodeInvalidFrame)
	}
	if record.ObservedAt < 0 || record.ProjectionRevision > domain.MaxProjectionRevision {
		return NewError(CodeInvalidFrame)
	}
	if !nonEmptyOpaque(record.ProjectionEpoch, MaxEpochBytes) {
		return NewError(CodeInvalidFrame)
	}
	if failure := record.validateText("title_display", record.TitleDisplay, limits.MaxTitleScalars, true); failure != nil {
		return failure
	}
	if failure := record.validateText("title_search", record.TitleSearch, limits.MaxTitleScalars, false); failure != nil {
		return failure
	}
	if failure := record.validateText("url_display", record.URLDisplay, limits.MaxURLBytes, true); failure != nil {
		return failure
	}
	if failure := record.validateText("url_search", record.URLSearch, limits.MaxURLBytes, false); failure != nil {
		return failure
	}
	if failure := record.validateText("domain_display", record.DomainDisplay, limits.MaxDomainBytes, true); failure != nil {
		return failure
	}
	if failure := record.validateText("domain_search", record.DomainSearch, limits.MaxDomainBytes, false); failure != nil {
		return failure
	}
	for _, optional := range []*string{
		record.WindowLabelDisplay, record.WindowLabelSearch,
		record.GroupLabelDisplay, record.GroupLabelSearch,
	} {
		if optional != nil && utf8.RuneCountInString(*optional) > limits.MaxTitleScalars {
			return NewError(CodePayloadLimit).WithBound("label_scalars", limits.MaxTitleScalars, utf8.RuneCountInString(*optional))
		}
	}
	return nil
}

func (record TabRecord) validateText(name, value string, bound int, required bool) *Error {
	if required && value == "" {
		return NewError(CodeInvalidFrame)
	}
	if utf8.RuneCountInString(value) > bound {
		return NewError(CodePayloadLimit).WithBound(name, bound, utf8.RuneCountInString(value))
	}
	if !utf8.ValidString(value) {
		return NewError(CodeInvalidFrame)
	}
	return nil
}

// Delta is the ordered extension to host change payload. The envelope revision
// is the target revision.
type Delta struct {
	BaseRevision  uint64      `json:"base_revision"`
	SequenceStart uint64      `json:"sequence_start"`
	SequenceEnd   uint64      `json:"sequence_end"`
	Operations    []Operation `json:"operations"`
}

// Operation is one ordered delta operation. The ordered kind travels on the
// wire as "operation"; a remove carries only the tab identity.
type Operation struct {
	Sequence uint64        `json:"sequence,omitempty"`
	Kind     OperationKind `json:"operation"`
	TabID    int64         `json:"tab_id"`
	Record   *TabRecord    `json:"record,omitempty"`
}

// DecodeDelta validates the delta payload and its contiguous sequence range.
func DecodeDelta(payload []byte, limits Limits) (Delta, *Error) {
	var (
		delta     Delta
		operation json.RawMessage
	)
	failure := DecodePayloadObject(payload, map[string]fieldDecoder{
		"base_revision":  decodeUint64Field(&delta.BaseRevision, false),
		"sequence_start": decodeUint64Field(&delta.SequenceStart, false),
		"sequence_end":   decodeUint64Field(&delta.SequenceEnd, false),
		"operations":     decodeRawArrayField(&operation, limits.MaxDeltaOperations, "delta_operations"),
	})
	if failure != nil {
		return Delta{}, failure
	}
	operations, failure := decodeOperationArray(operation, limits)
	if failure != nil {
		return Delta{}, failure
	}
	delta.Operations = operations
	if len(delta.Operations) == 0 {
		return Delta{}, NewError(CodeRevisionMismatch)
	}
	if delta.SequenceStart == 0 || delta.SequenceEnd < delta.SequenceStart {
		return Delta{}, NewError(CodeRevisionMismatch)
	}
	expected := delta.SequenceStart + uint64(len(delta.Operations)) - 1
	if expected != delta.SequenceEnd {
		return Delta{}, NewError(CodeRevisionMismatch)
	}
	// The declared range is the authority for ordering: each operation inherits
	// its sequence position from the contiguous range, so an out-of-order or
	// duplicated sequence cannot be smuggled through a per-operation field.
	for index := range delta.Operations {
		delta.Operations[index].Sequence = delta.SequenceStart + uint64(index)
	}
	return delta, nil
}

func decodeOperationArray(raw json.RawMessage, limits Limits) ([]Operation, *Error) {
	operations, failure := decodeStrictArray[Operation](raw)
	if failure != nil {
		return nil, failure
	}
	for index := range operations {
		if !operations[index].Kind.KnownOperation() {
			return nil, NewError(CodeInvalidFrame)
		}
		if operations[index].Kind == OperationRemove {
			if operations[index].Record != nil {
				return nil, NewError(CodeInvalidFrame)
			}
			continue
		}
		if operations[index].Record == nil {
			return nil, NewError(CodeInvalidFrame)
		}
		if failure := operations[index].Record.Validate(limits); failure != nil {
			return nil, failure
		}
	}
	return operations, nil
}

// SyncAck acknowledges an accepted snapshot or delta.
type SyncAck struct {
	AcceptedRevision uint64 `json:"accepted_revision"`
	AcceptedSequence uint64 `json:"accepted_sequence"`
}

// ResyncRequired tells the extension to send a new full snapshot.
type ResyncRequired struct {
	Reason           ResyncReason `json:"reason"`
	ExpectedRevision uint64       `json:"expected_revision"`
	ExpectedSequence uint64       `json:"expected_sequence"`
}

// Query is the extension to host query payload. The envelope revision is the
// revision the client believes is current.
type Query struct {
	Text            string `json:"query"`
	ResultLimit     int    `json:"result_limit"`
	CurrentWindowID int64  `json:"current_window_id"`
}

// DecodeQuery validates the query payload before any index work.
func DecodeQuery(payload []byte, limits Limits) (Query, *Error) {
	var query Query
	failure := DecodePayloadObject(payload, map[string]fieldDecoder{
		"query":             decodeStringField(&query.Text, limits.MaxQueryScalars*4, false),
		"result_limit":      decodeIntField(&query.ResultLimit, true),
		"current_window_id": decodeInt64Field(&query.CurrentWindowID, false),
	})
	if failure != nil {
		return Query{}, failure
	}
	if utf8.RuneCountInString(query.Text) > limits.MaxQueryScalars {
		return Query{}, NewError(CodePayloadLimit).WithBound("query_scalars", limits.MaxQueryScalars, utf8.RuneCountInString(query.Text))
	}
	if query.ResultLimit <= 0 || query.ResultLimit > limits.MaxResults {
		return Query{}, NewError(CodePayloadLimit).WithBound("result_limit", limits.MaxResults, query.ResultLimit)
	}
	if query.CurrentWindowID < 0 {
		return Query{}, NewError(CodeInvalidFrame)
	}
	return query, nil
}

// QueryResult is the host answer payload. It carries bounded display metadata
// and opaque identities only.
type QueryResult struct {
	Results             []ResultRow   `json:"results"`
	RankingModelVersion string        `json:"ranking_model_version"`
	ProjectionRevision  uint64        `json:"projection_revision"`
	Counters            QueryCounters `json:"counters"`
}

// QueryCounters are bounded counters plus the degraded annotation. A degraded
// store is announced here with its exact wire code while lexical results stay
// available.
type QueryCounters struct {
	Candidates   int64     `json:"candidates"`
	Returned     int64     `json:"returned"`
	TotalMicros  int64     `json:"total_micros"`
	DegradedCode ErrorCode `json:"degraded_code,omitempty"`
}

// ResultRow is one bounded, renderable result row.
type ResultRow struct {
	ResultID          string   `json:"result_id"`
	TabID             int64    `json:"tab_id"`
	WindowID          int64    `json:"window_id"`
	TitleDisplay      string   `json:"title_display"`
	DomainDisplay     string   `json:"domain_display"`
	GroupLabelDisplay *string  `json:"group_label_display,omitempty"`
	Pinned            bool     `json:"pinned"`
	Explanation       []string `json:"explanation,omitempty"`
}

// ResultReference is the opaque, bounded reference the extension echoes when it
// reports an activation. The host never resolves it into a browser call.
type ResultReference struct {
	TabID              int64  `json:"tab_id"`
	ProjectionEpoch    string `json:"projection_epoch"`
	ProjectionRevision uint64 `json:"projection_revision"`
	ResultID           string `json:"result_id"`
}

// ActivationStatus is the extension to host activation report. The extension
// owns activation; the host only records the outcome for recency metadata and
// diagnostics.
type ActivationStatus struct {
	Reference          ResultReference `json:"result_reference"`
	TabID              int64           `json:"tab_id"`
	ProjectionRevision uint64          `json:"projection_revision"`
	Outcome            string          `json:"outcome,omitempty"`
	ErrorCode          ErrorCode       `json:"error_code,omitempty"`
}

const (
	ActivationOutcomeActivated = "activated"
	ActivationOutcomeFailed    = "failed"
)

// DecodeActivationStatus validates the activation report payload for its
// message type.
func DecodeActivationStatus(payload []byte, messageType MessageType) (ActivationStatus, *Error) {
	var status ActivationStatus
	fields := map[string]fieldDecoder{
		"result_reference":    decodeResultReferenceField(&status.Reference),
		"tab_id":              decodeInt64Field(&status.TabID, true),
		"projection_revision": decodeUint64Field(&status.ProjectionRevision, false),
		"outcome":             decodeStringField(&status.Outcome, MaxOpaqueValueBytes, true),
		"error_code":          decodeErrorCodeField(&status.ErrorCode, true),
	}
	required := []string{"result_reference", "tab_id"}
	if messageType == TypeActivationFailed {
		required = append(required, "error_code")
	} else {
		required = append(required, "outcome")
	}
	if failure := decodePayload(payload, fields, required); failure != nil {
		return ActivationStatus{}, failure
	}
	if status.TabID < 0 || status.Reference.TabID < 0 {
		return ActivationStatus{}, NewError(CodeInvalidFrame)
	}
	if status.TabID != status.Reference.TabID || status.ProjectionRevision != status.Reference.ProjectionRevision {
		return ActivationStatus{}, NewError(CodeRevisionMismatch)
	}
	if !nonEmptyOpaque(status.Reference.ProjectionEpoch, MaxEpochBytes) || !nonEmptyOpaque(status.Reference.ResultID, MaxOpaqueValueBytes) {
		return ActivationStatus{}, NewError(CodeInvalidFrame)
	}
	if messageType == TypeActivationObserved && status.Outcome != ActivationOutcomeActivated {
		return ActivationStatus{}, NewError(CodeInvalidFrame)
	}
	if messageType == TypeActivationFailed && !status.ErrorCode.KnownCode() {
		return ActivationStatus{}, NewError(CodeInvalidFrame)
	}
	return status, nil
}

// ActivationAck acknowledges a recorded activation report.
type ActivationAck struct {
	Accepted           bool   `json:"accepted"`
	ProjectionRevision uint64 `json:"projection_revision"`
}

// HealthResult is the host health payload. It exposes availability and index
// freshness without any raw tab field.
type HealthResult struct {
	Availability       string         `json:"availability"`
	HostState          string         `json:"host_state"`
	ProtocolVersion    int            `json:"protocol_version"`
	SessionID          string         `json:"session_id"`
	StorageState       string         `json:"storage_state"`
	IndexState         string         `json:"index_state"`
	ProjectionRevision uint64         `json:"projection_revision"`
	ProjectionSequence uint64         `json:"projection_sequence"`
	FreshnessAgeMillis int64          `json:"freshness_age_ms"`
	Retryable          bool           `json:"retryable"`
	Counters           HealthCounters `json:"counters"`
}

// HealthCounters are the bounded runtime counters exposed to the surface.
type HealthCounters struct {
	FramesRead        uint64 `json:"frames_read"`
	FramesWritten     uint64 `json:"frames_written"`
	RequestsStarted   uint64 `json:"requests_started"`
	RequestsCompleted uint64 `json:"requests_completed"`
	RequestsRejected  uint64 `json:"requests_rejected"`
	Transitions       uint64 `json:"transitions"`
	ProtocolErrors    uint64 `json:"protocol_errors"`
}

// ErrorPayload is the required error response payload.
type ErrorPayload struct {
	Code             ErrorCode  `json:"code"`
	Retryable        bool       `json:"retryable"`
	MessageKey       MessageKey `json:"message_key"`
	RetryAfterMillis *int64     `json:"retry_after_ms,omitempty"`
	ExpectedRevision *uint64    `json:"expected_revision,omitempty"`
	ExpectedSequence *uint64    `json:"expected_sequence,omitempty"`
}

// PayloadOf renders a typed payload as an opaque envelope payload object.
func PayloadOf(value any) ([]byte, *Error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, NewError(CodeInternalFailure)
	}
	if len(encoded) == 0 || encoded[0] != '{' {
		return nil, NewError(CodeInternalFailure)
	}
	return encoded, nil
}

// EmptyPayload renders the empty JSON object used by request-only messages.
func EmptyPayload() []byte { return []byte(`{}`) }

func decodeResultReferenceField(target *ResultReference) fieldDecoder {
	return func(decoder *json.Decoder) *Error {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return NewError(CodeInvalidFrame)
		}
		return decodePayload(raw, map[string]fieldDecoder{
			"tab_id":              decodeInt64Field(&target.TabID, true),
			"projection_epoch":    decodeStringField(&target.ProjectionEpoch, MaxEpochBytes, true),
			"projection_revision": decodeUint64Field(&target.ProjectionRevision, true),
			"result_id":           decodeStringField(&target.ResultID, MaxOpaqueValueBytes, true),
		}, []string{"tab_id", "projection_epoch", "projection_revision", "result_id"})
	}
}

func decodeErrorCodeField(target *ErrorCode, optional bool) fieldDecoder {
	return func(decoder *json.Decoder) *Error {
		token, err := decoder.Token()
		if optional && (err != nil || token == nil) {
			return nil
		}
		if err != nil {
			return NewError(CodeInvalidFrame)
		}
		value, ok := token.(string)
		if !ok || len(value) > MaxMessageTypeBytes {
			return NewError(CodeInvalidFrame)
		}
		code := ErrorCode(value)
		if !code.KnownCode() {
			return NewError(CodeInvalidFrame)
		}
		*target = code
		return nil
	}
}

func decodeRawObjectField(target *json.RawMessage) fieldDecoder {
	return func(decoder *json.Decoder) *Error {
		raw, err := decodeObject(decoder)
		if err != nil {
			return NewError(CodeInvalidFrame)
		}
		*target = raw
		return nil
	}
}

// decodeRawArrayField captures an array payload and enforces its item bound by
// counting elements before any typed decode or allocation happens.
func decodeRawArrayField(target *json.RawMessage, maxItems int, boundName string) fieldDecoder {
	return func(decoder *json.Decoder) *Error {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return NewError(CodeInvalidFrame)
		}
		body := bytes.TrimSpace(raw)
		if len(body) == 0 || body[0] != '[' {
			return NewError(CodeInvalidFrame)
		}
		count, err := countArrayItems(body)
		if err != nil {
			return NewError(CodeInvalidFrame)
		}
		if maxItems > 0 && count > maxItems {
			return NewError(CodePayloadLimit).WithBound(boundName, maxItems, count)
		}
		*target = body
		return nil
	}
}

// decodeStrictArray decodes an already counted array with unknown-field
// rejection per element.
func decodeStrictArray[T any](raw json.RawMessage) ([]T, *Error) {
	body := bytes.TrimSpace(raw)
	if len(body) == 0 || body[0] != '[' {
		return nil, NewError(CodeInvalidFrame)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil || !isDelim(opening, '[') {
		return nil, NewError(CodeInvalidFrame)
	}
	var items []T
	for decoder.More() {
		var element json.RawMessage
		if err := decoder.Decode(&element); err != nil {
			return nil, NewError(CodeInvalidFrame)
		}
		var item T
		elementDecoder := json.NewDecoder(bytes.NewReader(element))
		elementDecoder.DisallowUnknownFields()
		elementDecoder.UseNumber()
		if err := elementDecoder.Decode(&item); err != nil {
			return nil, NewError(CodeInvalidFrame)
		}
		if _, err := elementDecoder.Token(); !errors.Is(err, io.EOF) {
			return nil, NewError(CodeInvalidFrame)
		}
		items = append(items, item)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, NewError(CodeInvalidFrame)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, NewError(CodeInvalidFrame)
	}
	return items, nil
}

func countArrayItems(body []byte) (int, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if _, err := decoder.Token(); err != nil {
		return 0, err
	}
	count := 0
	for decoder.More() {
		var discard json.RawMessage
		if err := decoder.Decode(&discard); err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}

func decodeIntField(target *int, required bool) fieldDecoder {
	return func(decoder *json.Decoder) *Error {
		token, err := decoder.Token()
		if err != nil {
			return NewError(CodeInvalidFrame)
		}
		if token == nil {
			if required {
				return NewError(CodeInvalidFrame)
			}
			return nil
		}
		number, ok := token.(json.Number)
		if !ok {
			return NewError(CodeInvalidFrame)
		}
		value, err := number.Int64()
		if err != nil {
			return NewError(CodeInvalidFrame)
		}
		*target = int(value)
		return nil
	}
}
