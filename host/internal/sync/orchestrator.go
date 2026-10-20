// Package sync owns the host-side projection synchronization orchestrator (IP-09).
package sync

import "context"

// Orchestrator coordinates hello->snapshot->index->ready and revision-safe handoff.
type Orchestrator struct {
	// State fields are added as IP-09 tasks land.
}

func NewOrchestrator() *Orchestrator { return &Orchestrator{} }

// HandleHello validates protocol/profile/capabilities and returns ack status.
func (o *Orchestrator) HandleHello(ctx context.Context, profileID string) (string, error) {
	return "hello_ack", nil
}
