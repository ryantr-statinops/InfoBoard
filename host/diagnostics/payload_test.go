package diagnostics

import (
	"testing"
	"time"
)

func TestPayloadReady(t *testing.T) {
	snapshot := readySnapshot()
	snapshot.UpdatedAt = time.Unix(100, 0).UTC()
	snapshot.HandshakeAt = time.Unix(90, 0).UTC()
	payload := snapshot.Payload(time.Unix(102, 500_000_000).UTC())
	if !payload.Valid() {
		t.Fatalf("payload invalid: %+v", payload)
	}
	if payload.ProjectionFreshness != FreshnessCurrent || payload.IndexFreshness != FreshnessCurrent {
		t.Fatalf("freshness = %s/%s", payload.ProjectionFreshness, payload.IndexFreshness)
	}
	if payload.DiagnosticAgeMillis != 2500 {
		t.Fatalf("diagnostic age = %d", payload.DiagnosticAgeMillis)
	}
	if payload.LastHandshakeAt != snapshot.HandshakeAt {
		t.Fatalf("handshake = %s", payload.LastHandshakeAt)
	}
	wire := payload.Wire()
	if wire.Availability != "healthy" || wire.StorageState != "ready" || wire.IndexState != "ready" {
		t.Fatalf("wire = %+v", wire)
	}
	if wire.ProjectionRevision != 7 || wire.ProjectionSequence != 3 || wire.FreshnessAgeMillis != 2500 {
		t.Fatalf("wire freshness = %+v", wire)
	}
}

func TestPayloadFreshnessStates(t *testing.T) {
	snapshot := readySnapshot()
	snapshot.Projection = ProjectionGapped
	if got := snapshot.Payload(time.Time{}).ProjectionFreshness; got != FreshnessStale {
		t.Fatalf("gapped projection freshness = %s", got)
	}
	snapshot = readySnapshot()
	snapshot.Index = IndexBuilding
	snapshot.IndexedRevision = 6
	payload := snapshot.Payload(time.Time{})
	if payload.IndexFreshness != FreshnessRebuilding || payload.ProjectionFreshness != FreshnessRebuilding {
		t.Fatalf("rebuilding freshness = %s/%s", payload.IndexFreshness, payload.ProjectionFreshness)
	}
	snapshot = readySnapshot()
	snapshot.IndexedRevision = 6
	if got := snapshot.Payload(time.Time{}).IndexFreshness; got != FreshnessStale {
		t.Fatalf("stale index freshness = %s", got)
	}
	snapshot = readySnapshot()
	snapshot.Index = IndexUnknown
	if got := snapshot.Payload(time.Time{}).IndexFreshness; got != FreshnessUnknown {
		t.Fatalf("unknown index freshness = %s", got)
	}
}

func TestPayloadWireDegradedAndIncompatible(t *testing.T) {
	snapshot := readySnapshot()
	snapshot.Storage = StorageDegraded
	payload := snapshot.Payload(time.Time{})
	wire := payload.Wire()
	if wire.Availability != "recovering" || wire.StorageState != "degraded" {
		t.Fatalf("degraded wire = %+v", wire)
	}
	if !payload.Valid() {
		t.Fatalf("degraded payload invalid: %+v", payload)
	}

	snapshot = readySnapshot()
	snapshot.Protocol = ProtocolMismatch
	payload = snapshot.Payload(time.Time{})
	wire = payload.Wire()
	if wire.Availability != "unavailable" {
		t.Fatalf("incompatible wire = %+v", wire)
	}
}

func TestPayloadRejectsNegativeAge(t *testing.T) {
	payload := readySnapshot().Payload(time.Time{})
	payload.DiagnosticAgeMillis = -1
	if payload.Valid() {
		t.Fatal("negative diagnostic age accepted")
	}
	payload = readySnapshot().Payload(time.Time{})
	payload.ProjectionFreshness = Freshness("invented")
	if payload.Valid() {
		t.Fatal("invalid freshness accepted")
	}
}
