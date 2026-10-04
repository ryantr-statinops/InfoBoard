package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"unicode/utf8"
)

// Error is the typed, redacted protocol failure. Its text is the wire code
// literal only, so no frame content can leak through an error message.
type Error struct {
	Code             ErrorCode
	MessageKey       MessageKey
	Retryable        bool
	RequestID        string
	RetryAfterMillis *int64
	ExpectedRevision *uint64
	ExpectedSequence *uint64
	LimitName        string
	LimitBound       int
	Observed         int
}

func (failure *Error) Error() string { return string(failure.Code) }

// NewError builds a typed protocol failure for a known code.
func NewError(code ErrorCode) *Error {
	return &Error{
		Code:       code,
		MessageKey: code.MessageKey(),
		Retryable:  code.Retryable(),
	}
}

// WithRequestID echoes the originating request identity when it is safely
// parseable.
func (failure *Error) WithRequestID(requestID string) *Error {
	if nonEmptyOpaque(requestID, MaxRequestIDBytes) {
		failure.RequestID = requestID
	}
	return failure
}

// WithBound records the exceeded bound without any payload content.
func (failure *Error) WithBound(name string, bound, observed int) *Error {
	failure.LimitName = name
	failure.LimitBound = bound
	failure.Observed = observed
	return failure
}

// WithExpectation records the revision or sequence the host still requires.
func (failure *Error) WithExpectation(revision, sequence *uint64) *Error {
	failure.ExpectedRevision = revision
	failure.ExpectedSequence = sequence
	return failure
}

// WithRetryAfter records a bounded retry hint in milliseconds.
func (failure *Error) WithRetryAfter(millis int64) *Error {
	if millis >= 0 {
		failure.RetryAfterMillis = &millis
	}
	return failure
}

var errTrailingJSON = errors.New("protocol: trailing bytes after JSON object")

