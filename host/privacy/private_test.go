package privacy

import (
	"errors"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

func TestPrivateDurableWriteNeverReachesStore(t *testing.T) {
	calls := 0
	guard := NewPrivateWriteGuard(DurableWriteFunc(func(Scope, string) error {
		calls++
		return nil
	}))
	normal := scopeA(t)
	if err := guard.Write(normal, "normal"); err != nil {
		t.Fatalf("normal write: %v", err)
	}
	privateScope, err := ParseScope("chrome", scopeProfileA, "private", "nonce-1")
	if err != nil {
		t.Fatal(err)
	}
	var protocolErr *protocol.Error
	if err := guard.Write(privateScope, "private"); !errors.As(err, &protocolErr) || protocolErr.Code != protocol.CodeProfileMismatch {
		t.Fatalf("private write error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("store calls = %d, want 1", calls)
	}
}

func TestPrivateActivationAndDiagnosticsAreNotPersisted(t *testing.T) {
	if ShouldPersistActivation(domain.ContextPrivate) {
		t.Fatal("private activation would be persisted")
	}
	if !ShouldPersistActivation(domain.ContextNormal) {
		t.Fatal("normal activation must remain eligible for persistence")
	}
	if PrivateDiagnosticAllowed() {
		t.Fatal("private diagnostics must never be allowed")
	}
}
