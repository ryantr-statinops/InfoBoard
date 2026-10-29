package diagnostics_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/diagnostics"
)

func fixturePath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "fixtures", "diagnostics", name)
}

func TestDiagnosticsHealthFixture(t *testing.T) {
	file, err := os.Open(fixturePath("health-transitions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	health, err := diagnostics.NewHealth(diagnostics.Snapshot{
		State:      diagnostics.StateDisconnected,
		Host:       diagnostics.HostUnavailable,
		Protocol:   diagnostics.ProtocolUnknown,
		Projection: diagnostics.ProjectionUnknown,
		Index:      diagnostics.IndexUnknown,
		Storage:    diagnostics.StorageUnknown,
	}, func() time.Time {
		return time.Unix(1_100, 0).UTC()
	})
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	transitions := 0
	for scanner.Scan() {
		var line struct {
			Step     string `json:"step"`
			ToState  string `json:"to_state"`
			Snapshot struct {
				Host               diagnostics.HostState       `json:"host"`
				Protocol           diagnostics.ProtocolState   `json:"protocol"`
				Projection         diagnostics.ProjectionState `json:"projection"`
				Index              diagnostics.IndexState      `json:"index"`
				Storage            diagnostics.StorageState    `json:"storage"`
				ProjectionRevision uint64                      `json:"projection_revision"`
				IndexedRevision    uint64                      `json:"indexed_revision"`
				Retryable          bool                        `json:"retryable"`
			} `json:"snapshot"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			t.Fatal(err)
		}
		if line.Step == "" {
			continue
		}
		next := diagnostics.Snapshot{
			State:              diagnostics.State(line.ToState),
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
		transitions++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if transitions != 8 || health.Transitions() != 8 {
		t.Fatalf("transitions = %d/%d", transitions, health.Transitions())
	}
	if health.Snapshot().State != diagnostics.StateStopping {
		t.Fatalf("final state = %s", health.Snapshot().State)
	}
}

func TestDiagnosticsErrorMatrixFixture(t *testing.T) {
	raw, err := os.ReadFile(fixturePath("error-status-matrix.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			ErrorClass  diagnostics.ErrorClass        `json:"error_class"`
			Status      diagnostics.Status            `json:"status"`
			Retryable   bool                          `json:"retryable"`
			Action      diagnostics.RecommendedAction `json:"action"`
			MaxAttempts int                           `json:"max_attempts"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) != 15 {
		t.Fatalf("cases = %d", len(fixture.Cases))
	}
	for _, test := range fixture.Cases {
		mapping, ok := diagnostics.Lookup(test.ErrorClass)
		if !ok {
			t.Fatalf("missing mapping for %s", test.ErrorClass)
		}
		if mapping.Status != test.Status || mapping.Retryable != test.Retryable ||
			mapping.Action != test.Action || mapping.Policy.MaxAttempts != test.MaxAttempts {
			t.Fatalf("mapping %s = %+v", test.ErrorClass, mapping)
		}
		if err := mapping.Validate(); err != nil {
			t.Fatalf("mapping %s invalid: %v", test.ErrorClass, err)
		}
	}
}

func TestDiagnosticsRedactionFixture(t *testing.T) {
	raw, err := os.ReadFile(fixturePath("redaction-sensitive-fields.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Context        map[string]any `json:"context"`
		ExpectedAbsent []string       `json:"expected_absent"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	sanitized := (diagnostics.Redaction{}).SanitizeMap(fixture.Context)
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatal(err)
	}
	if err := diagnostics.ValidateExport(encoded, fixture.ExpectedAbsent...); err != nil {
		t.Fatalf("redaction fixture leaked: %v", err)
	}
}

func TestDiagnosticsRetentionAndRepairFixtures(t *testing.T) {
	raw, err := os.ReadFile(fixturePath("retention-boundaries.json"))
	if err != nil {
		t.Fatal(err)
	}
	var retention struct {
		Retention struct {
			MaxAgeSeconds  int64 `json:"max_age_seconds"`
			MaxBytes       int64 `json:"max_bytes"`
			MaxRecordBytes int   `json:"max_record_bytes"`
		} `json:"retention"`
	}
	if err := json.Unmarshal(raw, &retention); err != nil {
		t.Fatal(err)
	}
	if retention.Retention.MaxAgeSeconds != int64(diagnostics.DiagnosticRetention/time.Second) ||
		retention.Retention.MaxBytes != diagnostics.MaxDiagnosticPayloadBytes ||
		retention.Retention.MaxRecordBytes != diagnostics.MaxDiagnosticRecordBytes {
		t.Fatalf("retention fixture = %+v", retention.Retention)
	}

	raw, err = os.ReadFile(fixturePath("repair-bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	var repair struct {
		Cases []struct {
			Name      string                      `json:"name"`
			Operation diagnostics.RepairOperation `json:"operation"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &repair); err != nil {
		t.Fatal(err)
	}
	if len(repair.Cases) != 6 {
		t.Fatalf("repair cases = %d", len(repair.Cases))
	}
	for _, test := range repair.Cases {
		if !test.Operation.Valid() {
			t.Fatalf("repair case %s has invalid operation %s", test.Name, test.Operation)
		}
	}
}
