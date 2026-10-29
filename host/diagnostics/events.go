package diagnostics

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// EventSchemaVersion is the version of the structured diagnostic event
// contract. It changes only when a field is added, removed, or redefined.
const EventSchemaVersion = 1

var (
	ErrInvalidEvent = errors.New("diagnostics: invalid event")
	ErrInvalidID    = errors.New("diagnostics: invalid opaque identifier")
)

// Component identifies the owner of a diagnostic event.
type Component string

const (
	ComponentHost        Component = "host"
	ComponentProtocol    Component = "protocol"
	ComponentProjection  Component = "projection"
	ComponentIndex       Component = "index"
	ComponentStorage     Component = "storage"
	ComponentQuery       Component = "query"
	ComponentRepair      Component = "repair"
	ComponentDiagnostics Component = "diagnostics"
	ComponentRetention   Component = "retention"
)

func (component Component) Valid() bool {
	switch component {
	case ComponentHost, ComponentProtocol, ComponentProjection, ComponentIndex,
		ComponentStorage, ComponentQuery, ComponentRepair, ComponentDiagnostics,
		ComponentRetention:
		return true
	default:
		return false
	}
}

// ReasonCode is the bounded, user-safe explanation for an event.
type ReasonCode string

const (
	ReasonStateTransition     ReasonCode = "state_transition"
	ReasonHostStart           ReasonCode = "host_start"
	ReasonHostExit            ReasonCode = "host_exit"
	ReasonHandshakeSuccess    ReasonCode = "handshake_success"
	ReasonHandshakeFailure    ReasonCode = "handshake_failure"
	ReasonProtocolRejection   ReasonCode = "protocol_rejection"
	ReasonSnapshotAccepted    ReasonCode = "snapshot_accepted"
	ReasonDeltaGap            ReasonCode = "delta_gap"
	ReasonIndexBuildStart     ReasonCode = "index_build_start"
	ReasonIndexBuildFinish    ReasonCode = "index_build_finish"
	ReasonIndexBuildFailure   ReasonCode = "index_build_failure"
	ReasonQueryTimeout        ReasonCode = "query_timeout"
	ReasonStorageMigration    ReasonCode = "storage_migration"
	ReasonStorageDegradation  ReasonCode = "storage_degradation"
	ReasonStorageRecovery     ReasonCode = "storage_recovery"
	ReasonRepairStart         ReasonCode = "repair_start"
	ReasonRepairFinish        ReasonCode = "repair_finish"
	ReasonRepairExhausted     ReasonCode = "repair_exhausted"
	ReasonReset               ReasonCode = "reset"
	ReasonRetentionEviction   ReasonCode = "retention_eviction"
	ReasonPersistenceDegraded ReasonCode = "diagnostics_persistence_degraded"
	ReasonInternalFailure     ReasonCode = "internal_failure"
)

func (reason ReasonCode) Valid() bool {
	switch reason {
	case ReasonStateTransition, ReasonHostStart, ReasonHostExit,
		ReasonHandshakeSuccess, ReasonHandshakeFailure, ReasonProtocolRejection,
		ReasonSnapshotAccepted, ReasonDeltaGap, ReasonIndexBuildStart,
		ReasonIndexBuildFinish, ReasonIndexBuildFailure, ReasonQueryTimeout,
		ReasonStorageMigration, ReasonStorageDegradation, ReasonStorageRecovery,
		ReasonRepairStart, ReasonRepairFinish, ReasonRepairExhausted, ReasonReset,
		ReasonRetentionEviction, ReasonPersistenceDegraded, ReasonInternalFailure:
		return true
	default:
		return false
	}
}

// ErrorClass is the bounded failure vocabulary. The first block mirrors the
// IP-07 protocol literals exactly; the second block covers local lifecycle and
// persistence conditions that have no wire literal.
type ErrorClass string

