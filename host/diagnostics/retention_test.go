package diagnostics

import (
	"context"
	"encoding/base64"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

type fakeRetentionStore struct {
	mu         sync.Mutex
	records    []storage.DiagnosticRecord
	fail       bool
	pruneCalls int
	maxAge     time.Duration
	maxBytes   int64
}

func (store *fakeRetentionStore) AppendDiagnostic(_ context.Context, record storage.DiagnosticRecord) storage.Result {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.fail {
		return storage.Result{Status: storage.StatusDegraded, ErrorClass: storage.ErrorStorageUnavailable, Retryable: true}
	}
	store.records = append(store.records, record)
	return storage.Result{Status: storage.StatusReady}
}

func (store *fakeRetentionStore) PruneRetention(_ context.Context, now time.Time) storage.Result {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.pruneCalls++
	if store.fail {
		return storage.Result{Status: storage.StatusDegraded, ErrorClass: storage.ErrorStorageUnavailable, Retryable: true}
	}
	if store.maxAge <= 0 {
		store.maxAge = DiagnosticRetention
	}
	if store.maxBytes <= 0 {
		store.maxBytes = MaxDiagnosticPayloadBytes
	}
	cutoff := now.Add(-store.maxAge).UnixMilli()
	kept := make([]storage.DiagnosticRecord, 0, len(store.records))
	for _, record := range store.records {
		if record.OccurredAt >= cutoff {
			kept = append(kept, record)
		}
	}
	store.records = kept
	var total int64
	for _, record := range store.records {
		total += int64(len(record.RedactedPayload))
	}
	for total > store.maxBytes && len(store.records) > 0 {
		total -= int64(len(store.records[0].RedactedPayload))
		store.records = store.records[1:]
	}
	return storage.Result{Status: storage.StatusReady}
}

func (store *fakeRetentionStore) snapshot() []storage.DiagnosticRecord {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]storage.DiagnosticRecord(nil), store.records...)
}

