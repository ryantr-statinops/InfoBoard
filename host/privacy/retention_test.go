package privacy

import (
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

func TestRetentionPolicyMatchesIP08AndProfileOwnership(t *testing.T) {
	if storage.MaxActivationRowsPerProfile != 500 {
		t.Fatalf("activation rows = %d", storage.MaxActivationRowsPerProfile)
	}
	if storage.ActivationRetentionSeconds != 30*24*60*60 {
		t.Fatalf("activation age = %d", storage.ActivationRetentionSeconds)
	}
	if storage.DiagnosticsRetentionMillis != 7*24*60*60*1000 {
		t.Fatalf("diagnostics age = %d", storage.DiagnosticsRetentionMillis)
	}
	// IP-08 owns the storage cap and defines it as 10 MiB. The IP-17 retention
	// audit deliberately applies the stricter 10,000,000-byte contract cap.
	if storage.MaxDiagnosticsPayloadBytes != 10*1024*1024 {
		t.Fatalf("diagnostics bytes = %d", storage.MaxDiagnosticsPayloadBytes)
	}
	if ShouldPersistActivation(domain.ContextPrivate) {
		t.Fatal("private activation retention must be disabled")
	}
	if PrivateDiagnosticAllowed() {
		t.Fatal("private diagnostics must be disabled")
	}
	owned := map[string]bool{}
	for _, table := range storage.OwnedTableNames() {
		owned[table] = true
	}
	for _, table := range []string{"configuration", "installation_state", "activation_metadata", "diagnostics", "schema_meta", "schema_migrations"} {
		if !owned[table] {
			t.Fatalf("owned table missing: %s", table)
		}
	}
	if owned["browser_state"] {
		t.Fatal("unrelated browser data was classified as owned")
	}
}