const (
	ErrorNone                  ErrorClass = ""
	ErrorInvalidFrame          ErrorClass = "INVALID_FRAME"
	ErrorProtocolMismatch      ErrorClass = "PROTOCOL_MISMATCH"
	ErrorProfileMismatch       ErrorClass = "PROFILE_MISMATCH"
	ErrorPayloadLimit          ErrorClass = "PAYLOAD_LIMIT"
	ErrorSnapshotRequired      ErrorClass = "SNAPSHOT_REQUIRED"
	ErrorRevisionMismatch      ErrorClass = "REVISION_MISMATCH"
	ErrorIndexRebuilding       ErrorClass = "INDEX_REBUILDING"
	ErrorQueryTimeout          ErrorClass = "QUERY_TIMEOUT"
	ErrorPersistenceDegraded   ErrorClass = "PERSISTENCE_DEGRADED"
	ErrorHostShutdown          ErrorClass = "HOST_SHUTDOWN"
	ErrorInternalFailure       ErrorClass = "INTERNAL_FAILURE"
	ErrorHostMissing           ErrorClass = "HOST_MISSING"
	ErrorPermissionLimited     ErrorClass = "PERMISSION_LIMITED"
	ErrorStorageQuarantined    ErrorClass = "STORAGE_QUARANTINED"
	ErrorSchemaUpgradeRequired ErrorClass = "SCHEMA_UPGRADE_REQUIRED"
)

func (class ErrorClass) Valid() bool {
	switch class {
	case ErrorNone, ErrorInvalidFrame, ErrorProtocolMismatch, ErrorProfileMismatch,
		ErrorPayloadLimit, ErrorSnapshotRequired, ErrorRevisionMismatch,
		ErrorIndexRebuilding, ErrorQueryTimeout, ErrorPersistenceDegraded,
		ErrorHostShutdown, ErrorInternalFailure, ErrorHostMissing,
		ErrorPermissionLimited, ErrorStorageQuarantined, ErrorSchemaUpgradeRequired:
		return true
	default:
		return false
	}
}

// RecommendedAction is a bounded user-safe next step.
type RecommendedAction string

const (
	ActionNone              RecommendedAction = "none"
	ActionRetry             RecommendedAction = "retry"
	ActionReconnect         RecommendedAction = "reconnect"
	ActionRefreshProfile    RecommendedAction = "refresh_profile"
	ActionRefreshProjection RecommendedAction = "refresh_projection"
	ActionRebuildIndex      RecommendedAction = "rebuild_index"
	ActionRepairPersistence RecommendedAction = "repair_persistence"
	ActionUpdateHost        RecommendedAction = "update_host"
	ActionShortenQuery      RecommendedAction = "shorten_query"
	ActionWait              RecommendedAction = "wait"
	ActionOpenDiagnostics   RecommendedAction = "open_diagnostics"
	ActionRestartHost       RecommendedAction = "restart_host"
)

func (action RecommendedAction) Valid() bool {
	switch action {
	case ActionNone, ActionRetry, ActionReconnect, ActionRefreshProfile,
		ActionRefreshProjection, ActionRebuildIndex, ActionRepairPersistence,
		ActionUpdateHost, ActionShortenQuery, ActionWait, ActionOpenDiagnostics,
		ActionRestartHost:
		return true
	default:
		return false
	}
}

// LatencyBucket is a bounded timing class; it avoids exporting raw timing
// distributions or request values.
type LatencyBucket string

const (
	LatencyUnknown LatencyBucket = "unknown"
	LatencyFast    LatencyBucket = "fast"
	LatencyMedium  LatencyBucket = "medium"
	LatencySlow    LatencyBucket = "slow"
)

func (bucket LatencyBucket) Valid() bool {
	return bucket == LatencyUnknown || bucket == LatencyFast || bucket == LatencyMedium || bucket == LatencySlow
}

// EventCounts are bounded counts, never identifiers or browser values.
type EventCounts struct {
	RecordCount  uint64
	DroppedCount uint64
	RetryCount   uint64
	FailureCount uint64
}

// EventTiming contains bounded durations and a latency bucket.
type EventTiming struct {
	DurationMillis int64         `json:"duration_ms"`
	QueueMillis    int64         `json:"queue_ms"`
	Latency        LatencyBucket `json:"latency_bucket"`
}

