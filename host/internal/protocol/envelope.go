// Package protocol owns the Native Messaging wire contract between the InfoBoard
// Manifest V3 extension and the Go host.
//
// Boundary: host/internal/runtime remains the only stdin reader and stdout
// writer and owns the 4-byte little-endian length prefix, the frame size cap,
// session lifecycle, cancellation, and health. This package never touches a
// stream. It consumes one already bounded payload, enforces the encoded JSON
// payload limit before decoding, and owns the envelope, message schemas,
// ordering, request correlation, and exact wire error literals.
//
// Fail-closed rules: unknown message types, unsupported protocol versions,
// duplicate or unknown envelope fields, trailing bytes, and every exceeded bound
// are rejected before any projection, index, or durable state is touched.
package protocol

import (
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"
)

// CurrentVersion is the only supported major protocol version.
const CurrentVersion = 1

// Envelope and payload field byte bounds. The frame cap itself belongs to
// host/internal/runtime; this package mirrors it only to reject before decode.
const (
	MaxPayloadBytes     = 256 << 10
	MaxFrameBytes       = 1 << 20
	MaxRequestIDBytes   = 128
	MaxProfileIDBytes   = 128
	MaxMessageTypeBytes = 64
	MaxQueryScalars     = 512
	MaxSnapshotRecords  = 10000
	MaxDeltaOperations  = 1000
	MaxResults          = 50
	MaxTitleScalars     = 512
	MaxDomainBytes      = 255
	MaxURLBytes         = 2048
	MaxCapabilities     = 32
	MaxOpaqueValueBytes = 128
	MaxEpochBytes       = 64
	MaxExplanationCodes = 8
)

// Limits are the bounded protocol budgets reported to the extension in
// hello_ack. They are validated before any decode, allocation, or index use.
type Limits struct {
	MaxPayloadBytes    int `json:"max_payload_bytes"`
	MaxQueryScalars    int `json:"max_query_scalars"`
	MaxSnapshotRecords int `json:"max_snapshot_records"`
	MaxDeltaOperations int `json:"max_delta_operations"`
	MaxResults         int `json:"max_results"`
	MaxRequestIDBytes  int `json:"max_request_id_bytes"`
	MaxProfileIDBytes  int `json:"max_profile_id_bytes"`
	MaxTitleScalars    int `json:"max_title_scalars"`
	MaxDomainBytes     int `json:"max_domain_bytes"`
	MaxURLBytes        int `json:"max_url_bytes"`
	MaxCapabilities    int `json:"max_capabilities"`
}

// DefaultLimits returns the bounded production limits declared by the phase
// contract. They are the single source of truth for both host and client.
func DefaultLimits() Limits {
	return Limits{
		MaxPayloadBytes:    MaxPayloadBytes,
		MaxQueryScalars:    MaxQueryScalars,
		MaxSnapshotRecords: MaxSnapshotRecords,
		MaxDeltaOperations: MaxDeltaOperations,
		MaxResults:         MaxResults,
		MaxRequestIDBytes:  MaxRequestIDBytes,
		MaxProfileIDBytes:  MaxProfileIDBytes,
		MaxTitleScalars:    MaxTitleScalars,
		MaxDomainBytes:     MaxDomainBytes,
		MaxURLBytes:        MaxURLBytes,
		MaxCapabilities:    MaxCapabilities,
	}
}

