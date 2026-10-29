package diagnostics

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

const (
	// DiagnosticRetention is the fixed age bound. Exactly seven days old is
	// still inside the window; only strictly older records are evicted.
	DiagnosticRetention = 7 * 24 * time.Hour
	// MaxDiagnosticPayloadBytes is the product logical budget for serialized
	// diagnostic payload bytes. It is intentionally stricter than the physical
	// SQLite page overhead budget owned by IP-08.
	MaxDiagnosticPayloadBytes int64 = 10_000_000
	// MaxDiagnosticRecordBytes bounds one serialized record before insertion.
	MaxDiagnosticRecordBytes = 64 * 1024
	// DefaultDiagnosticRingCapacity bounds the in-memory fallback ring.
	DefaultDiagnosticRingCapacity = 128
)

// RetentionStore is the IP-08 repository seam. The diagnostics package owns the
// logical policy; storage owns the table, transaction, and physical compaction.
type RetentionStore interface {
	AppendDiagnostic(context.Context, storage.DiagnosticRecord) storage.Result
	PruneRetention(context.Context, time.Time) storage.Result
}

// RetentionOptions are bounded policy overrides for tests and future
// configuration. Zero values use the phase-16 defaults.
type RetentionOptions struct {
	Now            func() time.Time
	MaxAge         time.Duration
	MaxBytes       int64
	MaxRecordBytes int
	RingCapacity   int
}

// RetentionCounters are bounded and themselves subject to redaction.
type RetentionCounters struct {
	Inserted          uint64 `json:"inserted"`
	Rejected          uint64 `json:"rejected"`
	AgeEvicted        uint64 `json:"age_evicted"`
	ByteEvicted       uint64 `json:"byte_evicted"`
	PersistenceFailed uint64 `json:"persistence_failed"`
	Dropped           uint64 `json:"dropped"`
	RingRecords       uint64 `json:"ring_records"`
	RingBytes         int64  `json:"ring_bytes"`
}

// RetentionStatus is the observable outcome of one append/prune operation.
type RetentionStatus string

const (
	RetentionInserted RetentionStatus = "inserted"
	RetentionRejected RetentionStatus = "rejected"
	RetentionDegraded RetentionStatus = "degraded"
	RetentionPruned   RetentionStatus = "pruned"
)

// RetentionResult is the bounded outcome of an append or prune.
type RetentionResult struct {
	Status     RetentionStatus   `json:"status"`
	ErrorClass ErrorClass        `json:"error_class,omitempty"`
	Retryable  bool              `json:"retryable"`
	Counters   RetentionCounters `json:"counters"`
}

type retentionEntry struct {
	event      Event
	payload    []byte
	occurredAt time.Time
}

// RetentionManager enforces age, byte, per-record, and in-memory ring bounds
// before and after the IP-08 append/prune seam.
type RetentionManager struct {
	mu       sync.Mutex
	store    RetentionStore
	recorder *Recorder
	options  RetentionOptions
	ring     []retentionEntry
	counters RetentionCounters
	degraded bool
}

// NewRetentionManager creates a manager. A nil store keeps the bounded ring
// only and reports persistence degraded until a store is available.
func NewRetentionManager(store RetentionStore, recorder *Recorder, options RetentionOptions) *RetentionManager {
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.MaxAge <= 0 {
		options.MaxAge = DiagnosticRetention
	}
	if options.MaxBytes <= 0 {
		options.MaxBytes = MaxDiagnosticPayloadBytes
	}
	if options.MaxRecordBytes <= 0 {
		options.MaxRecordBytes = MaxDiagnosticRecordBytes
	}
	if options.RingCapacity <= 0 || options.RingCapacity > 4096 {
		options.RingCapacity = DefaultDiagnosticRingCapacity
	}
	return &RetentionManager{store: store, recorder: recorder, options: options}
}

