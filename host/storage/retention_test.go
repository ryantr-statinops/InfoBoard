package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

const retentionProfileA = "123e4567-e89b-42d3-a456-426614174000"
const retentionProfileB = "123e4567-e89b-42d3-a456-426614174099"

func TestRecordActivationHonorsOptInAndKeepsNewestPerProfile(t *testing.T) {
	db, repository := newSchemaRepository(t)
	ctx := context.Background()
	versions := CompatibilityVersions{ProtocolVersion: protocol.CurrentVersion, RankingModelVersion: "rank-current"}
	configurationA := DefaultConfiguration(domain.ProfileID(retentionProfileA))
	configurationA.PersistActivationRecency = true
	configurationA.RankingModelVersion = versions.RankingModelVersion
	if result := repository.SaveConfiguration(ctx, configurationA, versions); result.Status != StatusReady {
		t.Fatalf("enable profile A activation persistence: %+v", result)
	}

	now := time.Now().Unix()
	for index := range MaxActivationRowsPerProfile + 1 {
		result := repository.RecordActivation(ctx, ActivationRecord{
			ProfileID: retentionProfileA, TabID: int64(index), Domain: "example.test",
			ActivatedAt: now, Source: domain.ActivationKeyboardEnter,
		})
		if result.Status != StatusReady || result.ErrorClass != "" {
			t.Fatalf("record activation %d: %+v", index, result)
		}
	}
	var count, minTabID, maxTabID int
	if err := db.QueryRow(`SELECT count(*), min(tab_id), max(tab_id) FROM activation_metadata WHERE profile_id = ?`, retentionProfileA).Scan(&count, &minTabID, &maxTabID); err != nil {
		t.Fatal(err)
	}
	if count != MaxActivationRowsPerProfile || minTabID != 1 || maxTabID != MaxActivationRowsPerProfile {
		t.Fatalf("retained activations = count:%d tab range:%d..%d", count, minTabID, maxTabID)
	}

	configurationB := DefaultConfiguration(domain.ProfileID(retentionProfileB))
	configurationB.RankingModelVersion = versions.RankingModelVersion
	if result := repository.SaveConfiguration(ctx, configurationB, versions); result.Status != StatusReady {
		t.Fatalf("save profile B default configuration: %+v", result)
	}
	recordB := ActivationRecord{ProfileID: retentionProfileB, TabID: 999, Domain: "example.test", ActivatedAt: now, Source: domain.ActivationSurfaceSelect}
	if result := repository.RecordActivation(ctx, recordB); result.Status != StatusReady {
		t.Fatalf("activation without opt-in result = %+v", result)
	}
	var profileBCount int
	if err := db.QueryRow(`SELECT count(*) FROM activation_metadata WHERE profile_id = ?`, retentionProfileB).Scan(&profileBCount); err != nil {
		t.Fatal(err)
	}
	if profileBCount != 0 {
		t.Fatalf("activation persisted without preference opt-in: %d rows", profileBCount)
	}

	configurationB.PersistActivationRecency = true
	if result := repository.SaveConfiguration(ctx, configurationB, versions); result.Status != StatusReady {
		t.Fatalf("enable profile B activation persistence: %+v", result)
	}
	if result := repository.RecordActivation(ctx, recordB); result.Status != StatusReady {
		t.Fatalf("profile B activation result = %+v", result)
	}
	if err := db.QueryRow(`SELECT count(*) FROM activation_metadata WHERE profile_id = ?`, retentionProfileB).Scan(&profileBCount); err != nil {
		t.Fatal(err)
	}
	if profileBCount != 1 {
		t.Fatalf("profile A retention affected profile B: %d rows", profileBCount)
	}
}