// Validate rejects an unusable limit set before a session accepts any frame.
func (limits Limits) Validate() error {
	switch {
	case limits.MaxPayloadBytes <= 0, limits.MaxPayloadBytes > MaxFrameBytes,
		limits.MaxQueryScalars <= 0, limits.MaxQueryScalars > MaxQueryScalars,
		limits.MaxSnapshotRecords <= 0, limits.MaxSnapshotRecords > MaxSnapshotRecords,
		limits.MaxDeltaOperations <= 0, limits.MaxDeltaOperations > MaxDeltaOperations,
		limits.MaxResults <= 0, limits.MaxResults > MaxResults,
		limits.MaxRequestIDBytes <= 0, limits.MaxRequestIDBytes > MaxRequestIDBytes,
		limits.MaxProfileIDBytes <= 0, limits.MaxProfileIDBytes > MaxProfileIDBytes,
		limits.MaxTitleScalars <= 0, limits.MaxTitleScalars > MaxTitleScalars,
		limits.MaxDomainBytes <= 0, limits.MaxDomainBytes > MaxDomainBytes,
		limits.MaxURLBytes <= 0, limits.MaxURLBytes > MaxURLBytes,
		limits.MaxCapabilities <= 0, limits.MaxCapabilities > MaxCapabilities:
		return ErrInvalidLimits
	}
	return nil
}

// Budgets are the bounded per-operation deadlines. They live in one typed
// config instead of being scattered as magic numbers.
type Budgets struct {
	FrameRead          time.Duration
	Handshake          time.Duration
	SyncApply          time.Duration
	QueryHostBudget    time.Duration
	QueryRequestBudget time.Duration
	Health             time.Duration
	ActivationStatus   time.Duration
	ShutdownDrain      time.Duration
}

// DefaultBudgets returns the declared operational deadlines.
func DefaultBudgets() Budgets {
	return Budgets{
		FrameRead:          2 * time.Second,
		Handshake:          2 * time.Second,
		SyncApply:          time.Second,
		QueryHostBudget:    40 * time.Millisecond,
		QueryRequestBudget: 100 * time.Millisecond,
		Health:             100 * time.Millisecond,
		ActivationStatus:   500 * time.Millisecond,
		ShutdownDrain:      time.Second,
	}
}

// Validate rejects an unusable budget set.
func (budgets Budgets) Validate() error {
	for _, budget := range []time.Duration{
		budgets.FrameRead, budgets.Handshake, budgets.SyncApply,
		budgets.QueryHostBudget, budgets.QueryRequestBudget,
		budgets.Health, budgets.ActivationStatus, budgets.ShutdownDrain,
	} {
		if budget <= 0 || budget > time.Minute {
			return ErrInvalidBudgets
		}
	}
	if budgets.QueryHostBudget > budgets.QueryRequestBudget {
		return ErrInvalidBudgets
	}
	return nil
}

// ErrorCode is the exact wire error literal set. No other code may reach the
// extension.
type ErrorCode string

const (
	CodeInvalidFrame        ErrorCode = "INVALID_FRAME"
	CodeProtocolMismatch    ErrorCode = "PROTOCOL_MISMATCH"
	CodeProfileMismatch     ErrorCode = "PROFILE_MISMATCH"
	CodePayloadLimit        ErrorCode = "PAYLOAD_LIMIT"
	CodeSnapshotRequired    ErrorCode = "SNAPSHOT_REQUIRED"
	CodeRevisionMismatch    ErrorCode = "REVISION_MISMATCH"
	CodeIndexRebuilding     ErrorCode = "INDEX_REBUILDING"
	CodeQueryTimeout        ErrorCode = "QUERY_TIMEOUT"
	CodePersistenceDegraded ErrorCode = "PERSISTENCE_DEGRADED"
	CodeHostShutdown        ErrorCode = "HOST_SHUTDOWN"
	CodeInternalFailure     ErrorCode = "INTERNAL_FAILURE"
)

// MessageKey is the bounded, redacted user-facing key paired with a code. It
// never carries raw titles, URLs, tokens, or page data.
type MessageKey string

