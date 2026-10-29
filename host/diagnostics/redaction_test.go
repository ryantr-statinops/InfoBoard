package diagnostics

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

func TestRedactionDropsSensitiveFields(t *testing.T) {
	secretURL := "https://example.test/path?q=secret-token#fragment"
	secretTitle := "Private payroll tab title"
	secretQuery := "query=secret search"
	secretToken := "Bearer abc.def.ghi"
	secretContent := "page body content"
	sensitive := []string{secretURL, secretTitle, secretQuery, secretToken, secretContent, "browser-42", "tab-99"}
	input := map[string]any{
		"event_id":              secretURL,
		"session_id":            secretToken,
		"component":             secretTitle,
		"from_state":            secretQuery,
		"to_state":              secretContent,
		"reason_code":           secretURL,
		"error_class":           secretTitle,
		"status":                secretURL,
		"recommended_action":    secretQuery,
		"ranking_model_version": secretURL,
		"host_version":          secretContent,
		"profile_scope":         secretToken,
		"context_kind":          secretTitle,
		"counts": map[string]any{
			"records":       secretTitle,
			"unknown_count": 7,
		},
		"backoff_ms":   []any{secretTitle, 100},
		"title":        secretTitle,
		"url":          secretURL,
		"query":        secretQuery,
		"token":        secretToken,
		"content":      secretContent,
		"browser_id":   "browser-42",
		"tab_id":       "tab-99",
		"private_data": "private context payload",
	}
	sanitized := Redaction{}.SanitizeMap(input)
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateExport(encoded, sensitive...); err != nil {
		t.Fatalf("sensitive value reached export: %v\n%s", err, encoded)
	}
	for _, key := range []string{"title", "url", "query", "token", "content", "browser_id", "tab_id", "private_data"} {
		if _, ok := sanitized[key]; ok {
			t.Fatalf("unknown key %q survived", key)
		}
	}
	if counts, ok := sanitized["counts"].(map[string]any); ok {
		if _, ok := counts["unknown_count"]; ok {
			t.Fatal("unknown nested count survived")
		}
	}
}

func TestRedactionPreservesBoundedFields(t *testing.T) {
	timestamp := time.Unix(1_700_000_000, 0).UTC()
	input := map[string]any{
		"schema_version":        EventSchemaVersion,
		"event_id":              "evt-session-1-1",
		"event_time":            timestamp,
		"session_id":            "session-1",
		"component":             "storage",
		"from_state":            "ready",
		"to_state":              "degraded",
		"reason_code":           "storage_degradation",
		"error_class":           "PERSISTENCE_DEGRADED",
		"retryable":             true,
		"recommended_action":    "repair_persistence",
		"protocol_version":      1,
		"ranking_model_version": "rank-1",
		"store_schema_version":  2,
		"projection_revision":   7,
		"indexed_revision":      7,
		"projection_sequence":   3,
		"duration_ms":           12,
		"queue_ms":              1,
		"latency_bucket":        "fast",
		"backoff_ms":            []int64{100, 500, 1_000},
		"counts":                map[string]any{"records": 7, "dropped": 1},
	}
	sanitized := Redaction{}.SanitizeMap(input)
	for key := range input {
		if _, ok := sanitized[key]; !ok {
			t.Fatalf("bounded field %q dropped", key)
		}
	}
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "storage_degradation") || !strings.Contains(string(encoded), "rank-1") {
		t.Fatalf("bounded fields missing: %s", encoded)
	}
	if !strings.Contains(string(encoded), "2023-11-14T22:13:20Z") {
		t.Fatalf("timestamp not normalized: %s", encoded)
	}
}

func TestRedactEventAndPayload(t *testing.T) {
	event := Event{
		SchemaVersion:       EventSchemaVersion,
		EventID:             "evt-session-1-1",
		EventTime:           time.Unix(10, 0).UTC(),
		SessionID:           "session-1",
		Component:           ComponentStorage,
		ReasonCode:          ReasonStorageDegradation,
		ErrorClass:          ErrorPersistenceDegraded,
		RecommendedAction:   ActionRepairPersistence,
		RankingModelVersion: "https://example.test/?token=secret",
		Timing:              EventTiming{Latency: LatencySlow},
	}
	redacted := Redaction{}.RedactEvent(event)
	encoded, err := json.Marshal(redacted)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateExport(encoded, "https://example.test", "token=secret"); err != nil {
		t.Fatalf("event leaked: %v\n%s", err, encoded)
	}
	if _, ok := redacted["ranking_model_version"]; ok {
		t.Fatal("unsafe ranking version survived")
	}

	profile := domain.ProfileID("123e4567-e89b-42d3-a456-426614174000")
	payload := readySnapshot().Payload(time.Unix(20, 0).UTC())
	payload.ProfileID = profile
	payload.ContextKind = domain.ContextPrivate
	payload.HostVersion = "https://example.test/?token=secret"
	redactedPayload := Redaction{}.RedactPayload(payload)
	encoded, err = json.Marshal(redactedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateExport(encoded, "https://example.test", "token=secret", string(profile)); err != nil {
		t.Fatalf("payload leaked: %v\n%s", err, encoded)
	}
	if redactedPayload["context_kind"] != "private" {
		t.Fatalf("context kind = %v", redactedPayload["context_kind"])
	}
	scope, _ := redactedPayload["profile_scope"].(string)
	if !strings.HasPrefix(scope, "local-sha256:") {
		t.Fatalf("profile scope = %q", scope)
	}
	if ProfileScope(profile) == ProfileScope(domain.ProfileID("223e4567-e89b-42d3-a456-426614174000")) {
		t.Fatal("profile scopes collided")
	}
}

func TestSafeErrorAndMappingRedaction(t *testing.T) {
	secret := "https://example.test/?token=secret"
	err := NewSafeError(ErrorQueryTimeout, errors.New(secret))
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("safe error leaked cause: %s", err.Error())
	}
	if got := RedactError(err); got != string(StatusQueryTimedOut) {
		t.Fatalf("safe error status = %s", got)
	}
	if got := RedactError(errors.New(secret)); got != string(StatusRuntimeRecoverableError) {
		t.Fatalf("unknown error status = %s", got)
	}
	mapping, _ := Lookup(ErrorHostMissing)
	redacted := Redaction{}.RedactMapping(mapping)
	encoded, err := json.Marshal(redacted)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateExport(encoded, secret); err != nil {
		t.Fatalf("mapping leaked: %v", err)
	}
	if redacted["status"] != "runtime_unavailable" || redacted["retryable"] != true {
		t.Fatalf("mapping projection = %+v", redacted)
	}
}

func TestExportEventsAndRecords(t *testing.T) {
	recorder, err := NewRecorder("session-1", 4, func() time.Time { return time.Unix(30, 0).UTC() })
	if err != nil {
		t.Fatal(err)
	}
	event, err := recorder.Record(Event{
		Component: ComponentHost, ReasonCode: ReasonHostStart, RecommendedAction: ActionNone,
		Timing: EventTiming{Latency: LatencyFast},
	})
	if err != nil {
		t.Fatal(err)
	}
	export, err := Redaction{}.ExportEvents([]Event{event})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateExport(export); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(export), "host_start") {
		t.Fatalf("export missing reason: %s", export)
	}
	records, err := Redaction{}.ExportRecords([]map[string]any{
		{"status": "runtime_unavailable", "title": "secret title"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(records), "secret title") {
		t.Fatalf("record leaked: %s", records)
	}
}
