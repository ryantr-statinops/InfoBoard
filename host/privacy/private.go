package privacy

import (
	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// DurableWriter is the IP-08 repository seam used by the private-write guard.
// The guard runs before the writer so a private context cannot create a row even
// when the caller supplies a normal-looking profile ID.
type DurableWriter interface {
	Write(scope Scope, payload string) error
}

// DurableWriteFunc adapts a function to DurableWriter for tests and small seams.
type DurableWriteFunc func(Scope, string) error

func (write DurableWriteFunc) Write(scope Scope, payload string) error {
	return write(scope, payload)
}

// PrivateWriteGuard rejects private durable writes before the underlying store
// is called.
type PrivateWriteGuard struct {
	writer DurableWriter
}

func NewPrivateWriteGuard(writer DurableWriter) *PrivateWriteGuard {
	return &PrivateWriteGuard{writer: writer}
}

func (guard *PrivateWriteGuard) Write(scope Scope, payload string) error {
	if failure := RejectPrivateDurableWrite(scope); failure != nil {
		return failure
	}
	if guard == nil || guard.writer == nil {
		return nil
	}
	return guard.writer.Write(scope, payload)
}

// ShouldPersistActivation is false for every private context. Activation recency
// is a normal-context feature; private activation outcomes stay in memory only.
func ShouldPersistActivation(contextKind domain.ContextKind) bool {
	return contextKind == domain.ContextNormal
}

// PrivateDiagnosticAllowed is always false: private identifiers and payloads are
// never representable in diagnostics.
func PrivateDiagnosticAllowed() bool {
	return false
}