const (
	MessageInvalidFrame        MessageKey = "invalid_frame"
	MessageProtocolMismatch    MessageKey = "protocol_mismatch"
	MessageProfileMismatch     MessageKey = "profile_mismatch"
	MessagePayloadLimit        MessageKey = "payload_limit"
	MessageSnapshotRequired    MessageKey = "snapshot_required"
	MessageRevisionMismatch    MessageKey = "revision_mismatch"
	MessageIndexRebuilding     MessageKey = "index_rebuilding"
	MessageQueryTimeout        MessageKey = "query_timeout"
	MessagePersistenceDegraded MessageKey = "persistence_degraded"
	MessageHostShutdown        MessageKey = "host_shutdown"
	MessageInternalFailure     MessageKey = "internal_failure"
	MessageResyncRequired      MessageKey = "resync_required"
	MessageHelloAccepted       MessageKey = "hello_accepted"
	MessageSyncAccepted        MessageKey = "sync_accepted"
	MessageQueryAccepted       MessageKey = "query_accepted"
	MessageActivationRecorded  MessageKey = "activation_recorded"
	MessageHealthReported      MessageKey = "health_reported"
	MessageIncompatibleRepair  MessageKey = "update_extension_required"
)

var errorCodes = map[ErrorCode]MessageKey{
	CodeInvalidFrame:        MessageInvalidFrame,
	CodeProtocolMismatch:    MessageProtocolMismatch,
	CodeProfileMismatch:     MessageProfileMismatch,
	CodePayloadLimit:        MessagePayloadLimit,
	CodeSnapshotRequired:    MessageSnapshotRequired,
	CodeRevisionMismatch:    MessageRevisionMismatch,
	CodeIndexRebuilding:     MessageIndexRebuilding,
	CodeQueryTimeout:        MessageQueryTimeout,
	CodePersistenceDegraded: MessagePersistenceDegraded,
	CodeHostShutdown:        MessageHostShutdown,
	CodeInternalFailure:     MessageInternalFailure,
}

// KnownCode reports whether a literal belongs to the exact error enum.
func (code ErrorCode) KnownCode() bool {
	_, known := errorCodes[code]
	return known
}

// MessageKey returns the bounded key paired with a known code.
func (code ErrorCode) MessageKey() MessageKey {
	if key, known := errorCodes[code]; known {
		return key
	}
	return MessageInternalFailure
}

// Retryable reports whether the extension may retry after resynchronizing or
// reconnecting without reinstalling the product.
func (code ErrorCode) Retryable() bool {
	switch code {
	case CodeSnapshotRequired, CodeRevisionMismatch, CodeIndexRebuilding,
		CodeQueryTimeout, CodePersistenceDegraded, CodeHostShutdown:
		return true
	default:
		return false
	}
}

// MessageType is the closed envelope type enum.
type MessageType string

const (
	TypeHello              MessageType = "hello"
	TypeHelloAck           MessageType = "hello_ack"
	TypeSnapshot           MessageType = "snapshot"
	TypeDelta              MessageType = "delta"
	TypeSyncAck            MessageType = "sync_ack"
	TypeResyncRequired     MessageType = "resync_required"
	TypeQuery              MessageType = "query"
	TypeQueryResult        MessageType = "query_result"
	TypeActivationObserved MessageType = "activation_observed"
	TypeActivationFailed   MessageType = "activation_failed"
	TypeActivationAck      MessageType = "activation_ack"
	TypeHealth             MessageType = "health"
	TypeHealthResult       MessageType = "health_result"
	TypeError              MessageType = "error"
)

// Direction states which side may send a message type.
type Direction string

const (
	DirectionToHost      Direction = "extension_to_host"
	DirectionToExtension Direction = "host_to_extension"
)

var messageTypes = map[MessageType]Direction{
	TypeHello:              DirectionToHost,
	TypeHelloAck:           DirectionToExtension,
	TypeSnapshot:           DirectionToHost,
	TypeDelta:              DirectionToHost,
	TypeSyncAck:            DirectionToExtension,
	TypeResyncRequired:     DirectionToExtension,
	TypeQuery:              DirectionToHost,
	TypeQueryResult:        DirectionToExtension,
	TypeActivationObserved: DirectionToHost,
	TypeActivationFailed:   DirectionToHost,
	TypeActivationAck:      DirectionToExtension,
	TypeHealth:             DirectionToHost,
	TypeHealthResult:       DirectionToExtension,
	TypeError:              DirectionToExtension,
}

