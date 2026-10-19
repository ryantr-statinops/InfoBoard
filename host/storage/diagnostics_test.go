package storage

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

// excludedContentSentinels are representative values from every T10-excluded
// class: tab title, full URL, query string, cookie/token, SQL text, and file
// path. FR-015/NFR-009 require that none of these survive in stored
// diagnostics or health output.
var excludedContentSentinels = []string{
	"My Secret Tab Title",
	"https://example.test/search?q=private+query",
	"SELECT * FROM activation_metadata",
	"session_token=abc123-secret",
	"/home/user/.config/infoboard.sqlite",
	"C:\\Users\\user\\secret\\tabs.json",
}

func validRecord() DiagnosticRecord {
	return DiagnosticRecord{
		OccurredAt:          time.Now().UnixMilli(),
		EventKind:           "storage.open_failed",
		ErrorClass:          ErrorStorageFailure,
		ProtocolVersion:     protocol.CurrentVersion,
		RankingModelVersion: "rank-1",
		SchemaVersion:       1,
		RedactedPayload:     RedactDiagnosticPayload("attempt_3"),
	}
}

func TestRedactDiagnosticPayloadEliminatesExcludedContent(t *testing.T) {
	input := strings.Join(excludedContentSentinels, " | ")
	redacted := RedactDiagnosticPayload(input)
	for _, sentinel := range excludedContentSentinels {
		if strings.Contains(redacted, sentinel) {
			t.Fatalf("redacted payload preserved excluded content %q in %q", sentinel, redacted)
		}
	}
	if len(redacted) > len(input) {
		t.Fatalf("redaction grew payload from %d to %d bytes", len(input), len(redacted))
	}
	if roundTripped := RedactDiagnosticPayload(redacted); roundTripped != redacted {
		t.Fatalf("redaction is not a fixed point: %q became %q", redacted, roundTripped)
	}
}

func TestDiagnosticVocabularyRejectsExcludedContent(t *testing.T) {
	if ValidDiagnosticEventKind("") || ValidDiagnosticErrorClass("") || ValidDiagnosticVersionLabel("") {
		t.Fatal("empty diagnostic field accepted")
	}
	for _, sentinel := range excludedContentSentinels {
		if ValidDiagnosticEventKind(sentinel) {
			t.Fatalf("event kind accepted excluded content %q", sentinel)
		}
		if ValidDiagnosticErrorClass(sentinel) {
			t.Fatalf("error class accepted excluded content %q", sentinel)
		}
		if ValidDiagnosticVersionLabel(sentinel) {
			t.Fatalf("version label accepted excluded content %q", sentinel)
		}
	}
	for _, kind := range []string{"storage.open_failed", "sqlite.schema_mismatch", "host.lifecycle_ready"} {
		if !ValidDiagnosticEventKind(kind) {
			t.Fatalf("closed event kind %q rejected", kind)
		}
	}
	for _, class := range []string{"storage_unavailable", "storage_failure", "database_corrupt"} {
		if !ValidDiagnosticErrorClass(class) {
			t.Fatalf("closed error class %q rejected", class)
		}
	}
	for _, version := range []string{"rank-1", "1.4.0+build.2"} {
		if !ValidDiagnosticVersionLabel(version) {
			t.Fatalf("opaque version label %q rejected", version)
		}
	}
}

func TestAppendDiagnosticRejectsUnredactedContent(t *testing.T) {
	_, repository := newSchemaRepository(t)
	raw := validRecord()
	raw.RedactedPayload = strings.Join(excludedContentSentinels, " ")
	if result := repository.AppendDiagnostic(context.Background(), raw); result.ErrorClass != ErrorInvalidConfiguration {
		t.Fatalf("unredacted diagnostic result = %+v", result)
	}
	titleKind := validRecord()
	titleKind.EventKind = "My Secret Tab Title"
	tokenClass := validRecord()
	tokenClass.ErrorClass = "session_token=abc123-secret"
	sqlPathLabel := validRecord()
	sqlPathLabel.RankingModelVersion = "/home/user/.config/infoboard.sqlite"
	for index, candidate := range []DiagnosticRecord{titleKind, tokenClass, sqlPathLabel} {
		if result := repository.AppendDiagnostic(context.Background(), candidate); result.ErrorClass != ErrorInvalidConfiguration {
			t.Fatalf("excluded diagnostic %d stored with result %+v", index, result)
		}
	}
}

func TestStoredDiagnosticsContainNoExcludedContent(t *testing.T) {
	db, repository := newSchemaRepository(t)
	record := validRecord()
	record.RedactedPayload = RedactDiagnosticPayload(strings.Join(excludedContentSentinels, " "))
	if result := repository.AppendDiagnostic(context.Background(), record); result.Status != StatusReady {
		t.Fatalf("append redacted diagnostic = %+v", result)
	}
	var stored string
	if err := db.QueryRow(`SELECT redacted_payload FROM diagnostics`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	for _, sentinel := range excludedContentSentinels {
		if strings.Contains(stored, sentinel) {
			t.Fatalf("stored diagnostics contain excluded content %q: %q", sentinel, stored)
		}
	}
	if stored != RedactDiagnosticPayload(stored) {
		t.Fatalf("stored payload is not in redacted normal form: %q", stored)
	}
}

func TestStoredDiagnosticsHaveBoundedShape(t *testing.T) {
	db, repository := newSchemaRepository(t)
	record := validRecord()
	if result := repository.AppendDiagnostic(context.Background(), record); result.Status != StatusReady {
		t.Fatalf("append diagnostic = %+v", result)
	}
	var kind, class, ranking, payload string
	var payloadBytes int
	row := db.QueryRow(`SELECT event_kind, error_class, ranking_model_version, redacted_payload, payload_bytes FROM diagnostics`)
	if err := row.Scan(&kind, &class, &ranking, &payload, &payloadBytes); err != nil {
		t.Fatal(err)
	}
	if !ValidDiagnosticEventKind(kind) || !ValidDiagnosticErrorClass(class) || !ValidDiagnosticVersionLabel(ranking) {
		t.Fatalf("stored fields outside closed vocabulary: kind=%q class=%q ranking=%q", kind, class, ranking)
	}
	if payloadBytes != len(payload) {
		t.Fatalf("stored payload_bytes=%d does not match UTF-8 length %d", payloadBytes, len(payload))
	}
	var encoded []byte
	encoded, _ = json.Marshal(map[string]string{"redacted_payload": payload})
	for _, sentinel := range excludedContentSentinels {
		if strings.Contains(string(encoded), sentinel) {
			t.Fatalf("exported diagnostic JSON contains excluded content %q", sentinel)
		}
	}
}
