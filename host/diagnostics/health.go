// Package diagnostics owns the aggregate health model, bounded diagnostic
// events, redaction, repair orchestration, and retention policy for the local
// InfoBoard host. It consumes the runtime lifecycle snapshot and the IP-08
// storage seam; it does not define a second protocol transport or SQLite schema.
package diagnostics

import (
	"errors"
	"sync"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

var (
	// ErrIllegalTransition is returned when a caller tries to publish a state
	// change that the health model cannot legally reach.
	ErrIllegalTransition = errors.New("diagnostics: illegal health transition")
	// ErrInvalidSnapshot is returned when a snapshot contains a value outside
	// one of the closed health vocabularies.
	ErrInvalidSnapshot = errors.New("diagnostics: invalid health snapshot")
)

// State is the aggregate, user-facing health state. It is intentionally
// separate from the runtime lifecycle substate so diagnostics can distinguish
// an incompatible host from a recoverable transport failure.
type State string

const (
	StateDisconnected  State = "disconnected"
	StateConnecting    State = "connecting"
	StateHandshaking   State = "handshaking"
	StateSynchronizing State = "synchronizing"
	StateReady         State = "ready"
	StateDegraded      State = "degraded"
	StateRecovering    State = "recovering"
	StateIncompatible  State = "incompatible"
	StateStopping      State = "stopping"
)

var states = map[State]struct{}{
	StateDisconnected: {}, StateConnecting: {}, StateHandshaking: {},
	StateSynchronizing: {}, StateReady: {}, StateDegraded: {},
	StateRecovering: {}, StateIncompatible: {}, StateStopping: {},
}

// Valid reports whether the state belongs to the closed health vocabulary.
func (state State) Valid() bool {
	_, ok := states[state]
	return ok
}

// Terminal reports whether the state is final for the current host session.
func (state State) Terminal() bool { return state == StateStopping }

// LexicalAvailable reports whether the in-memory lexical path may serve work.
// Persistence degradation is visible in the state but does not disable search.
func (state State) LexicalAvailable() bool {
	return state == StateReady || state == StateDegraded
}

// HostState is the availability of the Native Messaging host process.
type HostState string

const (
	HostAvailable   HostState = "available"
	HostUnavailable HostState = "unavailable"
)

func (state HostState) Valid() bool { return state == HostAvailable || state == HostUnavailable }

// ProtocolState is the compatibility of the negotiated protocol.
type ProtocolState string

const (
	ProtocolCompatible ProtocolState = "compatible"
	ProtocolMismatch   ProtocolState = "mismatch"
	ProtocolUnknown    ProtocolState = "unknown"
)

func (state ProtocolState) Valid() bool {
	return state == ProtocolCompatible || state == ProtocolMismatch || state == ProtocolUnknown
}

// ProjectionState is the freshness of the accepted browser projection.
type ProjectionState string

const (
	ProjectionCurrent ProjectionState = "current"
	ProjectionUnknown ProjectionState = "unknown"
	ProjectionGapped  ProjectionState = "gapped"
)

func (state ProjectionState) Valid() bool {
	return state == ProjectionCurrent || state == ProjectionUnknown || state == ProjectionGapped
}

// IndexState is the readiness of the in-memory lexical index.
type IndexState string

const (
	IndexEmpty    IndexState = "empty"
	IndexBuilding IndexState = "building"
	IndexReady    IndexState = "ready"
	IndexStale    IndexState = "stale"
	IndexFailed   IndexState = "failed"
	IndexUnknown  IndexState = "unknown"
)

func (state IndexState) Valid() bool {
	switch state {
	case IndexEmpty, IndexBuilding, IndexReady, IndexStale, IndexFailed, IndexUnknown:
		return true
	default:
		return false
	}
}

// StorageState is the IP-08 persistence posture. A degraded or disabled store
// never disables lexical search by itself.
type StorageState string

const (
	StorageHealthy         StorageState = "healthy"
	StorageDegraded        StorageState = "degraded"
	StorageQuarantined     StorageState = "quarantined"
	StorageUpgradeRequired StorageState = "upgrade_required"
	StorageDisabled        StorageState = "disabled"
	StorageUnknown         StorageState = "unknown"
)

func (state StorageState) Valid() bool {
	switch state {
	case StorageHealthy, StorageDegraded, StorageQuarantined, StorageUpgradeRequired, StorageDisabled, StorageUnknown:
		return true
	default:
		return false
	}
}

// Snapshot is one atomic aggregate health observation. It contains only
// bounded enums, versions, revisions, counts, and timestamps; raw browser data
// has no representation here.
type Snapshot struct {
	State               State
	Host                HostState
	Protocol            ProtocolState
	Projection          ProjectionState
	Index               IndexState
	Storage             StorageState
	ProfileID           domain.ProfileID
	ContextKind         domain.ContextKind
	ProtocolVersion     int
	RankingModelVersion string
	SchemaVersion       uint64
	ProjectionRevision  uint64
	IndexedRevision     uint64
	ProjectionSequence  uint64
	HandshakeAt         time.Time
	UpdatedAt           time.Time
	Retryable           bool
	EventSequence       uint64
}

// Valid rejects values outside the closed health vocabularies.
func (snapshot Snapshot) Valid() bool {
	return snapshot.State.Valid() &&
		snapshot.Host.Valid() &&
		snapshot.Protocol.Valid() &&
		snapshot.Projection.Valid() &&
		snapshot.Index.Valid() &&
		snapshot.Storage.Valid()
}

// Ready reports whether the aggregate state is fully healthy. A degraded store
// is represented by StateDegraded and is intentionally not ready.
func (snapshot Snapshot) Ready() bool {
	return snapshot.State == StateReady &&
		snapshot.Host == HostAvailable &&
		snapshot.Protocol == ProtocolCompatible &&
		snapshot.Projection == ProjectionCurrent &&
		snapshot.Index == IndexReady &&
		snapshot.IndexedRevision == snapshot.ProjectionRevision &&
		snapshot.Storage == StorageHealthy
}

// LexicalAvailable reports whether the current aggregate state can serve the
// in-memory lexical path, including persistence-degraded operation.
func (snapshot Snapshot) LexicalAvailable() bool {
	return snapshot.State.LexicalAvailable() &&
		snapshot.Host == HostAvailable &&
		snapshot.Protocol == ProtocolCompatible &&
		snapshot.Projection == ProjectionCurrent
}

// Normalize applies the aggregate rules that must hold before publication:
// persistence-only degradation is surfaced as StateDegraded, and a ready
// snapshot whose index is not built from the current revision is not ready.
func (snapshot Snapshot) Normalize() Snapshot {
	if snapshot.State != StateStopping && snapshot.Protocol == ProtocolMismatch {
		snapshot.State = StateIncompatible
		return snapshot
	}
	if snapshot.State != StateStopping && snapshot.State != StateIncompatible && snapshot.Host == HostUnavailable {
		snapshot.State = StateDisconnected
		snapshot.Retryable = true
		return snapshot
	}
	if snapshot.State == StateReady && snapshot.Storage != StorageHealthy {
		snapshot.State = StateDegraded
	}
	if snapshot.State == StateDegraded && snapshot.Storage == StorageHealthy && snapshot.readyComponents() {
		snapshot.State = StateReady
	}
	if snapshot.State == StateReady && !snapshot.readyComponents() {
		snapshot.State = StateRecovering
		snapshot.Retryable = true
	}
	return snapshot
}

func (snapshot Snapshot) readyComponents() bool {
	return snapshot.Host == HostAvailable &&
		snapshot.Protocol == ProtocolCompatible &&
		snapshot.Projection == ProjectionCurrent &&
		snapshot.Index == IndexReady &&
		snapshot.IndexedRevision == snapshot.ProjectionRevision
}

// Observer receives every successfully published aggregate snapshot.
type Observer interface {
	Observe(Snapshot)
}

// ObserverFunc adapts a function to Observer.
type ObserverFunc func(Snapshot)

// Observe implements Observer.
func (observe ObserverFunc) Observe(snapshot Snapshot) { observe(snapshot) }

// Health owns the aggregate health state machine. It publishes a snapshot and
// its monotonic event sequence atomically so readers cannot observe a new state
// with old freshness metadata.
type Health struct {
	mu          sync.Mutex
	now         func() time.Time
	snapshot    Snapshot
	observers   []Observer
	transitions uint64
}

// NewHealth builds a health model from an initial snapshot. A nil clock uses
// time.Now.
func NewHealth(initial Snapshot, now func() time.Time) (*Health, error) {
	if now == nil {
		now = time.Now
	}
	initial = initial.Normalize()
	if !initial.Valid() {
		return nil, ErrInvalidSnapshot
	}
	if initial.UpdatedAt.IsZero() {
		initial.UpdatedAt = now()
	}
	return &Health{now: now, snapshot: initial}, nil
}

// Snapshot returns a copy of the current aggregate snapshot.
func (health *Health) Snapshot() Snapshot {
	if health == nil {
		return Snapshot{}
	}
	health.mu.Lock()
	defer health.mu.Unlock()
	return health.snapshot
}

// Subscribe registers an observer. Observers are invoked after the lock is
// released and must not block the caller.
func (health *Health) Subscribe(observer Observer) {
	if health == nil || observer == nil {
		return
	}
	health.mu.Lock()
	health.observers = append(health.observers, observer)
	health.mu.Unlock()
}

// Transition validates and publishes a new aggregate snapshot. An illegal or
// invalid transition leaves the public snapshot unchanged and returns an error
// so the caller can emit a bounded INTERNAL_FAILURE diagnostic.
func (health *Health) Transition(next Snapshot) error {
	if health == nil {
		return ErrIllegalTransition
	}
	next = next.Normalize()
	if !next.Valid() {
		return ErrInvalidSnapshot
	}
	health.mu.Lock()
	current := health.snapshot
	if !transitionAllowed(current.State, next.State) {
		health.mu.Unlock()
		return ErrIllegalTransition
	}
	health.transitions++
	next.EventSequence = current.EventSequence + 1
	next.UpdatedAt = health.now()
	health.snapshot = next
	published := next
	observers := append([]Observer(nil), health.observers...)
	health.mu.Unlock()
	for _, observer := range observers {
		observer.Observe(published)
	}
	return nil
}

// Transitions returns the number of accepted aggregate transitions.
func (health *Health) Transitions() uint64 {
	if health == nil {
		return 0
	}
	health.mu.Lock()
	defer health.mu.Unlock()
	return health.transitions
}

var legalTransitions = map[State]map[State]struct{}{
	StateDisconnected: {
		StateConnecting: {}, StateIncompatible: {}, StateStopping: {},
	},
	StateConnecting: {
		StateHandshaking: {}, StateRecovering: {}, StateDisconnected: {},
		StateIncompatible: {}, StateStopping: {},
	},
	StateHandshaking: {
		StateSynchronizing: {}, StateRecovering: {}, StateDisconnected: {},
		StateIncompatible: {}, StateStopping: {},
	},
	StateSynchronizing: {
		StateReady: {}, StateRecovering: {}, StateDisconnected: {},
		StateIncompatible: {}, StateStopping: {},
	},
	StateReady: {
		StateDegraded: {}, StateRecovering: {}, StateDisconnected: {},
		StateIncompatible: {}, StateStopping: {},
	},
	StateDegraded: {
		StateReady: {}, StateRecovering: {}, StateDisconnected: {},
		StateIncompatible: {}, StateStopping: {},
	},
	StateRecovering: {
		StateConnecting: {}, StateHandshaking: {}, StateSynchronizing: {},
		StateDisconnected: {}, StateIncompatible: {}, StateStopping: {},
	},
	StateIncompatible: {
		StateStopping: {},
	},
	StateStopping: {},
}

func transitionAllowed(from, to State) bool {
	if !from.Valid() || !to.Valid() {
		return false
	}
	if from == to {
		return true
	}
	_, ok := legalTransitions[from][to]
	return ok
}
