package diagnostics

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

type healthFixtureSnapshot struct {
	Host               HostState       `json:"host"`
	Protocol           ProtocolState   `json:"protocol"`
	Projection         ProjectionState `json:"projection"`
	Index              IndexState      `json:"index"`
	Storage            StorageState    `json:"storage"`
	ProjectionRevision uint64          `json:"projection_revision"`
	IndexedRevision    uint64          `json:"indexed_revision"`
	Retryable          bool            `json:"retryable"`
}

type healthFixtureLine struct {
	Step                  string                `json:"step"`
	Component             Component             `json:"component"`
	ReasonCode            ReasonCode            `json:"reason_code"`
	FromState             State                 `json:"from_state"`
	ToState               State                 `json:"to_state"`
	Snapshot              healthFixtureSnapshot `json:"snapshot"`
	ExpectedEventSequence uint64                `json:"expected_event_sequence"`
}

type healthFixtureFinal struct {
	State            State     `json:"state"`
	FinalHealth      State     `json:"final_health"`
	IndexFreshness   Freshness `json:"index_freshness"`
	LexicalAvailable bool      `json:"lexical_available"`
	EventCount       uint64    `json:"event_count"`
}

func fixturePath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "fixtures", "diagnostics", name)
}

func loadHealthFixture(t *testing.T) ([]healthFixtureLine, healthFixtureFinal) {
	t.Helper()
	file, err := os.Open(fixturePath("health-transitions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var lines []healthFixtureLine
	var final healthFixtureFinal
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var envelope struct {
			healthFixtureLine
			Final *healthFixtureFinal `json:"final,omitempty"`
		}
		if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
			t.Fatalf("health fixture line: %v", err)
		}
		if envelope.Final != nil {
			final = *envelope.Final
			continue
		}
		lines = append(lines, envelope.healthFixtureLine)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(lines) == 0 || final.EventCount == 0 {
		t.Fatal("health fixture is incomplete")
	}
	return lines, final
}

func runHealthFixture(t *testing.T) (*Health, healthFixtureFinal) {
	t.Helper()
	lines, final := loadHealthFixture(t)
	health, err := NewHealth(baseSnapshot(), func() time.Time { return time.Unix(1_100, 0).UTC() })
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		next := Snapshot{
			State:              line.ToState,
			Host:               line.Snapshot.Host,
			Protocol:           line.Snapshot.Protocol,
			Projection:         line.Snapshot.Projection,
			Index:              line.Snapshot.Index,
			Storage:            line.Snapshot.Storage,
			ProjectionRevision: line.Snapshot.ProjectionRevision,
			IndexedRevision:    line.Snapshot.IndexedRevision,
			Retryable:          line.Snapshot.Retryable,
		}
		if err := health.Transition(next); err != nil {
			t.Fatalf("fixture step %s: %v", line.Step, err)
		}
		if got := health.Snapshot().EventSequence; got != line.ExpectedEventSequence {
			t.Fatalf("fixture step %s sequence = %d, want %d", line.Step, got, line.ExpectedEventSequence)
		}
	}
	if got := health.Snapshot().State; got != final.FinalHealth {
		t.Fatalf("final health = %s, want %s", got, final.FinalHealth)
	}
	if got := health.Transitions(); got != final.EventCount {
		t.Fatalf("transition count = %d, want %d", got, final.EventCount)
	}
	return health, final
}

type repairFixtureCase struct {
	Name                 string            `json:"name"`
	Operation            RepairOperation   `json:"operation"`
	ConnectionFailures   int               `json:"connection_failures"`
	ExpectedStatus       RepairStatus      `json:"expected_status"`
	ExpectedAttempts     int               `json:"expected_attempts"`
	ExpectedAction       RecommendedAction `json:"expected_action"`
	ExpectedSecondStatus RepairStatus      `json:"expected_second_status"`
	ExpectedIndex        IndexState        `json:"expected_index"`
	DeadlineMS           int               `json:"deadline_ms"`
}

