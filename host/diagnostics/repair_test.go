package diagnostics

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeRepairBackend struct {
	mu                 sync.Mutex
	connectionCalls    int
	connectionFailures int
	resyncCalls        []uint64
	resyncStarted      chan uint64
	rebuildErr         error
	persistenceErr     error
	block              chan struct{}
}

func (backend *fakeRepairBackend) RetryConnection(ctx context.Context, attempt int) error {
	backend.mu.Lock()
	backend.connectionCalls++
	failures := backend.connectionFailures
	block := backend.block
	backend.mu.Unlock()
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if attempt <= failures {
		return errors.New("connection refused")
	}
	return nil
}

func (backend *fakeRepairBackend) ResyncProjection(ctx context.Context, revision uint64) error {
	backend.mu.Lock()
	backend.resyncCalls = append(backend.resyncCalls, revision)
	started := backend.resyncStarted
	backend.mu.Unlock()
	if started != nil {
		started <- revision
	}
	if revision == 1 {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func (backend *fakeRepairBackend) RebuildIndex(ctx context.Context, revision uint64) error {
	backend.mu.Lock()
	block := backend.block
	err := backend.rebuildErr
	backend.mu.Unlock()
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

func (backend *fakeRepairBackend) RepairPersistence(ctx context.Context) error {
	backend.mu.Lock()
	block := backend.block
	err := backend.persistenceErr
	backend.mu.Unlock()
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

func repairFixture(t *testing.T, snapshot Snapshot, backend RepairBackend) (*Health, *Recorder, *RepairController) {
	t.Helper()
	now := func() time.Time { return time.Unix(1_000, 0).UTC() }
	health, err := NewHealth(snapshot, now)
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := NewRecorder("session-repair", 32, now)
	if err != nil {
		t.Fatal(err)
	}
	controller := NewRepairController(health, recorder, backend, RepairOptions{
		RetryConnectionBudget: 100 * time.Millisecond,
		PersistenceDeadline:   50 * time.Millisecond,
		Backoff:               []time.Duration{time.Millisecond, 2 * time.Millisecond},
		Cooldown:              20 * time.Millisecond,
		Now:                   now,
		Sleep:                 func(context.Context, time.Duration) error { return nil },
	})
	return health, recorder, controller
}

func TestRepairBounds(t *testing.T) {
	fixture := loadRepairFixture(t)
	if len(fixture) == 0 || fixture[0].Operation != RepairRetryConnection || fixture[0].ExpectedAttempts != 3 {
		t.Fatalf("repair fixture bounds = %+v", fixture)
	}
	backend := &fakeRepairBackend{connectionFailures: 2}
	_, recorder, controller := repairFixture(t, baseSnapshot(), backend)
	result := controller.Run(context.Background(), RepairRetryConnection, 0)
	if err := result.Validate(); err != nil {
		t.Fatalf("result invalid: %v", err)
	}
	if result.Status != RepairSucceeded || result.Attempts != 3 || result.Retryable {
		t.Fatalf("result = %+v", result)
	}
	if backend.connectionCalls != 3 {
		t.Fatalf("connection calls = %d", backend.connectionCalls)
	}
	events := recorder.Events()
	if len(events) != 2 || events[0].ReasonCode != ReasonRepairStart || events[1].ReasonCode != ReasonRepairFinish {
		t.Fatalf("events = %+v", events)
	}
}

func TestRepairExhaustionAndCooldown(t *testing.T) {
	backend := &fakeRepairBackend{connectionFailures: 99}
	_, recorder, controller := repairFixture(t, baseSnapshot(), backend)
	result := controller.Run(context.Background(), RepairRetryConnection, 0)
	if result.Status != RepairExhausted || result.Attempts != 3 || !result.Retryable {
		t.Fatalf("exhausted result = %+v", result)
	}
	if result.Action != ActionOpenDiagnostics || result.ErrorClass != ErrorHostMissing {
		t.Fatalf("exhausted action/class = %s/%s", result.Action, result.ErrorClass)
	}
	cooldown := controller.Run(context.Background(), RepairRetryConnection, 0)
	if cooldown.Status != RepairCooldown || !cooldown.Retryable {
		t.Fatalf("cooldown result = %+v", cooldown)
	}
	events := recorder.Events()
	foundExhausted := false
	for _, event := range events {
		if event.ReasonCode == ReasonRepairExhausted {
			foundExhausted = true
		}
	}
	if !foundExhausted {
		t.Fatalf("missing repair_exhausted event: %+v", events)
	}
}

func TestRepairOneInFlight(t *testing.T) {
	backend := &fakeRepairBackend{block: make(chan struct{})}
	_, _, controller := repairFixture(t, baseSnapshot(), backend)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan RepairResult, 1)
	go func() { done <- controller.Run(ctx, RepairRetryConnection, 0) }()
	deadline := time.Now().Add(time.Second)
	for !controller.Active() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !controller.Active() {
		t.Fatal("repair did not become active")
	}
	second := controller.Run(context.Background(), RepairRetryConnection, 0)
	if second.Status != RepairInFlight {
		t.Fatalf("second result = %+v", second)
	}
	cancel()
	first := <-done
	if first.Status != RepairCancelled {
		t.Fatalf("cancelled result = %+v", first)
	}
}

func TestRepairIndexSuccessAndCancellation(t *testing.T) {
	backend := &fakeRepairBackend{}
	health, recorder, controller := repairFixture(t, readySnapshot(), backend)
	result := controller.Run(context.Background(), RepairRebuildIndex, 7)
	if result.Status != RepairSucceeded || result.FinalHealth != StateReady {
		t.Fatalf("index result = %+v", result)
	}
	snapshot := health.Snapshot()
	if snapshot.Index != IndexReady || snapshot.IndexedRevision != 7 || snapshot.ProjectionRevision != 7 {
		t.Fatalf("index snapshot = %+v", snapshot)
	}
	if len(recorder.Events()) < 3 {
		t.Fatalf("index events = %+v", recorder.Events())
	}

	blocking := &fakeRepairBackend{block: make(chan struct{})}
	health, _, controller = repairFixture(t, readySnapshot(), blocking)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan RepairResult, 1)
	go func() { done <- controller.Run(ctx, RepairRebuildIndex, 7) }()
	deadline := time.Now().Add(time.Second)
	for !controller.Active() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	cancelled := <-done
	if cancelled.Status != RepairCancelled {
		t.Fatalf("rebuild cancellation = %+v", cancelled)
	}
	if health.Snapshot().Index != IndexFailed || health.Snapshot().State != StateRecovering {
		t.Fatalf("cancelled index snapshot = %+v", health.Snapshot())
	}
}

func TestRepairPersistenceSchemaUpgradeStops(t *testing.T) {
	backend := &fakeRepairBackend{persistenceErr: NewSafeError(ErrorSchemaUpgradeRequired, errors.New("future schema"))}
	_, _, controller := repairFixture(t, readySnapshot(), backend)
	result := controller.Run(context.Background(), RepairRepairPersistence, 0)
	if result.Status != RepairExhausted || result.Attempts != 1 {
		t.Fatalf("schema upgrade result = %+v", result)
	}
	if result.Action != ActionUpdateHost || result.ErrorClass != ErrorSchemaUpgradeRequired {
		t.Fatalf("schema upgrade action/class = %s/%s", result.Action, result.ErrorClass)
	}
}

func TestRepairPersistenceSuccessRestoresHealth(t *testing.T) {
	snapshot := readySnapshot()
	snapshot.Storage = StorageDegraded
	backend := &fakeRepairBackend{}
	health, _, controller := repairFixture(t, snapshot, backend)
	result := controller.Run(context.Background(), RepairRepairPersistence, 0)
	if result.Status != RepairSucceeded {
		t.Fatalf("persistence result = %+v", result)
	}
	if health.Snapshot().Storage != StorageHealthy || health.Snapshot().State != StateReady {
		t.Fatalf("persistence snapshot = %+v", health.Snapshot())
	}
}

func TestResyncNewerRevisionCancelsOlder(t *testing.T) {
	backend := &fakeRepairBackend{resyncStarted: make(chan uint64, 4)}
	_, _, controller := repairFixture(t, readySnapshot(), backend)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	first := make(chan RepairResult, 1)
	go func() { first <- controller.Resync(ctx, 1) }()
	select {
	case revision := <-backend.resyncStarted:
		if revision != 1 {
			t.Fatalf("first revision = %d", revision)
		}
	case <-ctx.Done():
		t.Fatal("first resync did not start")
	}
	second := make(chan RepairResult, 1)
	go func() { second <- controller.Resync(ctx, 2) }()
	select {
	case revision := <-backend.resyncStarted:
		if revision != 2 {
			t.Fatalf("second revision = %d", revision)
		}
	case <-ctx.Done():
		t.Fatal("second resync did not start")
	}
	firstResult := <-first
	if firstResult.Status != RepairCancelled {
		t.Fatalf("first result = %+v", firstResult)
	}
	secondResult := <-second
	if secondResult.Status != RepairSucceeded {
		t.Fatalf("second result = %+v", secondResult)
	}
}
