package lifecycle

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

func TestLifecycleStatusValid(t *testing.T) {
	for _, status := range []LifecycleStatus{StatusComplete, StatusDegraded, StatusRetryable, StatusIncomplete} {
		if !status.Valid() {
			t.Fatalf("status %q must be valid", status)
		}
	}
	for _, status := range []LifecycleStatus{"", "done", "retry", "secret"} {
		if status.Valid() {
			t.Fatalf("status %q must be invalid", status)
		}
	}
}

func TestLifecycleErrorClassesAreClosed(t *testing.T) {
	classes := []ErrorClass{
		ErrorNone,
		ErrorInvalidConfiguration,
		ErrorProfileMismatch,
		ErrorStorageUnavailable,
		ErrorStorageTimeout,
		ErrorStorageFailure,
		ErrorSchemaUpgradeRequired,
		ErrorDatabaseQuarantined,
		ErrorDatabaseCorrupt,
		ErrorSchemaIntegrity,
		ErrorQuarantineFailed,
		ErrorRecoveryFailed,
		ErrorUnsafeDatabasePath,
		ErrorCompatibilityMismatch,
		ErrorResetRuntimeUnavailable,
		ErrorProjectionResetFailed,
		ErrorProjectionIncomplete,
		ErrorCancelled,
		ErrorUninstallNotConfigured,
		ErrorSessionCloseFailed,
		ErrorCleanupManifestInvalid,
		ErrorCleanupCancelled,
		ErrorCleanupFailed,
		ErrorPathSafety,
		ErrorUnknown,
	}
	for _, class := range classes {
		if !class.Valid() {
			t.Fatalf("error class %q must be valid", class)
		}
	}
	for _, class := range []ErrorClass{"secret", "C:\\private\\db.sqlite", "https://example.test"} {
		if class.Valid() {
			t.Fatalf("error class %q must be invalid", class)
		}
	}
}

func TestErrorClassFromStorageIsClosed(t *testing.T) {
	cases := map[storage.ErrorClass]ErrorClass{
		storage.ErrorProfileMismatch:       ErrorProfileMismatch,
		storage.ErrorInvalidConfiguration:  ErrorInvalidConfiguration,
		storage.ErrorStorageUnavailable:    ErrorStorageUnavailable,
		storage.ErrorStorageTimeout:        ErrorStorageTimeout,
		storage.ErrorStorageFailure:        ErrorStorageFailure,
		storage.ErrorSchemaUpgradeRequired: ErrorSchemaUpgradeRequired,
		storage.ErrorDatabaseQuarantined:   ErrorDatabaseQuarantined,
		storage.ErrorDatabaseCorrupt:       ErrorDatabaseCorrupt,
		storage.ErrorSchemaIntegrity:       ErrorSchemaIntegrity,
		storage.ErrorQuarantineFailed:      ErrorQuarantineFailed,
		storage.ErrorRecoveryFailed:        ErrorRecoveryFailed,
		storage.ErrorUnsafeDatabasePath:    ErrorUnsafeDatabasePath,
		"":                                 ErrorNone,
		"caller text":                      ErrorUnknown,
	}
	for input, want := range cases {
		if got := errorClassFromStorage(input); got != want {
			t.Fatalf("storage class %q mapped to %q, want %q", input, got, want)
		}
	}
}

func TestLifecycleResultValidateRejectsEscapedValues(t *testing.T) {
	valid := LifecycleResult{
		Operation:          OperationReset,
		ProfileID:          profileA,
		Status:             StatusComplete,
		Persistence:        PersistenceReady,
		Durable:            true,
		ErrorClass:         ErrorNone,
		CompletedSteps:     []string{StepStorageReset, StepIndexCleared, StepSnapshotRequested},
		OwnedStateVerified: true,
		ProjectionRevision: 9,
		Configuration:      DefaultProfileConfig(profileA, 1, "rank-1"),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}
	cases := []LifecycleResult{
		{Operation: OperationReset, ProfileID: profileA, Status: "done", ErrorClass: ErrorNone},
		{Operation: OperationReset, ProfileID: profileA, Status: StatusComplete, ErrorClass: "secret"},
		{Operation: OperationReset, ProfileID: "not-a-profile", Status: StatusComplete, ErrorClass: ErrorNone},
		{Operation: OperationReset, ProfileID: profileA, Status: StatusComplete, ErrorClass: ErrorNone, CompletedSteps: []string{"C:\\secret"}},
	}
	for _, result := range cases {
		if err := result.Validate(); err == nil {
			t.Fatalf("invalid result passed: %+v", result)
		}
	}
}

func TestRedactedDiagnosticsExcludesConfigurationAndCallerText(t *testing.T) {
	result := LifecycleResult{
		Operation:          OperationUninstall,
		ProfileID:          profileA,
		Status:             StatusIncomplete,
		Retryable:          true,
		Persistence:        PersistenceUnavailable,
		Durable:            false,
		ErrorClass:         ErrorPathSafety,
		CompletedSteps:     []string{StepSessionClosed, StepStorageUninstall},
		OwnedStateVerified: false,
		ProjectionRevision: 0,
		Configuration:      DefaultProfileConfig(profileA, 1, "rank-1"),
	}
	diagnostics := result.RedactedDiagnostics()
	encoded, err := json.Marshal(diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"configuration", "title", "url", "query", "file://", "https://", "C:\\"} {
		if strings.Contains(strings.ToLower(text), strings.ToLower(forbidden)) {
			t.Fatalf("diagnostics contain forbidden content %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, `"error_class":"path_safety"`) || !strings.Contains(text, `"status":"incomplete"`) {
		t.Fatalf("diagnostics missing closed fields: %s", text)
	}
}