// KnownType reports whether a value belongs to the message type enum.
func (messageType MessageType) KnownType() bool {
	_, known := messageTypes[messageType]
	return known
}

// Direction returns the permitted sender direction.
func (messageType MessageType) Direction() Direction {
	if direction, known := messageTypes[messageType]; known {
		return direction
	}
	return ""
}

// ResyncReason is the safe enum explaining why the host needs a full snapshot.
type ResyncReason string

const (
	ResyncRevisionMismatch ResyncReason = "revision_mismatch"
	ResyncSequenceGap      ResyncReason = "sequence_gap"
	ResyncSnapshotStale    ResyncReason = "snapshot_stale"
	ResyncIndexRebuilding  ResyncReason = "index_rebuilding"
	ResyncProfileChanged   ResyncReason = "profile_changed"
)

var resyncReasons = map[ResyncReason]struct{}{
	ResyncRevisionMismatch: {}, ResyncSequenceGap: {}, ResyncSnapshotStale: {},
	ResyncIndexRebuilding: {}, ResyncProfileChanged: {},
}

// KnownReason reports whether a resync reason is part of the safe enum.
func (reason ResyncReason) KnownReason() bool {
	_, known := resyncReasons[reason]
	return known
}

// Envelope is the strict application message wrapper. Exactly these six fields
// are present on the wire; envelope.ProfileID is the sole profile identity and
// Payload stays opaque until the codec hands it to the typed message schema.
type Envelope struct {
	Protocol           int             `json:"protocol"`
	Type               MessageType     `json:"type"`
	RequestID          string          `json:"request_id"`
	ProfileID          string          `json:"profile_id"`
	ProjectionRevision uint64          `json:"projection_revision"`
	Payload            json.RawMessage `json:"payload"`
}

// OperationKind is the ordered delta operation enum.
type OperationKind string

const (
	OperationCreate   OperationKind = "create"
	OperationUpdate   OperationKind = "update"
	OperationMove     OperationKind = "move"
	OperationGroup    OperationKind = "group"
	OperationPin      OperationKind = "pin"
	OperationActivate OperationKind = "activate"
	OperationRemove   OperationKind = "remove"
)

var operationKinds = map[OperationKind]struct{}{
	OperationCreate: {}, OperationUpdate: {}, OperationMove: {}, OperationGroup: {},
	OperationPin: {}, OperationActivate: {}, OperationRemove: {},
}

// KnownOperation reports whether a delta operation kind is part of the enum.
func (kind OperationKind) KnownOperation() bool {
	_, known := operationKinds[kind]
	return known
}

var (
	// ErrInvalidLimits, ErrInvalidBudgets and ErrQueryBudget are internal
	// configuration and deadline causes. They are never wire codes.
	ErrInvalidLimits  = errors.New("protocol: invalid limits")
	ErrInvalidBudgets = errors.New("protocol: invalid budgets")
	ErrQueryBudget    = errors.New("protocol: query host budget exceeded")
)

// boundedScalars reports whether a value fits a Unicode scalar bound.
func boundedScalars(value string, max int) bool {
	return len(value) > 0 && utf8.RuneCountInString(value) <= max
}

// boundedBytes reports whether a value fits a byte bound. An empty value is
// allowed only when max is non-negative and the caller permits emptiness.
func withinBytes(value string, max int) bool {
	return len(value) <= max
}

// nonEmptyOpaque validates an opaque identifier: bounded, printable-free of
// control characters, and never empty.
func nonEmptyOpaque(value string, max int) bool {
	if value == "" || len(value) > max {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] == 0x7f {
			return false
		}
	}
	return true
}
