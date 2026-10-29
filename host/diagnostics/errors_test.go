package diagnostics

import (
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

func TestErrorStatusMatrix(t *testing.T) {
	tests := []struct {
		class      ErrorClass
		status     Status
		retryable  bool
		action     RecommendedAction
		maxAttempt int
		budget     int64
	}{
		{ErrorInvalidFrame, StatusRuntimeProtocolError, false, ActionReconnect, 0, 0},
		{ErrorProtocolMismatch, StatusRuntimeIncompatible, false, ActionUpdateHost, 0, 0},
		{ErrorProfileMismatch, StatusProfileRefreshRequired, true, ActionRefreshProfile, 1, 2_000},
		{ErrorPayloadLimit, StatusInputTooLarge, false, ActionShortenQuery, 0, 0},
		{ErrorSnapshotRequired, StatusProjectionRefresh, true, ActionRefreshProjection, 1, 2_000},
		{ErrorRevisionMismatch, StatusResultsRefresh, true, ActionRefreshProjection, 1, 2_000},
		{ErrorIndexRebuilding, StatusIndexRebuilding, true, ActionWait, 1, 2_000},
		{ErrorQueryTimeout, StatusQueryTimedOut, true, ActionRetry, 1, 1_000},
		{ErrorPersistenceDegraded, StatusPersistenceDegraded, true, ActionRepairPersistence, 3, 5_000},
		{ErrorHostShutdown, StatusRuntimeRestarting, true, ActionReconnect, 3, 5_000},
		{ErrorInternalFailure, StatusRuntimeRecoverableError, true, ActionRetry, 1, 1_000},
		{ErrorHostMissing, StatusRuntimeUnavailable, true, ActionReconnect, 3, 5_000},
		{ErrorPermissionLimited, StatusPermissionLimited, true, ActionRetry, 1, 1_000},
		{ErrorStorageQuarantined, StatusPersistenceQuarantined, true, ActionRepairPersistence, 1, 10_000},
		{ErrorSchemaUpgradeRequired, StatusPersistenceUpgrade, false, ActionUpdateHost, 0, 0},
	}
	for _, test := range tests {
		mapping, ok := Lookup(test.class)
		if !ok {
			t.Fatalf("%s has no mapping", test.class)
		}
		if err := mapping.Validate(); err != nil {
			t.Fatalf("%s invalid mapping: %v", test.class, err)
		}
		if mapping.Status != test.status || mapping.Retryable != test.retryable ||
			mapping.Action != test.action || mapping.Policy.MaxAttempts != test.maxAttempt ||
			mapping.Policy.TotalBudgetMillis != test.budget {
			t.Fatalf("%s mapping = %+v, want status=%s retryable=%t action=%s attempts=%d budget=%d",
				test.class, mapping, test.status, test.retryable, test.action, test.maxAttempt, test.budget)
		}
		if mapping.Retryable && mapping.ExhaustedAction == ActionNone {
			t.Fatalf("%s retryable mapping has no exhausted action", test.class)
		}
	}
}

func TestAllErrorClassesMapAndProtocolLiterals(t *testing.T) {
	for _, class := range AllErrorClasses() {
		if _, ok := Lookup(class); !ok {
			t.Fatalf("class %s missing", class)
		}
	}
	protocolCodes := []protocol.ErrorCode{
		protocol.CodeInvalidFrame, protocol.CodeProtocolMismatch,
		protocol.CodeProfileMismatch, protocol.CodePayloadLimit,
		protocol.CodeSnapshotRequired, protocol.CodeRevisionMismatch,
		protocol.CodeIndexRebuilding, protocol.CodeQueryTimeout,
		protocol.CodePersistenceDegraded, protocol.CodeHostShutdown,
		protocol.CodeInternalFailure,
	}
	for _, code := range protocolCodes {
		if _, ok := MapProtocolCode(code); !ok {
			t.Fatalf("protocol code %s missing", code)
		}
	}
}

func TestRetryPolicyBounds(t *testing.T) {
	mapping, _ := Lookup(ErrorHostMissing)
	if mapping.Policy.MaxAttempts != 3 || len(mapping.Policy.BackoffMillis) != 3 {
		t.Fatalf("host missing policy = %+v", mapping.Policy)
	}
	if mapping.Policy.BackoffMillis[0] != 100 || mapping.Policy.BackoffMillis[1] != 500 || mapping.Policy.BackoffMillis[2] != 1_000 {
		t.Fatalf("backoff = %v", mapping.Policy.BackoffMillis)
	}
	if mapping.Policy.TotalBudgetMillis != 5_000 {
		t.Fatalf("total budget = %d", mapping.Policy.TotalBudgetMillis)
	}

	invalid := mapping
	invalid.Policy.BackoffMillis = []int64{100, 500, 5_000}
	if invalid.Validate() == nil {
		t.Fatal("backoff above budget accepted")
	}
	invalid = mapping
	invalid.Retryable = false
	if invalid.Validate() == nil {
		t.Fatal("retryability mismatch accepted")
	}
	if _, ok := Lookup(ErrorClass("NOT_A_REAL_CLASS")); ok {
		t.Fatal("unknown class mapped")
	}
	if Map(ErrorClass("NOT_A_REAL_CLASS")).Status != StatusRuntimeRecoverableError {
		t.Fatal("unknown class did not fall back to internal failure")
	}
}