func loadRepairFixture(t *testing.T) []repairFixtureCase {
	t.Helper()
	raw, err := os.ReadFile(fixturePath("repair-bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []repairFixtureCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture.Cases
}

func TestRepairCancellation(t *testing.T) {
	var rebuildCase, persistenceCase *repairFixtureCase
	for index := range loadRepairFixture(t) {
		test := loadRepairFixture(t)[index]
		switch test.Name {
		case "rebuild_cancellation":
			rebuildCase = &test
		case "persistence_reopen_timeout":
			persistenceCase = &test
		}
	}
	if rebuildCase == nil || persistenceCase == nil {
		t.Fatal("repair fixture cases missing")
	}

	blocking := &fakeRepairBackend{block: make(chan struct{})}
	health, _, controller := repairFixture(t, readySnapshot(), blocking)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan RepairResult, 1)
	go func() { done <- controller.Run(ctx, RepairRebuildIndex, 7) }()
	deadline := time.Now().Add(time.Second)
	for !controller.Active() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	cancelled := <-done
	if cancelled.Status != rebuildCase.ExpectedStatus || health.Snapshot().Index != rebuildCase.ExpectedIndex {
		t.Fatalf("rebuild cancellation = %+v snapshot=%+v", cancelled, health.Snapshot())
	}

	blocking = &fakeRepairBackend{block: make(chan struct{})}
	_, _, controller = repairFixture(t, readySnapshot(), blocking)
	timeout := controller.Run(context.Background(), RepairRepairPersistence, 0)
	if timeout.Status != persistenceCase.ExpectedStatus || timeout.Attempts != persistenceCase.ExpectedAttempts {
		t.Fatalf("persistence timeout = %+v", timeout)
	}
}

func TestHealthFreshness(t *testing.T) {
	health, final := runHealthFixture(t)
	payload := health.Snapshot().Payload(time.Unix(1_200, 0).UTC())
	if payload.IndexFreshness != final.IndexFreshness {
		t.Fatalf("index freshness = %s, want %s", payload.IndexFreshness, final.IndexFreshness)
	}
	if health.Snapshot().LexicalAvailable() != final.LexicalAvailable {
		t.Fatalf("lexical availability = %t, want %t", health.Snapshot().LexicalAvailable(), final.LexicalAvailable)
	}
	if payload.State != final.FinalHealth {
		t.Fatalf("payload state = %s, want %s", payload.State, final.FinalHealth)
	}
}

func loadErrorFixture(t *testing.T) []struct {
	ErrorClass    ErrorClass        `json:"error_class"`
	Status        Status            `json:"status"`
	Retryable     bool              `json:"retryable"`
	Action        RecommendedAction `json:"action"`
	MaxAttempts   int               `json:"max_attempts"`
	TotalBudgetMS int64             `json:"total_budget_ms"`
} {
	t.Helper()
	raw, err := os.ReadFile(fixturePath("error-status-matrix.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			ErrorClass    ErrorClass        `json:"error_class"`
			Status        Status            `json:"status"`
			Retryable     bool              `json:"retryable"`
			Action        RecommendedAction `json:"action"`
			MaxAttempts   int               `json:"max_attempts"`
			TotalBudgetMS int64             `json:"total_budget_ms"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture.Cases
}

func TestRedaction(t *testing.T) {
	raw, err := os.ReadFile(fixturePath("redaction-sensitive-fields.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		SensitiveValues []string       `json:"sensitive_values"`
		Context         map[string]any `json:"context"`
		ExpectedAbsent  []string       `json:"expected_absent"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	sanitized := Redaction{}.SanitizeMap(fixture.Context)
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateExport(encoded, fixture.ExpectedAbsent...); err != nil {
		t.Fatalf("redaction fixture leaked: %v\n%s", err, encoded)
	}
	for _, key := range []string{"title", "url", "query", "token", "content", "browser_id", "tab_id", "private_context_data"} {
		if _, ok := sanitized[key]; ok {
			t.Fatalf("unknown sensitive key %q survived", key)
		}
	}
	if _, ok := sanitized["counts"].(map[string]any); !ok {
		t.Fatalf("counts shape = %T", sanitized["counts"])
	}
}

func TestRetentionBoundaries(t *testing.T) {
	raw, err := os.ReadFile(fixturePath("retention-boundaries.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Retention struct {
			MaxAgeSeconds  int64 `json:"max_age_seconds"`
			MaxBytes       int64 `json:"max_bytes"`
			MaxRecordBytes int   `json:"max_record_bytes"`
		} `json:"retention"`
		Cases []struct {
			Name                  string `json:"name"`
			AgeSeconds            int64  `json:"age_seconds"`
			Bytes                 int64  `json:"bytes"`
			ExpectedRetained      bool   `json:"expected_retained"`
			ExpectedTotalLTE      int64  `json:"expected_total_lte"`
			ExpectedOldestEvicted bool   `json:"expected_oldest_evicted"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Retention.MaxAgeSeconds != int64(DiagnosticRetention/time.Second) ||
		fixture.Retention.MaxBytes != MaxDiagnosticPayloadBytes ||
		fixture.Retention.MaxRecordBytes != MaxDiagnosticRecordBytes {
		t.Fatalf("fixture retention constants = %+v", fixture.Retention)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	for _, test := range fixture.Cases {
		if test.AgeSeconds == 0 {
			continue
		}
		store := &fakeRetentionStore{}
		manager, recorder := retentionFixture(t, store, now, RetentionOptions{})
		event := retentionEvent(t, recorder, now.Add(-time.Duration(test.AgeSeconds)*time.Second), ReasonHostStart)
		result := manager.Append(context.Background(), event)
		retained := result.Status == RetentionInserted
		if retained != test.ExpectedRetained {
			t.Fatalf("%s retained = %t, want %t: %+v", test.Name, retained, test.ExpectedRetained, result)
		}
	}

	for _, test := range fixture.Cases {
		if test.Bytes == 0 {
			continue
		}
		store := &fakeRetentionStore{maxBytes: fixture.Retention.MaxBytes}
		store.records = []storage.DiagnosticRecord{
			{OccurredAt: now.UnixMilli() - 2, RedactedPayload: "a"},
			{OccurredAt: now.UnixMilli() - 1, RedactedPayload: strings.Repeat("a", int(test.Bytes))},
		}
		manager, _ := retentionFixture(t, store, now, RetentionOptions{MaxBytes: fixture.Retention.MaxBytes})
		manager.Prune(context.Background())
		var total int64
		for _, record := range store.snapshot() {
			total += int64(len(record.RedactedPayload))
		}
		if total > test.ExpectedTotalLTE {
			t.Fatalf("%s total = %d > %d", test.Name, total, test.ExpectedTotalLTE)
		}
		if test.ExpectedOldestEvicted && len(store.snapshot()) >= 2 {
			t.Fatalf("%s did not evict oldest records", test.Name)
		}
	}
}

func TestPersistenceDegraded(t *testing.T) {
	snapshot := readySnapshot()
	snapshot.Storage = StorageDegraded
	health, err := NewHealth(snapshot, nil)
	if err != nil {
		t.Fatal(err)
	}
	current := health.Snapshot()
	if current.State != StateDegraded || !current.LexicalAvailable() || current.Ready() {
		t.Fatalf("degraded health = %+v", current)
	}
	payload := current.Payload(time.Unix(1_300, 0).UTC())
	if payload.Storage != StorageDegraded || payload.Wire().StorageState != "degraded" {
		t.Fatalf("degraded payload = %+v wire=%+v", payload, payload.Wire())
	}
	redacted := Redaction{}.RedactPayload(payload)
	if redacted["storage"] != "degraded" {
		t.Fatalf("redacted storage = %v", redacted["storage"])
	}
}
