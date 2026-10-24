// Package query owns the bounded query orchestration boundary between the
// Native Messaging protocol, the immutable lexical index, and IP-11 ranking.
package query

import (
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/index"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

// Source is the immutable published index seam. Query never reads a staged or
// older generation.
type Source interface {
	State() *index.State
	Ready() (uint64, bool)
	Rebuilding() bool
}

// Defaults supplies the profile-scoped result limit from IP-08. A missing or
// invalid value falls back to the configured safe default.
type Defaults interface {
	ResultLimit(profileID string) (int, bool)
}

// Recency supplies activation recency when durable metadata is available. A
// nil lookup or missing tab uses the session-only zero value.
type Recency interface {
	ActivatedAt(profileID string, tabID int64) (int64, bool)
}

// Health supplies the non-blocking persistence posture. Query results remain
// available when this reports degraded.
type Health interface {
	PersistenceState() string
}

// Observation contains bounded query counters only. Raw query text, titles,
// URLs, and identities are never included in diagnostics.
type Observation struct {
	Status           protocol.QueryStatus
	Candidates       int
	Returned         int
	TotalMicros      int64
	PersistenceState string
}

// Observer receives bounded query observations.
type Observer interface {
	ObserveQuery(Observation)
}

// Config is the immutable query configuration supplied by the host wiring.
type Config struct {
	// FallbackResultLimit is the IP-08 safe default used when the profile
	// configuration is unavailable.
	FallbackResultLimit int
	// MaxResultLimit is the configured maximum, capped by the negotiated
	// protocol maximum.
	MaxResultLimit int
	// RankingModelVersion stamps successful results.
	RankingModelVersion string
	// Now is injectable for deterministic tests.
	Now func() time.Time
}
