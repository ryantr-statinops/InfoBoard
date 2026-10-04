package runtime

import (
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// Availability is the public runtime availability required by FR-012. It is
// derived from the lifecycle substate and never carries raw tab metadata.
type Availability string

const (
	AvailabilityHealthy     Availability = "healthy"
	AvailabilityUnavailable Availability = "unavailable"
	AvailabilityRecovering  Availability = "recovering"
)

// PersistenceStatus is the IP-08 supplied storage posture. A degraded store is
// reported but never makes the in-memory lexical runtime unavailable by itself.
type PersistenceStatus string

const (
	PersistenceUnknown  PersistenceStatus = "unknown"
	PersistenceReady    PersistenceStatus = "ready"
	PersistenceDegraded PersistenceStatus = "degraded"
	PersistenceDisabled PersistenceStatus = "disabled"
)

// FailureClass is the runtime-owned safe classification of a failure. The
// Native Messaging wire error literals are owned by host/internal/protocol
// (IP-07); the protocol handler maps these classes onto its own wire codes.
type FailureClass string

const (
	FailureNone                   FailureClass = ""
	FailureConfiguration          FailureClass = "configuration_invalid"
	FailureRegistration           FailureClass = "registration_invalid"
	FailureStartupTimeout         FailureClass = "startup_timeout"
	FailureHandshakeTimeout       FailureClass = "handshake_timeout"
	FailureHandshakeRejected      FailureClass = "handshake_rejected"
	FailureProtocolIncompatible   FailureClass = "protocol_incompatible"
	FailureProfileMismatch        FailureClass = "profile_mismatch"
	FailureSnapshotRejected       FailureClass = "snapshot_rejected"
	FailureSynchronizationTimeout FailureClass = "synchronization_timeout"
	FailureRequestTimeout         FailureClass = "request_timeout"
	FailureTransportEOF           FailureClass = "transport_eof"
	FailureTransportTruncated     FailureClass = "transport_truncated"
	FailureTransportMalformed     FailureClass = "transport_malformed"
	FailureTransportBrokenPipe    FailureClass = "transport_broken_pipe"
	FailureTransportUnavailable   FailureClass = "transport_unavailable"
	FailureFrameTooLarge          FailureClass = "frame_too_large"
	FailureQueueOverflow          FailureClass = "queue_overflow"
	FailureConcurrencyLimit       FailureClass = "concurrency_limit"
	FailureDependency             FailureClass = "dependency_unavailable"
	FailureShutdown               FailureClass = "shutdown"
	FailureCancelled              FailureClass = "cancelled"
	FailureInternal               FailureClass = "internal_failure"
)

// Retryable reports whether the extension may recover the session through a
// fresh browser-provided connection without reinstalling the product (NFR-006).
// It is the single canonical table: a rejected frame, an invalid configuration,
// and an incompatible protocol are contract violations that a new connection
// alone cannot repair.
func (class FailureClass) Retryable() bool {
	switch class {
	case FailureStartupTimeout, FailureHandshakeTimeout, FailureHandshakeRejected,
		FailureSnapshotRejected, FailureSynchronizationTimeout, FailureRequestTimeout,
		FailureTransportEOF, FailureTransportBrokenPipe, FailureTransportUnavailable,
		FailureQueueOverflow, FailureConcurrencyLimit,
		FailureDependency, FailureShutdown, FailureCancelled:
		return true
	default:
		return false
	}
}

// HealthSnapshot is the bounded, redacted runtime status consumed by the
// IP-07 protocol layer and by IP-16 diagnostics. It carries only state, safe
// error classes, retryability, counts, versions, timings, and projection
// revision/count when the projection owner supplies them. Raw titles, URLs,
// query strings, tokens, and page data are structurally absent.
type HealthSnapshot struct {
	SessionID          string
	State              LifecycleState
	Availability       Availability
	Failure            FailureClass
	Retryable          bool
	ProfileID          domain.ProfileID
	ContextKind        domain.ContextKind
	HostVersion        string
	ProtocolVersion    int
	Persistence        PersistenceStatus
	ProjectionKnown    bool
	ProjectionEpoch    string
	ProjectionRevision uint64
	ProjectionCount    uint64
	StateSinceMillis   int64
	UptimeMillis       int64
	Transitions        uint64
	FramesRead         uint64
	FramesWritten      uint64
	RequestsStarted    uint64
	RequestsCompleted  uint64
	RequestsRejected   uint64
	DroppedDiagnostics uint64
}

// AcceptsWork reports whether the session may accept application work. Only the
// Ready state qualifies, so a session with unknown profile, invalid version,
// failed limits, or a rebuilding projection can never serve queries.
func (snapshot HealthSnapshot) AcceptsWork() bool {
	return snapshot.State == StateReady && snapshot.Availability == AvailabilityHealthy
}

// HealthObserver receives every accepted lifecycle transition. Observers must
// not block: they are invoked synchronously from the transition that published
// the snapshot, and a session never waits on diagnostic consumers.
type HealthObserver interface {
	Observe(HealthSnapshot)
}

// HealthObserverFunc adapts a function to HealthObserver.
type HealthObserverFunc func(HealthSnapshot)

// Observe implements HealthObserver.
func (observe HealthObserverFunc) Observe(snapshot HealthSnapshot) { observe(snapshot) }

type healthTracker struct {
	mu          sync.Mutex
	clock       Clock
	startedAt   time.Time
	snapshot    HealthSnapshot
	observers   []HealthObserver
	transitions uint64
}

func newHealthTracker(clock Clock, sessionID, hostVersion string, protocolVersion int) *healthTracker {
	now := clock.Now()
	return &healthTracker{
		clock:     clock,
		startedAt: now,
		snapshot: HealthSnapshot{
			SessionID:        sessionID,
			State:            StateStarting,
			Availability:     availabilityOf(StateStarting),
			HostVersion:      hostVersion,
			ProtocolVersion:  protocolVersion,
			Persistence:      PersistenceUnknown,
			StateSinceMillis: now.UnixMilli(),
		},
	}
}

func (tracker *healthTracker) transition(state LifecycleState, failure FailureClass, retryable bool) {
	tracker.mu.Lock()
	if tracker.snapshot.State == state && tracker.snapshot.Failure == failure {
		tracker.refreshLocked()
		tracker.mu.Unlock()
		return
	}
	now := tracker.clock.Now()
	tracker.transitions++
	tracker.snapshot.State = state
	tracker.snapshot.Availability = availabilityOf(state)
	tracker.snapshot.Failure = failure
	tracker.snapshot.Retryable = retryable
	tracker.snapshot.StateSinceMillis = now.UnixMilli()
	tracker.snapshot.UptimeMillis = now.Sub(tracker.startedAt).Milliseconds()
	tracker.snapshot.Transitions = tracker.transitions
	published := tracker.snapshot
	observers := append([]HealthObserver(nil), tracker.observers...)
	tracker.mu.Unlock()
	for _, observer := range observers {
		observer.Observe(published)
	}
}

func (tracker *healthTracker) mutate(apply func(*HealthSnapshot)) {
	tracker.mu.Lock()
	apply(&tracker.snapshot)
	tracker.refreshLocked()
	published := tracker.snapshot
	observers := append([]HealthObserver(nil), tracker.observers...)
	tracker.mu.Unlock()
	for _, observer := range observers {
		observer.Observe(published)
	}
}

// count updates bounded counters without republishing observers, so hot-path
// bookkeeping cannot amplify observer traffic.
func (tracker *healthTracker) count(apply func(*HealthSnapshot)) {
	tracker.mu.Lock()
	apply(&tracker.snapshot)
	tracker.refreshLocked()
	tracker.mu.Unlock()
}

func (tracker *healthTracker) refreshLocked() {
	tracker.snapshot.UptimeMillis = tracker.clock.Now().Sub(tracker.startedAt).Milliseconds()
}

func (tracker *healthTracker) current() HealthSnapshot {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.refreshLocked()
	return tracker.snapshot
}

func (tracker *healthTracker) subscribe(observer HealthObserver) {
	if observer == nil {
		return
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.observers = append(tracker.observers, observer)
}

// DiagnosticLevel is the severity of a bounded diagnostic entry.
type DiagnosticLevel string

const (
	LevelDebug DiagnosticLevel = "debug"
	LevelInfo  DiagnosticLevel = "info"
	LevelWarn  DiagnosticLevel = "warn"
	LevelError DiagnosticLevel = "error"
)

// Event is the closed vocabulary of lifecycle diagnostics. Entries carrying an
// event outside this set are dropped, so no free-form or tab-derived text can
// reach the diagnostic sink.
type Event string

const (
	EventBootstrapAccepted   Event = "bootstrap.accepted"
	EventBootstrapRejected   Event = "bootstrap.rejected"
	EventLimitsRejected      Event = "limits.rejected"
	EventRegistrationInvalid Event = "registration.invalid"
	EventPersistenceStarted  Event = "persistence.started"
	EventPersistenceSettled  Event = "persistence.settled"
	EventPersistenceFlushed  Event = "persistence.flushed"
	EventHandshakeStarted    Event = "handshake.started"
	EventHandshakeFailed     Event = "handshake.failed"
	EventSynchronizeStarted  Event = "synchronize.started"
	EventSynchronizeFailed   Event = "synchronize.failed"
	EventStateChanged        Event = "state.changed"
	EventFrameRejected       Event = "frame.rejected"
	EventFrameReadFailed     Event = "frame.read_failed"
	EventFrameWriteFailed    Event = "frame.write_failed"
	EventRequestStarted      Event = "request.started"
	EventRequestRejected     Event = "request.rejected"
	EventRequestFailed       Event = "request.failed"
	EventShutdownStarted     Event = "shutdown.started"
	EventShutdownDrained     Event = "shutdown.drained"
	EventSessionFailed       Event = "session.failed"
	EventSessionClosed       Event = "session.closed"
)

var diagnosticEvents = map[Event]struct{}{
	EventBootstrapAccepted: {}, EventBootstrapRejected: {}, EventLimitsRejected: {},
	EventRegistrationInvalid: {}, EventPersistenceStarted: {}, EventPersistenceSettled: {},
	EventPersistenceFlushed: {}, EventHandshakeStarted: {}, EventHandshakeFailed: {},
	EventSynchronizeStarted: {}, EventSynchronizeFailed: {}, EventStateChanged: {},
	EventFrameRejected: {}, EventFrameReadFailed: {}, EventFrameWriteFailed: {},
	EventRequestStarted: {}, EventRequestRejected: {}, EventRequestFailed: {},
	EventShutdownStarted: {}, EventShutdownDrained: {}, EventSessionFailed: {},
	EventSessionClosed: {},
}

// Diagnostic is one bounded, redacted lifecycle record. Every field is either a
// closed-vocabulary constant or derived from a HealthSnapshot.
type Diagnostic struct {
	UptimeMillis    int64             `json:"uptime_ms"`
	SessionID       string            `json:"session_id"`
	Level           DiagnosticLevel   `json:"level"`
	Event           Event             `json:"event"`
	State           LifecycleState    `json:"state"`
	Availability    Availability      `json:"availability"`
	Failure         FailureClass      `json:"failure,omitempty"`
	Retryable       bool              `json:"retryable"`
	DurationMillis  int64             `json:"duration_ms,omitempty"`
	Count           int64             `json:"count,omitempty"`
	HostVersion     string            `json:"host_version,omitempty"`
	ProtocolVersion int               `json:"protocol_version,omitempty"`
	Persistence     PersistenceStatus `json:"persistence"`
	Transitions     uint64            `json:"transitions,omitempty"`
	Dropped         uint64            `json:"dropped,omitempty"`
	ProjectionCount uint64            `json:"projection_count,omitempty"`
}

// DiagnosticSink consumes bounded diagnostics. It is never the stdout protocol
// channel; the host routes diagnostics to stderr or a local file sink only.
type DiagnosticSink interface {
	Write(Diagnostic)
}

// DiagnosticSinkFunc adapts a function to DiagnosticSink.
type DiagnosticSinkFunc func(Diagnostic)

// Write implements DiagnosticSink.
func (write DiagnosticSinkFunc) Write(entry Diagnostic) { write(entry) }

// Diagnostics keeps a bounded in-memory tail and forwards a bounded number of
// bytes to a sink. Retention is a ring: once the configured entry budget is
// full the oldest entry is replaced, and total forwarded bytes are capped.
type Diagnostics struct {
	mu         sync.Mutex
	entries    []Diagnostic
	next       int
	dropped    uint64
	forwarded  int
	sink       DiagnosticSink
	byteBudget int
	maxEntries int
}

// NewDiagnostics builds a bounded diagnostics recorder. A nil sink discards
// entries after retention, which keeps the bounded tail inspectable in tests.
func NewDiagnostics(sink DiagnosticSink, maxEntries, byteBudget int) *Diagnostics {
	if maxEntries <= 0 {
		maxEntries = 1
	}
	if byteBudget <= 0 {
		byteBudget = 1
	}
	return &Diagnostics{entries: make([]Diagnostic, 0, maxEntries), sink: sink, maxEntries: maxEntries, byteBudget: byteBudget}
}

// Observe records one entry derived from a health snapshot. It returns false
// when the event or session identity is outside the redacted allowlist.
func (diags *Diagnostics) Observe(snapshot HealthSnapshot, level DiagnosticLevel, event Event, durationMillis, count int64) bool {
	entry, accepted := diags.record(snapshot, level, event, durationMillis, count)
	if !accepted {
		return false
	}
	diags.forward(entry)
	return true
}

// record retains the bounded entry and reports whether the sink may forward it.
// The sink is invoked outside the lock so a slow consumer cannot stall the
// session or the lifecycle transitions it observes.
func (diags *Diagnostics) record(snapshot HealthSnapshot, level DiagnosticLevel, event Event, durationMillis, count int64) (Diagnostic, bool) {
	diags.mu.Lock()
	defer diags.mu.Unlock()
	if _, allowed := diagnosticEvents[event]; !allowed || !validSessionID(snapshot.SessionID) {
		diags.dropped++
		return Diagnostic{}, false
	}
	entry := Diagnostic{
		UptimeMillis:    snapshot.UptimeMillis,
		SessionID:       snapshot.SessionID,
		Level:           level,
		Event:           event,
		State:           snapshot.State,
		Availability:    snapshot.Availability,
		Failure:         snapshot.Failure,
		Retryable:       snapshot.Retryable,
		DurationMillis:  durationMillis,
		Count:           count,
		HostVersion:     snapshot.HostVersion,
		ProtocolVersion: snapshot.ProtocolVersion,
		Persistence:     snapshot.Persistence,
		Transitions:     snapshot.Transitions,
		ProjectionCount: snapshot.ProjectionCount,
	}
	if len(diags.entries) < diags.maxEntries {
		diags.entries = append(diags.entries, entry)
	} else {
		diags.entries[diags.next] = entry
		diags.next = (diags.next + 1) % diags.maxEntries
	}
	return entry, diags.sink != nil && diags.forwarded < diags.byteBudget
}

func (diags *Diagnostics) forward(entry Diagnostic) {
	diags.mu.Lock()
	if diags.sink == nil || diags.forwarded >= diags.byteBudget {
		diags.dropped++
		diags.mu.Unlock()
		return
	}
	sink := diags.sink
	diags.forwarded += len(entry.SessionID) + len(entry.Event) + len(entry.Failure) + len(entry.HostVersion) + len(entry.Persistence)
	diags.mu.Unlock()
	sink.Write(entry)
}

// Entries returns the retained diagnostic tail in chronological order.
func (diags *Diagnostics) Entries() []Diagnostic {
	diags.mu.Lock()
	defer diags.mu.Unlock()
	if len(diags.entries) < diags.maxEntries {
		return append([]Diagnostic(nil), diags.entries...)
	}
	ordered := make([]Diagnostic, 0, len(diags.entries))
	ordered = append(ordered, diags.entries[diags.next:]...)
	ordered = append(ordered, diags.entries[:diags.next]...)
	return ordered
}

// Dropped returns the number of redacted or budget-suppressed diagnostic entries.
func (diags *Diagnostics) Dropped() uint64 {
	diags.mu.Lock()
	defer diags.mu.Unlock()
	return diags.dropped
}

type streamSink struct {
	mu     sync.Mutex
	out    io.Writer
	budget int
	used   int
}

// NewBoundedSink returns a DiagnosticSink that writes one compact JSON line per
// entry to out and forwards at most budget bytes in total. It is used for stderr
// or a local diagnostics file only, never for the protocol stream.
func NewBoundedSink(out io.Writer, budget int) DiagnosticSink {
	sink := newStreamSink(out, budget)
	if sink == nil {
		return nil
	}
	return sink
}

func newStreamSink(out io.Writer, budget int) *streamSink {
	if out == nil || budget <= 0 {
		return nil
	}
	return &streamSink{out: out, budget: budget}
}

func (sink *streamSink) Write(entry Diagnostic) {
	line, err := json.Marshal(entry)
	if err != nil {
		return
	}
	line = append(line, '\n')
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if sink.used >= sink.budget {
		return
	}
	if remaining := sink.budget - sink.used; len(line) > remaining {
		line = line[:remaining]
	}
	sink.used += len(line)
	_, _ = sink.out.Write(line)
}

func validSessionID(value string) bool {
	if len(value) != 32 {
		return false
	}
	return !strings.ContainsFunc(value, func(r rune) bool {
		return !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f')
	})
}