func TestPruneRetentionUsesAgeAndLogicalDiagnosticPayloadBudget(t *testing.T) {
	db, repository := newSchemaRepository(t)
	ctx := context.Background()
	now := time.Unix(2_000_000_000, 0)
	activationBoundary := now.Unix() - ActivationRetentionSeconds
	for _, activatedAt := range []int64{activationBoundary - 1, activationBoundary} {
		if _, err := db.Exec(`INSERT INTO activation_metadata(profile_id, context_kind, tab_id, domain, activated_at, source) VALUES (?, 'normal', ?, 'example.test', ?, 'keyboard_enter')`, retentionProfileA, activatedAt, activatedAt); err != nil {
			t.Fatal(err)
		}
	}

	diagnosticBoundary := now.UnixMilli() - DiagnosticsRetentionMillis
	payload := fmt.Sprintf("%0*d", MaxDiagnosticRowBytes, 0)
	if _, err := db.Exec(`INSERT INTO diagnostics(occurred_at, event_kind, error_class, retryable, duration_ms, projection_revision, protocol_version, ranking_model_version, schema_version, payload_bytes, redacted_payload) VALUES (?, 'storage.test', 'storage_failure', 0, 0, 0, ?, '', 1, 1, 'x')`, diagnosticBoundary-1, protocol.CurrentVersion); err != nil {
		t.Fatal(err)
	}
	for index := range MaxDiagnosticsPayloadBytes/MaxDiagnosticRowBytes + 1 {
		occurredAt := now.UnixMilli() - int64(MaxDiagnosticsPayloadBytes/MaxDiagnosticRowBytes+1-index)
		if _, err := db.Exec(`INSERT INTO diagnostics(occurred_at, event_kind, error_class, retryable, duration_ms, projection_revision, protocol_version, ranking_model_version, schema_version, payload_bytes, redacted_payload) VALUES (?, 'storage.test', 'storage_failure', 0, 0, 0, ?, '', 1, ?, ?)`, occurredAt, protocol.CurrentVersion, len(payload), payload); err != nil {
			t.Fatal(err)
		}
	}

	result := repository.PruneRetention(ctx, now)
	if result.Status != StatusReady || result.ErrorClass != "" {
		t.Fatalf("retention result = %+v", result)
	}
	var activationCount, retainedActivationAt int
	if err := db.QueryRow(`SELECT count(*), min(activated_at) FROM activation_metadata WHERE profile_id = ?`, retentionProfileA).Scan(&activationCount, &retainedActivationAt); err != nil {
		t.Fatal(err)
	}
	if activationCount != 1 || int64(retainedActivationAt) != activationBoundary {
		t.Fatalf("activation age boundary count:%d timestamp:%d want:%d", activationCount, retainedActivationAt, activationBoundary)
	}
	var diagnosticCount, payloadBytes, oldestDiagnosticAt int64
	if err := db.QueryRow(`SELECT count(*), coalesce(sum(payload_bytes), 0), min(occurred_at) FROM diagnostics`).Scan(&diagnosticCount, &payloadBytes, &oldestDiagnosticAt); err != nil {
		t.Fatal(err)
	}
	wantDiagnosticCount := int64(MaxDiagnosticsPayloadBytes / MaxDiagnosticRowBytes)
	if diagnosticCount != wantDiagnosticCount || payloadBytes != MaxDiagnosticsPayloadBytes || oldestDiagnosticAt != now.UnixMilli()-wantDiagnosticCount {
		t.Fatalf("diagnostic retention count:%d bytes:%d oldest:%d want count:%d bytes:%d oldest:%d", diagnosticCount, payloadBytes, oldestDiagnosticAt, wantDiagnosticCount, MaxDiagnosticsPayloadBytes, now.UnixMilli()-wantDiagnosticCount)
	}
}

func TestAppendDiagnosticCountsUTF8PayloadBytes(t *testing.T) {
	db, repository := newSchemaRepository(t)
	result := repository.AppendDiagnostic(context.Background(), DiagnosticRecord{
		OccurredAt: time.Now().UnixMilli(), EventKind: "storage.test", ErrorClass: ErrorStorageFailure,
		ProtocolVersion: protocol.CurrentVersion, SchemaVersion: 1, RedactedPayload: "é",
	})
	if result.Status != StatusReady || result.ErrorClass != "" {
		t.Fatalf("append diagnostic result = %+v", result)
	}
	var payloadBytes int
	if err := db.QueryRow(`SELECT payload_bytes FROM diagnostics`).Scan(&payloadBytes); err != nil {
		t.Fatal(err)
	}
	if payloadBytes != 2 {
		t.Fatalf("stored UTF-8 payload byte count = %d, want 2", payloadBytes)
	}
}