// Event is the versioned structured diagnostic record. It contains only
// allowlisted enums, versions, revisions, counts, and timestamps. Raw browser
// data has no representation in this type.
type Event struct {
	SchemaVersion       int               `json:"schema_version"`
	EventID             string            `json:"event_id"`
	EventTime           time.Time         `json:"event_time"`
	EventSequence       uint64            `json:"event_sequence,omitempty"`
	SessionID           string            `json:"session_id,omitempty"`
	Component           Component         `json:"component"`
	FromState           State             `json:"from_state,omitempty"`
	ToState             State             `json:"to_state,omitempty"`
	ReasonCode          ReasonCode        `json:"reason_code"`
	ErrorClass          ErrorClass        `json:"error_class,omitempty"`
	Retryable           bool              `json:"retryable"`
	RecommendedAction   RecommendedAction `json:"recommended_action"`
	ProtocolVersion     int               `json:"protocol_version"`
	RankingModelVersion string            `json:"ranking_model_version,omitempty"`
	StoreSchemaVersion  uint64            `json:"store_schema_version,omitempty"`
	ProjectionRevision  uint64            `json:"projection_revision,omitempty"`
	IndexedRevision     uint64            `json:"indexed_revision,omitempty"`
	ProjectionSequence  uint64            `json:"projection_sequence,omitempty"`
	Counts              EventCounts       `json:"counts"`
	Timing              EventTiming       `json:"timing"`
}

// Validate rejects an event that could escape the bounded diagnostic
// vocabulary. Empty from/to states are allowed for non-transition events.
func (event Event) Validate() error {
	if event.SchemaVersion != EventSchemaVersion {
		return fmt.Errorf("%w: schema version", ErrInvalidEvent)
	}
	if !validOpaqueID(event.EventID, 96) || !validOpaqueID(event.SessionID, 64) {
		return fmt.Errorf("%w: identifier", ErrInvalidEvent)
	}
	if event.EventTime.IsZero() || event.EventTime.Location() != time.UTC {
		return fmt.Errorf("%w: event time", ErrInvalidEvent)
	}
	if !event.Component.Valid() || !event.ReasonCode.Valid() ||
		!event.ErrorClass.Valid() || !event.RecommendedAction.Valid() {
		return fmt.Errorf("%w: vocabulary", ErrInvalidEvent)
	}
	if event.FromState != "" && !event.FromState.Valid() {
		return fmt.Errorf("%w: from state", ErrInvalidEvent)
	}
	if event.ToState != "" && !event.ToState.Valid() {
		return fmt.Errorf("%w: to state", ErrInvalidEvent)
	}
	if event.ProtocolVersion < 0 || event.Timing.DurationMillis < 0 || event.Timing.QueueMillis < 0 {
		return fmt.Errorf("%w: negative timing", ErrInvalidEvent)
	}
	if !event.Timing.Latency.Valid() {
		return fmt.Errorf("%w: latency bucket", ErrInvalidEvent)
	}
	if !validBoundedText(event.RankingModelVersion, 64) {
		return fmt.Errorf("%w: ranking version", ErrInvalidEvent)
	}
	return nil
}

// NewSessionID returns a non-reversible, random opaque session correlation ID.
// It is not derived from a browser session or profile value.
func NewSessionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "sess-" + hex.EncodeToString(raw[:]), nil
}

// Recorder is a bounded in-memory event ring plus an optional persistence sink.
// A sink failure marks persistence degraded but never drops the event from the
// current-session ring.
type Recorder struct {
	mu                sync.Mutex
	sessionID         string
	now               func() time.Time
	capacity          int
	events            []Event
	sequence          uint64
	dropped           uint64
	persistenceFailed uint64
	lastErr           error
	sink              func(Event) error
	lastSnapshot      Snapshot
}

// NewRecorder creates a recorder with a bounded ring capacity. A capacity
// outside 1..1024 is clamped to the default 64.
func NewRecorder(sessionID string, capacity int, now func() time.Time) (*Recorder, error) {
	if !validOpaqueID(sessionID, 64) {
		return nil, ErrInvalidID
	}
	if capacity < 1 || capacity > 1024 {
		capacity = 64
	}
	if now == nil {
		now = time.Now
	}
	return &Recorder{sessionID: sessionID, now: now, capacity: capacity}, nil
}

// SetSink installs a bounded persistence sink. The sink must not call back into
// the recorder.
func (recorder *Recorder) SetSink(sink func(Event) error) {
	if recorder == nil {
		return
	}
	recorder.mu.Lock()
	recorder.sink = sink
	recorder.mu.Unlock()
}