func retentionFixture(t *testing.T, store RetentionStore, now time.Time, options RetentionOptions) (*RetentionManager, *Recorder) {
	t.Helper()
	recorder, err := NewRecorder("session-retention", 256, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if options.Now == nil {
		options.Now = func() time.Time { return now }
	}
	manager := NewRetentionManager(store, recorder, options)
	return manager, recorder
}

func retentionEvent(t *testing.T, recorder *Recorder, at time.Time, reason ReasonCode) Event {
	t.Helper()
	event, err := recorder.Record(Event{
		EventTime:         at.UTC(),
		Component:         ComponentHost,
		ReasonCode:        reason,
		RecommendedAction: ActionNone,
		Timing:            EventTiming{Latency: LatencyFast},
	})
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func TestRetentionConstants(t *testing.T) {
	if DiagnosticRetention != 7*24*time.Hour {
		t.Fatalf("retention = %s", DiagnosticRetention)
	}
	if MaxDiagnosticPayloadBytes != 10_000_000 {
		t.Fatalf("byte budget = %d", MaxDiagnosticPayloadBytes)
	}
	if MaxDiagnosticRecordBytes != 64*1024 {
		t.Fatalf("record budget = %d", MaxDiagnosticRecordBytes)
	}
}

func TestRetentionAgeAndByteBoundaries(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	store := &fakeRetentionStore{}
	manager, recorder := retentionFixture(t, store, now, RetentionOptions{
		MaxBytes:     900,
		RingCapacity: 16,
	})
	exact := retentionEvent(t, recorder, now.Add(-DiagnosticRetention), ReasonHostStart)
	if result := manager.Append(context.Background(), exact); result.Status != RetentionInserted {
		t.Fatalf("exactly seven days old = %+v", result)
	}
	older := retentionEvent(t, recorder, now.Add(-DiagnosticRetention-time.Second), ReasonHostStart)
	result := manager.Append(context.Background(), older)
	if result.Status != RetentionRejected || result.Counters.AgeEvicted != 1 {
		t.Fatalf("older event = %+v", result)
	}

	store.fail = true
	for index := 0; index < 8; index++ {
		event := retentionEvent(t, recorder, now, ReasonHostStart)
		result = manager.Append(context.Background(), event)
		if result.Status != RetentionDegraded {
			t.Fatalf("degraded append %d = %+v", index, result)
		}
	}
	counters := manager.Counters()
	if counters.RingBytes > 900 || counters.ByteEvicted == 0 || counters.RingRecords == 0 {
		t.Fatalf("ring counters = %+v", counters)
	}
	if len(manager.RingEvents()) == 0 {
		t.Fatal("degraded ring is empty")
	}
}

func TestRetentionRejectsOversizedAndMalformed(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	store := &fakeRetentionStore{}
	manager, recorder := retentionFixture(t, store, now, RetentionOptions{MaxRecordBytes: 80})
	event := retentionEvent(t, recorder, now, ReasonHostStart)
	result := manager.Append(context.Background(), event)
	if result.Status != RetentionRejected || result.Counters.Rejected != 1 {
		t.Fatalf("oversized result = %+v", result)
	}
	if len(store.snapshot()) != 0 {
		t.Fatal("oversized event reached the store")
	}

	malformed := event
	malformed.ReasonCode = ReasonCode("not a bounded reason")
	result = manager.Append(context.Background(), malformed)
	if result.Status != RetentionRejected || result.Counters.Rejected != 2 {
		t.Fatalf("malformed result = %+v", result)
	}
}

func TestRetentionFlushesPendingRingAfterRecovery(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	store := &fakeRetentionStore{fail: true}
	manager, recorder := retentionFixture(t, store, now, RetentionOptions{})
	for index := 0; index < 2; index++ {
		result := manager.Append(context.Background(), retentionEvent(t, recorder, now, ReasonHostStart))
		if result.Status != RetentionDegraded {
			t.Fatalf("pending append = %+v", result)
		}
	}
	if !manager.PersistenceDegraded() || len(manager.RingEvents()) != 2 {
		t.Fatalf("pending state = degraded:%t ring:%d", manager.PersistenceDegraded(), len(manager.RingEvents()))
	}
	store.mu.Lock()
	store.fail = false
	store.mu.Unlock()
	result := manager.Prune(context.Background())
	if result.Status != RetentionPruned {
		t.Fatalf("prune result = %+v", result)
	}
	if len(manager.RingEvents()) != 0 || len(store.snapshot()) != 2 {
		t.Fatalf("flush state = ring:%d store:%d", len(manager.RingEvents()), len(store.snapshot()))
	}
	if manager.Counters().Inserted != 2 || manager.Counters().PersistenceFailed < 2 {
		t.Fatalf("flush counters = %+v", manager.Counters())
	}
}

func TestRetentionPruneFailureDoesNotLoop(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	store := &fakeRetentionStore{fail: true}
	manager, recorder := retentionFixture(t, store, now, RetentionOptions{})
	manager.Append(context.Background(), retentionEvent(t, recorder, now, ReasonHostStart))
	before := manager.Counters()
	for index := 0; index < 3; index++ {
		result := manager.Prune(context.Background())
		if result.Status != RetentionDegraded {
			t.Fatalf("prune failure %d = %+v", index, result)
		}
	}
	after := manager.Counters()
	if after.PersistenceFailed <= before.PersistenceFailed || len(manager.RingEvents()) != 1 {
		t.Fatalf("failure loop state before=%+v after=%+v ring=%d", before, after, len(manager.RingEvents()))
	}
}

func TestRetentionStoreSeamPrunesToBudget(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	store := &fakeRetentionStore{maxBytes: 1_000}
	manager, recorder := retentionFixture(t, store, now, RetentionOptions{})
	for index := 0; index < 20; index++ {
		result := manager.Append(context.Background(), retentionEvent(t, recorder, now, ReasonHostStart))
		if result.Status != RetentionInserted {
			t.Fatalf("insert %d = %+v", index, result)
		}
	}
	var total int64
	for _, record := range store.snapshot() {
		total += int64(len(record.RedactedPayload))
	}
	if total > store.maxBytes {
		t.Fatalf("store bytes = %d > %d", total, store.maxBytes)
	}
	if store.pruneCalls == 0 {
		t.Fatal("store prune seam was not called")
	}
	if manager.Counters().Inserted != 20 {
		t.Fatalf("inserted counter = %d", manager.Counters().Inserted)
	}
}

func TestRetentionStoragePayloadIsRedacted(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	store := &fakeRetentionStore{}
	manager, recorder := retentionFixture(t, store, now, RetentionOptions{})
	event := retentionEvent(t, recorder, now, ReasonHostStart)
	event.RankingModelVersion = "https://example.test/?token=secret"
	if result := manager.Append(context.Background(), event); result.Status != RetentionInserted {
		t.Fatalf("append = %+v", result)
	}
	records := store.snapshot()
	if len(records) != 1 {
		t.Fatalf("records = %d", len(records))
	}
	if records[0].RankingModelVersion != "" {
		t.Fatalf("unsafe ranking version persisted: %q", records[0].RankingModelVersion)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(records[0].RedactedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(decoded), "example.test") || strings.Contains(string(decoded), "token=secret") {
		t.Fatalf("sensitive payload persisted: %s", decoded)
	}
	if err := ValidateExport(decoded, "example.test", "token=secret"); err != nil {
		t.Fatal(err)
	}
}
