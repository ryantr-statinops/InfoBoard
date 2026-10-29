package diagnostics

import (
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

// Freshness is an explicit, bounded freshness marker. A missing field is never
// treated as current; callers must choose one of these states.
type Freshness string

const (
	FreshnessCurrent    Freshness = "current"
	FreshnessStale      Freshness = "stale"
	FreshnessUnknown    Freshness = "unknown"
	FreshnessRebuilding Freshness = "rebuilding"
)

func (freshness Freshness) Valid() bool {
	switch freshness {
	case FreshnessCurrent, FreshnessStale, FreshnessUnknown, FreshnessRebuilding:
		return true
	default:
		return false
	}
}

// Counters are the bounded runtime counters exposed through the health payload.
// They contain no request, browser, or page data.
type Counters struct {
	FramesRead         uint64
	FramesWritten      uint64
	RequestsStarted    uint64
	RequestsCompleted  uint64
	RequestsRejected   uint64
	Transitions        uint64
	ProtocolErrors     uint64
	DiagnosticsDropped uint64
}

// HealthPayload is the host-side health projection consumed by diagnostics
// exporters and, for compatible fields, by the existing IP-07 health_result.
// Extended fields remain host-side in this phase; full extension wiring belongs
// to the packaging and release phases.
type HealthPayload struct {
	HostVersion         string
	ProtocolVersion     int
	RankingModelVersion string
	SchemaVersion       uint64
	ProfileID           domain.ProfileID
	ContextKind         domain.ContextKind
	State               State
	Host                HostState
	Protocol            ProtocolState
	Projection          ProjectionState
	Index               IndexState
	Storage             StorageState
	ProjectionFreshness Freshness
	IndexFreshness      Freshness
	ProjectionRevision  uint64
	IndexedRevision     uint64
	ProjectionSequence  uint64
	LastHandshakeAt     time.Time
	UpdatedAt           time.Time
	DiagnosticAgeMillis int64
	Retryable           bool
	Counters            Counters
}

// Valid rejects a payload that contains an out-of-vocabulary state or a
// negative duration.
func (payload HealthPayload) Valid() bool {
	return payload.State.Valid() &&
		payload.Host.Valid() &&
		payload.Protocol.Valid() &&
		payload.Projection.Valid() &&
		payload.Index.Valid() &&
		payload.Storage.Valid() &&
		payload.ProjectionFreshness.Valid() &&
		payload.IndexFreshness.Valid() &&
		payload.DiagnosticAgeMillis >= 0
}

// Payload projects one aggregate snapshot into the bounded health payload.
// The projection is pure and deterministic for a fixed now value.
func (snapshot Snapshot) Payload(now time.Time) HealthPayload {
	snapshot = snapshot.Normalize()
	if now.IsZero() {
		now = snapshot.UpdatedAt
	}
	age := int64(0)
	if !snapshot.UpdatedAt.IsZero() && now.After(snapshot.UpdatedAt) {
		age = now.Sub(snapshot.UpdatedAt).Milliseconds()
	}
	return HealthPayload{
		HostVersion:         "",
		ProtocolVersion:     snapshot.ProtocolVersion,
		RankingModelVersion: snapshot.RankingModelVersion,
		SchemaVersion:       snapshot.SchemaVersion,
		ProfileID:           snapshot.ProfileID,
		ContextKind:         snapshot.ContextKind,
		State:               snapshot.State,
		Host:                snapshot.Host,
		Protocol:            snapshot.Protocol,
		Projection:          snapshot.Projection,
		Index:               snapshot.Index,
		Storage:             snapshot.Storage,
		ProjectionFreshness: snapshot.projectionFreshness(),
		IndexFreshness:      snapshot.indexFreshness(),
		ProjectionRevision:  snapshot.ProjectionRevision,
		IndexedRevision:     snapshot.IndexedRevision,
		ProjectionSequence:  snapshot.ProjectionSequence,
		LastHandshakeAt:     snapshot.HandshakeAt,
		UpdatedAt:           snapshot.UpdatedAt,
		DiagnosticAgeMillis: age,
		Retryable:           snapshot.Retryable,
	}
}

func (snapshot Snapshot) projectionFreshness() Freshness {
	switch snapshot.Projection {
	case ProjectionCurrent:
		if snapshot.State == StateSynchronizing || snapshot.Index == IndexBuilding {
			return FreshnessRebuilding
		}
		return FreshnessCurrent
	case ProjectionGapped:
		return FreshnessStale
	default:
		return FreshnessUnknown
	}
}

func (snapshot Snapshot) indexFreshness() Freshness {
	switch snapshot.Index {
	case IndexReady:
		if snapshot.IndexedRevision == snapshot.ProjectionRevision {
			return FreshnessCurrent
		}
		return FreshnessStale
	case IndexBuilding:
		return FreshnessRebuilding
	case IndexStale, IndexFailed:
		return FreshnessStale
	default:
		return FreshnessUnknown
	}
}

// Wire projects the compatible subset onto the frozen IP-07 health_result
// shape. It deliberately does not add protocol fields in this phase.
func (payload HealthPayload) Wire() protocol.HealthResult {
	return protocol.HealthResult{
		Availability:       wireAvailability(payload.State),
		HostState:          string(payload.State),
		ProtocolVersion:    payload.ProtocolVersion,
		SessionID:          "",
		StorageState:       wireStorage(payload.Storage),
		IndexState:         string(payload.Index),
		ProjectionRevision: payload.ProjectionRevision,
		ProjectionSequence: payload.ProjectionSequence,
		FreshnessAgeMillis: payload.DiagnosticAgeMillis,
		Retryable:          payload.Retryable,
		Counters: protocol.HealthCounters{
			FramesRead:        payload.Counters.FramesRead,
			FramesWritten:     payload.Counters.FramesWritten,
			RequestsStarted:   payload.Counters.RequestsStarted,
			RequestsCompleted: payload.Counters.RequestsCompleted,
			RequestsRejected:  payload.Counters.RequestsRejected,
			Transitions:       payload.Counters.Transitions,
			ProtocolErrors:    payload.Counters.ProtocolErrors,
		},
	}
}

func wireAvailability(state State) string {
	switch state {
	case StateReady:
		return "healthy"
	case StateDisconnected, StateIncompatible, StateStopping:
		return "unavailable"
	default:
		return "recovering"
	}
}

func wireStorage(state StorageState) string {
	switch state {
	case StorageHealthy:
		return "ready"
	case StorageDegraded, StorageQuarantined, StorageUpgradeRequired:
		return "degraded"
	case StorageDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}