// Record validates and appends an event, filling schema version, event ID, time,
// and session correlation. The event is retained even when the sink fails.
func (recorder *Recorder) Record(event Event) (Event, error) {
	if recorder == nil {
		return Event{}, ErrInvalidEvent
	}
	recorder.mu.Lock()
	recorder.sequence++
	event.SchemaVersion = EventSchemaVersion
	event.SessionID = recorder.sessionID
	event.EventID = fmt.Sprintf("evt-%s-%d", recorder.sessionID, recorder.sequence)
	if event.EventTime.IsZero() {
		event.EventTime = recorder.now().UTC()
	}
	if event.Timing.Latency == "" {
		event.Timing.Latency = LatencyUnknown
	}
	if err := event.Validate(); err != nil {
		recorder.sequence--
		recorder.mu.Unlock()
		return Event{}, err
	}
	if len(recorder.events) >= recorder.capacity {
		copy(recorder.events, recorder.events[1:])
		recorder.events = recorder.events[:len(recorder.events)-1]
		recorder.dropped++
	}
	recorder.events = append(recorder.events, event)
	sink := recorder.sink
	recorder.mu.Unlock()

	if sink != nil {
		if err := sink(event); err != nil {
			recorder.mu.Lock()
			recorder.persistenceFailed++
			recorder.lastErr = err
			recorder.mu.Unlock()
		}
	}
	return event, nil
}

// RecordTransition records exactly one transition event. The caller supplies
// the reason, error class, and action so the record remains actionable.
func (recorder *Recorder) RecordTransition(before, after Snapshot, reason ReasonCode, class ErrorClass, action RecommendedAction) (Event, error) {
	if reason == "" {
		reason = ReasonStateTransition
	}
	if action == "" {
		action = ActionForState(after.State)
	}
	return recorder.Record(Event{
		Component:           ComponentHost,
		EventSequence:       after.EventSequence,
		FromState:           before.State,
		ToState:             after.State,
		ReasonCode:          reason,
		ErrorClass:          class,
		Retryable:           after.Retryable,
		RecommendedAction:   action,
		ProtocolVersion:     after.ProtocolVersion,
		RankingModelVersion: after.RankingModelVersion,
		StoreSchemaVersion:  after.SchemaVersion,
		ProjectionRevision:  after.ProjectionRevision,
		IndexedRevision:     after.IndexedRevision,
		ProjectionSequence:  after.ProjectionSequence,
		Counts: EventCounts{
			RecordCount:  after.ProjectionSequence,
			DroppedCount: recorder.Dropped(),
		},
	})
}

// Attach wires the recorder to a health model. The transition hook runs before
// the new snapshot becomes visible and emits exactly one event for every
// accepted transition.
func (recorder *Recorder) Attach(health *Health) error {
	if recorder == nil || health == nil {
		return ErrInvalidEvent
	}
	recorder.mu.Lock()
	recorder.lastSnapshot = health.Snapshot()
	recorder.mu.Unlock()
	health.SetTransitionHook(func(next Snapshot) {
		recorder.mu.Lock()
		before := recorder.lastSnapshot
		recorder.lastSnapshot = next
		recorder.mu.Unlock()
		_, _ = recorder.RecordTransition(before, next, ReasonStateTransition, ErrorNone, ActionForState(next.State))
	})
	return nil
}

// Events returns a copy of the bounded in-memory ring.
func (recorder *Recorder) Events() []Event {
	if recorder == nil {
		return nil
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return append([]Event(nil), recorder.events...)
}

// Dropped reports how many oldest events were evicted by the ring bound.
func (recorder *Recorder) Dropped() uint64 {
	if recorder == nil {
		return 0
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return recorder.dropped
}

// PersistenceFailed reports how many sink writes failed.
func (recorder *Recorder) PersistenceFailed() uint64 {
	if recorder == nil {
		return 0
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return recorder.persistenceFailed
}

// PersistenceDegraded reports whether the optional sink has failed.
func (recorder *Recorder) PersistenceDegraded() bool {
	return recorder.PersistenceFailed() > 0
}

// LastError returns the last sink error without exposing it in an event.
func (recorder *Recorder) LastError() error {
	if recorder == nil {
		return nil
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return recorder.lastErr
}

// ActionForState maps aggregate state to a bounded next action.
func ActionForState(state State) RecommendedAction {
	switch state {
	case StateReady:
		return ActionNone
	case StateDegraded:
		return ActionRepairPersistence
	case StateIncompatible:
		return ActionUpdateHost
	case StateStopping:
		return ActionNone
	case StateDisconnected:
		return ActionReconnect
	case StateRecovering:
		return ActionRetry
	default:
		return ActionWait
	}
}

func validOpaqueID(value string, limit int) bool {
	if value == "" || len(value) > limit {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' || char == ':' {
			continue
		}
		return false
	}
	return true
}

func validBoundedText(value string, limit int) bool {
	if len(value) > limit {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}