// Append validates, redacts, bounds, and persists one event. An oversized,
// malformed, or expired record is counted and dropped without source content.
func (manager *RetentionManager) Append(ctx context.Context, event Event) RetentionResult {
	if manager == nil {
		return RetentionResult{Status: RetentionRejected, ErrorClass: ErrorInternalFailure, Retryable: false}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	manager.mu.Lock()
	now := manager.options.Now().UTC()
	manager.mu.Unlock()

	if event.EventTime.IsZero() {
		event.EventTime = now
	}
	event.EventTime = event.EventTime.UTC()
	if event.EventTime.After(now) {
		event.EventTime = now
	}
	if now.Sub(event.EventTime) > manager.options.MaxAge {
		manager.mu.Lock()
		manager.counters.AgeEvicted++
		result := manager.resultLocked(RetentionRejected, ErrorPersistenceDegraded, false)
		manager.mu.Unlock()
		return result
	}
	if err := event.Validate(); err != nil {
		manager.mu.Lock()
		manager.counters.Rejected++
		result := manager.resultLocked(RetentionRejected, ErrorInvalidFrame, false)
		manager.mu.Unlock()
		return result
	}
	payload, err := json.Marshal(Redaction{}.RedactEvent(event))
	if err != nil || !utf8.Valid(payload) || len(payload) > manager.options.MaxRecordBytes {
		manager.mu.Lock()
		manager.counters.Rejected++
		result := manager.resultLocked(RetentionRejected, ErrorPayloadLimit, false)
		manager.mu.Unlock()
		return result
	}

	entry := retentionEntry{event: event, payload: payload, occurredAt: event.EventTime}
	if manager.store == nil {
		manager.mu.Lock()
		manager.counters.PersistenceFailed++
		manager.degraded = true
		manager.addRingLocked(entry, now)
		result := manager.resultLocked(RetentionDegraded, ErrorPersistenceDegraded, true)
		manager.mu.Unlock()
		return result
	}

	record := toStorageRecord(event, payload)
	storageResult := manager.store.AppendDiagnostic(ctx, record)
	if storageResult.Status != storage.StatusReady {
		manager.mu.Lock()
		manager.counters.PersistenceFailed++
		manager.degraded = true
		manager.addRingLocked(entry, now)
		out := manager.resultLocked(RetentionDegraded, ErrorPersistenceDegraded, true)
		manager.mu.Unlock()
		return out
	}

	manager.mu.Lock()
	manager.counters.Inserted++
	manager.mu.Unlock()
	if prune := manager.store.PruneRetention(ctx, now); prune.Status != storage.StatusReady {
		manager.mu.Lock()
		manager.counters.PersistenceFailed++
		manager.degraded = true
		manager.mu.Unlock()
	}
	manager.mu.Lock()
	manager.pruneRingLocked(now)
	result := manager.resultLocked(RetentionInserted, ErrorNone, false)
	manager.mu.Unlock()
	return result
}

// Prune runs startup/insert-time retention and flushes only pending ring
// records that still satisfy the age and byte policy.
func (manager *RetentionManager) Prune(ctx context.Context) RetentionResult {
	if manager == nil {
		return RetentionResult{Status: RetentionRejected, ErrorClass: ErrorInternalFailure, Retryable: false}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	manager.mu.Lock()
	now := manager.options.Now().UTC()
	store := manager.store
	manager.mu.Unlock()
	if store != nil {
		result := store.PruneRetention(ctx, now)
		if result.Status != storage.StatusReady {
			manager.mu.Lock()
			manager.counters.PersistenceFailed++
			manager.degraded = true
			out := manager.resultLocked(RetentionDegraded, ErrorPersistenceDegraded, true)
			manager.mu.Unlock()
			return out
		}
		manager.flushRing(ctx, now)
	}
	manager.mu.Lock()
	manager.pruneRingLocked(now)
	out := manager.resultLocked(RetentionPruned, ErrorNone, false)
	manager.mu.Unlock()
	return out
}

// Counters returns a bounded copy of the current counters.
func (manager *RetentionManager) Counters() RetentionCounters {
	if manager == nil {
		return RetentionCounters{}
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.counters.RingRecords = uint64(len(manager.ring))
	manager.counters.RingBytes = manager.ringBytesLocked()
	return manager.counters
}

// PersistenceDegraded reports whether any append/prune failed.
func (manager *RetentionManager) PersistenceDegraded() bool {
	if manager == nil {
		return false
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.degraded
}

// RingEvents returns a copy of the pending in-memory ring.
func (manager *RetentionManager) RingEvents() []Event {
	if manager == nil {
		return nil
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	events := make([]Event, 0, len(manager.ring))
	for _, entry := range manager.ring {
		events = append(events, entry.event)
	}
	return events
}

func (manager *RetentionManager) flushRing(ctx context.Context, now time.Time) {
	manager.mu.Lock()
	pending := append([]retentionEntry(nil), manager.ring...)
	manager.mu.Unlock()
	kept := make([]retentionEntry, 0, len(pending))
	for _, entry := range pending {
		if now.Sub(entry.occurredAt) > manager.options.MaxAge {
			manager.mu.Lock()
			manager.counters.AgeEvicted++
			manager.mu.Unlock()
			continue
		}
		result := manager.store.AppendDiagnostic(ctx, toStorageRecord(entry.event, entry.payload))
		if result.Status != storage.StatusReady {
			manager.mu.Lock()
			manager.counters.PersistenceFailed++
			manager.degraded = true
			manager.mu.Unlock()
			kept = append(kept, entry)
			continue
		}
		manager.mu.Lock()
		manager.counters.Inserted++
		manager.mu.Unlock()
	}
	manager.mu.Lock()
	manager.ring = kept
	manager.pruneRingLocked(now)
	manager.mu.Unlock()
}

func (manager *RetentionManager) addRingLocked(entry retentionEntry, now time.Time) {
	manager.ring = append(manager.ring, entry)
	if len(manager.ring) > manager.options.RingCapacity {
		manager.ring = manager.ring[1:]
		manager.counters.Dropped++
	}
	manager.pruneRingLocked(now)
}

func (manager *RetentionManager) pruneRingLocked(now time.Time) {
	for len(manager.ring) > 0 {
		entry := manager.ring[0]
		if now.Sub(entry.occurredAt) > manager.options.MaxAge {
			manager.ring = manager.ring[1:]
			manager.counters.AgeEvicted++
			continue
		}
		if manager.ringBytesLocked() > manager.options.MaxBytes {
			manager.ring = manager.ring[1:]
			manager.counters.ByteEvicted++
			continue
		}
		break
	}
	manager.counters.RingRecords = uint64(len(manager.ring))
	manager.counters.RingBytes = manager.ringBytesLocked()
}

func (manager *RetentionManager) ringBytesLocked() int64 {
	var total int64
	for _, entry := range manager.ring {
		total += int64(len(entry.payload))
	}
	return total
}

func (manager *RetentionManager) resultLocked(status RetentionStatus, class ErrorClass, retryable bool) RetentionResult {
	manager.counters.RingRecords = uint64(len(manager.ring))
	manager.counters.RingBytes = manager.ringBytesLocked()
	return RetentionResult{Status: status, ErrorClass: class, Retryable: retryable, Counters: manager.counters}
}

func toStorageRecord(event Event, payload []byte) storage.DiagnosticRecord {
	eventKind := strings.ReplaceAll(string(event.ReasonCode), "_", ".")
	errorClass := strings.ToLower(string(event.ErrorClass))
	if errorClass == "" {
		errorClass = "none"
	}
	protocolVersion := event.ProtocolVersion
	if protocolVersion <= 0 {
		protocolVersion = 1
	}
	schemaVersion := event.StoreSchemaVersion
	if schemaVersion == 0 {
		schemaVersion = storage.SupportedSchemaVersion
	}
	ranking := event.RankingModelVersion
	if !storage.ValidDiagnosticVersionLabel(ranking) {
		ranking = ""
	}
	return storage.DiagnosticRecord{
		OccurredAt:          event.EventTime.UnixMilli(),
		EventKind:           eventKind,
		ErrorClass:          storage.ErrorClass(errorClass),
		Retryable:           event.Retryable,
		DurationMillis:      event.Timing.DurationMillis,
		ProjectionRevision:  event.ProjectionRevision,
		ProtocolVersion:     protocolVersion,
		RankingModelVersion: ranking,
		SchemaVersion:       schemaVersion,
		RedactedPayload:     base64.RawURLEncoding.EncodeToString(payload),
	}
}
