package sync

import "context"

// Orchestrator coordinates hello->snapshot->index->ready and revision-safe handoff.
type Orchestrator struct {
	readyRevision int64
	readyProfile  string
}

func NewOrchestrator() *Orchestrator { return &Orchestrator{} }

func (o *Orchestrator) HandleHello(ctx context.Context, profileID string) (string, error) {
	return "hello_ack", nil
}

// T03: atomic snapshot publication. Validate before allocation; publish one
// complete generation; discard staging on failure (no new queryable revision).
func (o *Orchestrator) PublishSnapshot(ctx context.Context, profileID string, records int, build func() (int64, error)) (int64, error) {
	if records < 0 {
		return 0, errSnapshotRequired
	}
	rev, err := build()
	if err != nil {
		return 0, err
	}
	o.readyRevision = rev
	o.readyProfile = profileID
	return rev, nil
}

// T04: ordered delta processing. Requires expected predecessor; exact duplicate
// is idempotent ack; gap/conflict/stale/out-of-order -> no-op + resync_required.
func (o *Orchestrator) ProcessDelta(ctx context.Context, profileID string, expectedRev int64, newRev int64) (int64, error) {
	if profileID != o.readyProfile {
		return o.readyRevision, &syncError{code: "PROFILE_MISMATCH"}
	}
	if expectedRev != o.readyRevision {
		// duplicate ack if newRev == readyRevision
		if newRev == o.readyRevision {
			return o.readyRevision, nil
		}
		return o.readyRevision, &syncError{code: "resync_required"}
	}
	o.readyRevision = newRev
	return newRev, nil
}

var errSnapshotRequired = &syncError{code: "SNAPSHOT_REQUIRED"}

type syncError struct{ code string }

func (e *syncError) Error() string { return e.code }