// DecodeEnvelope converts one already bounded frame payload into a strict
// envelope. It enforces the encoded payload limit before decoding, accepts
// exactly one JSON object, rejects trailing bytes, and rejects missing, null,
// wrong-typed, duplicated, and unknown fields.
func DecodeEnvelope(payload []byte, limits Limits) (Envelope, *Error) {
	if len(payload) == 0 {
		return Envelope{}, NewError(CodeInvalidFrame)
	}
	if limits.MaxPayloadBytes > 0 && len(payload) > limits.MaxPayloadBytes {
		return Envelope{}, NewError(CodePayloadLimit).WithBound("encoded_payload_bytes", limits.MaxPayloadBytes, len(payload))
	}
	if !utf8.Valid(payload) {
		return Envelope{}, NewError(CodeInvalidFrame)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil || !isDelim(opening, '{') {
		return Envelope{}, NewError(CodeInvalidFrame)
	}
	var (
		envelope Envelope
		seen     = make(map[string]struct{}, 6)
	)
	for decoder.More() {
		key, ok := tokenKey(decoder)
		if !ok {
			return Envelope{}, NewError(CodeInvalidFrame)
		}
		if _, duplicate := seen[key]; duplicate {
			return Envelope{}, NewError(CodeInvalidFrame)
		}
		seen[key] = struct{}{}
		if failure, handled := decodeEnvelopeField(decoder, key, &envelope, limits); handled {
			return Envelope{}, failure
		}
	}
	closing, err := decoder.Token()
	if err != nil || !isDelim(closing, '}') {
		return Envelope{}, NewError(CodeInvalidFrame)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Envelope{}, NewError(CodeInvalidFrame)
	}
	if len(seen) != 6 {
		return Envelope{}, NewError(CodeInvalidFrame)
	}
	return envelope, nil
}

func decodeEnvelopeField(decoder *json.Decoder, key string, envelope *Envelope, limits Limits) (*Error, bool) {
	switch key {
	case "protocol":
		version, err := decodeInteger(decoder)
		if err != nil {
			return NewError(CodeInvalidFrame), true
		}
		if version < 0 || version > int64(CurrentVersion)+1 {
			return NewError(CodeInvalidFrame), true
		}
		if version != CurrentVersion {
			return NewError(CodeProtocolMismatch), true
		}
		envelope.Protocol = int(version)
	case "type":
		value, err := decodeString(decoder)
		if err != nil {
			return NewError(CodeInvalidFrame), true
		}
		if len(value) == 0 || len(value) > MaxMessageTypeBytes {
			return NewError(CodePayloadLimit).WithBound("message_type_bytes", MaxMessageTypeBytes, len(value)), true
		}
		messageType := MessageType(value)
		if !messageType.KnownType() {
			return NewError(CodeInvalidFrame), true
		}
		envelope.Type = messageType
	case "request_id":
		value, err := decodeString(decoder)
		if err != nil {
			return NewError(CodeInvalidFrame), true
		}
		if !nonEmptyOpaque(value, limits.MaxRequestIDBytes) {
			return NewError(CodeInvalidFrame), true
		}
		envelope.RequestID = value
	case "profile_id":
		value, err := decodeString(decoder)
		if err != nil {
			return NewError(CodeInvalidFrame), true
		}
		if !nonEmptyOpaque(value, limits.MaxProfileIDBytes) {
			return NewError(CodeInvalidFrame), true
		}
		envelope.ProfileID = value
	case "projection_revision":
		revision, err := decodeUnsigned(decoder)
		if err != nil {
			return NewError(CodeInvalidFrame), true
		}
		envelope.ProjectionRevision = revision
	case "payload":
		raw, err := decodeObject(decoder)
		if err != nil {
			return NewError(CodeInvalidFrame), true
		}
		if limits.MaxPayloadBytes > 0 && len(raw) > limits.MaxPayloadBytes {
			return NewError(CodePayloadLimit).WithBound("payload_object_bytes", limits.MaxPayloadBytes, len(raw)), true
		}
		envelope.Payload = raw
	default:
		return NewError(CodeInvalidFrame), true
	}
	return nil, false
}

// EncodeEnvelope renders a canonical envelope. Field order is fixed so byte
// digests stay stable for idempotent replay comparison.
func EncodeEnvelope(envelope Envelope, limits Limits) ([]byte, *Error) {
	if envelope.Protocol != CurrentVersion {
		return nil, NewError(CodeProtocolMismatch)
	}
	if !envelope.Type.KnownType() || len(envelope.Type) > MaxMessageTypeBytes {
		return nil, NewError(CodeInvalidFrame)
	}
	if !nonEmptyOpaque(envelope.RequestID, limits.MaxRequestIDBytes) {
		return nil, NewError(CodeInvalidFrame)
	}
	if !nonEmptyOpaque(envelope.ProfileID, limits.MaxProfileIDBytes) {
		return nil, NewError(CodeInvalidFrame)
	}
	body := bytes.TrimSpace(envelope.Payload)
	if len(body) == 0 || body[0] != '{' {
		return nil, NewError(CodeInvalidFrame)
	}
	var buffer bytes.Buffer
	buffer.Grow(len(body) + 160)
	buffer.WriteString(`{"protocol":`)
	buffer.WriteString(strconv.Itoa(envelope.Protocol))
	buffer.WriteString(`,"type":`)
	writeJSONString(&buffer, string(envelope.Type))
	buffer.WriteString(`,"request_id":`)
	writeJSONString(&buffer, envelope.RequestID)
	buffer.WriteString(`,"profile_id":`)
	writeJSONString(&buffer, envelope.ProfileID)
	buffer.WriteString(`,"projection_revision":`)
	buffer.WriteString(strconv.FormatUint(envelope.ProjectionRevision, 10))
	buffer.WriteString(`,"payload":`)
	buffer.Write(body)
	buffer.WriteByte('}')
	if limits.MaxPayloadBytes > 0 && buffer.Len() > limits.MaxPayloadBytes {
		return nil, NewError(CodePayloadLimit).WithBound("encoded_payload_bytes", limits.MaxPayloadBytes, buffer.Len())
	}
	return buffer.Bytes(), nil
}

func writeJSONString(buffer *bytes.Buffer, value string) {
	encoded, err := json.Marshal(value)
	if err != nil {
		buffer.WriteString(`""`)
		return
	}
	buffer.Write(encoded)
}

// DecodePayloadObject decodes one strict payload object against an explicit
// field allowlist. Unknown, duplicated, missing, or null fields fail closed
// before the caller applies anything.
func DecodePayloadObject(payload []byte, fields map[string]fieldDecoder) *Error {
	return decodePayload(payload, fields, nil)
}

// decodePayload decodes a strict payload object. When required is non-nil only
// the listed fields must be present; the rest are optional but still bounded
// and still reject duplicates or unknown keys.
func decodePayload(payload []byte, fields map[string]fieldDecoder, required []string) *Error {
	body := bytes.TrimSpace(payload)
	if len(body) == 0 || body[0] != '{' {
		return NewError(CodeInvalidFrame)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil || !isDelim(opening, '{') {
		return NewError(CodeInvalidFrame)
	}
	seen := make(map[string]struct{}, len(fields))
	for decoder.More() {
		key, ok := tokenKey(decoder)
		if !ok {
			return NewError(CodeInvalidFrame)
		}
		if _, duplicate := seen[key]; duplicate {
			return NewError(CodeInvalidFrame)
		}
		seen[key] = struct{}{}
		decode, known := fields[key]
		if !known {
			return NewError(CodeInvalidFrame)
		}
		if failure := decode(decoder); failure != nil {
			return failure
		}
	}
	closing, err := decoder.Token()
	if err != nil || !isDelim(closing, '}') {
		return NewError(CodeInvalidFrame)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return NewError(CodeInvalidFrame)
	}
	if required == nil {
		if len(seen) != len(fields) {
			return NewError(CodeInvalidFrame)
		}
		return nil
	}
	for _, key := range required {
		if _, present := seen[key]; !present {
			return NewError(CodeInvalidFrame)
		}
	}
	return nil
}

// fieldDecoder reads exactly one typed value from a payload object.
type fieldDecoder func(decoder *json.Decoder) *Error

func decodeString(decoder *json.Decoder) (string, error) {
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	text, ok := value.(string)
	if !ok {
		return "", errors.New("protocol: value is not a string")
	}
	return text, nil
}

func decodeStringField(target *string, maxBytes int, optional bool) fieldDecoder {
	return func(decoder *json.Decoder) *Error {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return NewError(CodeInvalidFrame)
		}
		if isDelim(token, 'n') {
			if optional {
				return nil
			}
			return NewError(CodeInvalidFrame)
		}
		if token == nil {
			if optional {
				return nil
			}
			return NewError(CodeInvalidFrame)
		}
		value, ok := token.(string)
		if !ok {
			return NewError(CodeInvalidFrame)
		}
		if !optional && value == "" {
			return NewError(CodeInvalidFrame)
		}
		if maxBytes > 0 && len(value) > maxBytes {
			return NewError(CodePayloadLimit).WithBound("string_field_bytes", maxBytes, len(value))
		}
		*target = value
		return nil
	}
}

func decodeInteger(decoder *json.Decoder) (int64, error) {
	token, err := decoder.Token()
	if err != nil {
		return 0, err
	}
	number, ok := token.(json.Number)
	if !ok {
		return 0, errors.New("protocol: value is not a number")
	}
	value, err := number.Int64()
	if err != nil {
		return 0, err
	}
	return value, nil
}

func decodeUnsigned(decoder *json.Decoder) (uint64, error) {
	token, err := decoder.Token()
	if err != nil {
		return 0, err
	}
	number, ok := token.(json.Number)
	if !ok {
		return 0, errors.New("protocol: value is not a number")
	}
	value, err := strconv.ParseUint(number.String(), 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func decodeUint64Field(target *uint64, required bool) fieldDecoder {
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
		value, err := strconv.ParseUint(number.String(), 10, 64)
		if err != nil {
			return NewError(CodeInvalidFrame)
		}
		*target = value
		return nil
	}
}

func decodeInt64Field(target *int64, required bool) fieldDecoder {
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
		*target = value
		return nil
	}
}

func decodeBoolField(target *bool) fieldDecoder {
	return func(decoder *json.Decoder) *Error {
		token, err := decoder.Token()
		if err != nil {
			return NewError(CodeInvalidFrame)
		}
		value, ok := token.(bool)
		if !ok {
			return NewError(CodeInvalidFrame)
		}
		*target = value
		return nil
	}
}

func decodeUint64SliceField(target *[]uint64, maxItems int, required bool) fieldDecoder {
	return func(decoder *json.Decoder) *Error {
		values, failure := decodeNumberArray(decoder, maxItems, required)
		if failure != nil {
			return failure
		}
		*target = values
		return nil
	}
}

func decodeNumberArray(decoder *json.Decoder, maxItems int, required bool) ([]uint64, *Error) {
	opening, err := decoder.Token()
	if required && (err != nil || !isDelim(opening, '[')) {
		return nil, NewError(CodeInvalidFrame)
	}
	if err != nil {
		if optional := err == io.EOF; optional && !required {
			return nil, nil
		}
		return nil, NewError(CodeInvalidFrame)
	}
	if !isDelim(opening, '[') {
		return nil, NewError(CodeInvalidFrame)
	}
	var values []uint64
	for decoder.More() {
		if maxItems > 0 && len(values) >= maxItems {
			return nil, NewError(CodePayloadLimit).WithBound("array_items", maxItems, len(values)+1)
		}
		token, err := decoder.Token()
		if err != nil {
			return nil, NewError(CodeInvalidFrame)
		}
		number, ok := token.(json.Number)
		if !ok {
			return nil, NewError(CodeInvalidFrame)
		}
		value, err := strconv.ParseUint(number.String(), 10, 64)
		if err != nil {
			return nil, NewError(CodeInvalidFrame)
		}
		values = append(values, value)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, NewError(CodeInvalidFrame)
	}
	return values, nil
}

func decodeStringSliceField(target *[]string, maxItems, maxItemBytes int, required bool) fieldDecoder {
	return func(decoder *json.Decoder) *Error {
		opening, err := decoder.Token()
		if required && (err != nil || !isDelim(opening, '[')) {
			return NewError(CodeInvalidFrame)
		}
		if err != nil {
			if !required && errors.Is(err, io.EOF) {
				return nil
			}
			return NewError(CodeInvalidFrame)
		}
		if !isDelim(opening, '[') {
			return NewError(CodeInvalidFrame)
		}
		var values []string
		for decoder.More() {
			if len(values) >= maxItems {
				return NewError(CodePayloadLimit).WithBound("array_items", maxItems, len(values)+1)
			}
			token, err := decoder.Token()
			if err != nil {
				return NewError(CodeInvalidFrame)
			}
			value, ok := token.(string)
			if !ok {
				return NewError(CodeInvalidFrame)
			}
			if value == "" || len(value) > maxItemBytes {
				return NewError(CodePayloadLimit).WithBound("array_item_bytes", maxItemBytes, len(value))
			}
			values = append(values, value)
		}
		if _, err := decoder.Token(); err != nil {
			return NewError(CodeInvalidFrame)
		}
		*target = values
		return nil
	}
}

func decodeObject(decoder *json.Decoder) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, errors.New("protocol: payload is not an object")
	}
	return trimmed, nil
}

func tokenKey(decoder *json.Decoder) (string, bool) {
	token, err := decoder.Token()
	if err != nil {
		return "", false
	}
	key, ok := token.(string)
	if !ok {
		return "", false
	}
	return key, true
}

func isDelim(token json.Token, want json.Delim) bool {
	delim, ok := token.(json.Delim)
	return ok && delim == want
}
