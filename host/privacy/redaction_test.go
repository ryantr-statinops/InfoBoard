package privacy

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/diagnostics"
)

func TestRedactionDropsSensitiveFieldsAndSentinels(t *testing.T) {
	var fixture struct {
		SensitiveValues []string       `json:"sensitive_values"`
		Context         map[string]any `json:"context"`
	}
	if err := json.Unmarshal(privacyFixture(t, "redaction-sentinels.json"), &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.SensitiveValues) < 8 {
		t.Fatalf("sentinels = %d", len(fixture.SensitiveValues))
	}
	input := map[string]any{
		"event_id":      fixture.SensitiveValues[0],
		"session_id":    fixture.SensitiveValues[5],
		"component":     fixture.SensitiveValues[3],
		"error_class":   fixture.SensitiveValues[8],
		"title_display": fixture.SensitiveValues[0],
		"url_display":   fixture.SensitiveValues[1],
		"nested":        map[string]any{"unknown": fixture.SensitiveValues[0]},
		"retryable":     true,
		"duration_ms":   int64(1),
		"counts": map[string]any{
			"records": int64(1), "dropped": int64(0), "retries": int64(0), "failures": int64(0),
			"leaked": fixture.SensitiveValues[0],
		},
	}
	redactor := NewRedactor()
	sanitized := redactor.SanitizeMap(input)
	if _, exists := sanitized["title_display"]; exists {
		t.Fatal("title_display survived redaction")
	}
	if _, exists := sanitized["nested"]; exists {
		t.Fatal("nested object survived redaction")
	}
	if _, exists := sanitized["event_id"]; exists {
		t.Fatal("invalid event_id survived redaction")
	}
	if sanitized["retryable"] != true {
		t.Fatalf("retryable = %v", sanitized["retryable"])
	}
	export, err := redactor.ExportRecords([]map[string]any{sanitized})
	if err != nil {
		t.Fatal(err)
	}
	if err := redactor.ValidateExport(export, fixture.SensitiveValues...); err != nil {
		t.Fatalf("sensitive export: %v", err)
	}
	for _, sentinel := range fixture.SensitiveValues {
		if bytes.Contains(export, []byte(sentinel)) {
			t.Fatalf("sentinel survived export: %s", sentinel)
		}
	}
}

func TestRedactionDisplayURLDigestAndSafeError(t *testing.T) {
	redactor := NewRedactor()
	raw := "https://user:SENTINEL-PASSWORD@example.test/path?token=SENTINEL-TOKEN#SENTINEL-FRAGMENT"
	display := redactor.DisplayURL(raw)
	if display != "https://example.test/path" {
		t.Fatalf("display URL = %q", display)
	}
	digest := redactor.Correlation("SENTINEL-PRIVATE-ID-42")
	if !strings.HasPrefix(digest, "local-sha256:") || strings.Contains(digest, "SENTINEL") {
		t.Fatalf("digest = %q", digest)
	}
	if digest != redactor.Correlation("SENTINEL-PRIVATE-ID-42") {
		t.Fatal("digest is not deterministic")
	}
	safe := redactor.RedactError(redactor.SafeError(diagnostics.ErrorClass("storage_unavailable"), errors.New("SENTINEL-ERROR-TEXT")))
	if strings.Contains(safe, "SENTINEL") || safe == "" {
		t.Fatalf("safe error = %q", safe)
	}
	payload := redactor.RedactDiagnosticPayload("https://example.test/path?token=SENTINEL")
	if strings.Contains(payload, "?") || strings.Contains(payload, "/") {
		t.Fatalf("diagnostic payload was not scrubbed: %q", payload)
	}
}
