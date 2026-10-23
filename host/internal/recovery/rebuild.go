package recovery

import (
	"context"
	"sync"
)

// Rebuilder coordinates reconnect and full rebuild without reusing stale state.
// Every session gets a generation; callbacks from an older generation can never
// publish into the current projection.
type Rebuilder struct {
	mu         sync.Mutex
	generation uint64
	rebuilding bool
}

func NewRebuilder() *Rebuilder { return &Rebuilder{generation: 1} }

// Generation returns the current session generation.
func (r *Rebuilder) Generation() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.generation
}

// InvalidateSession retires the current generation on disconnect. Any callback
// holding an older generation must be discarded.
func (r *Rebuilder) InvalidateSession(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.generation++
	r.rebuilding = true
	return nil
}

// IsCurrent reports whether a callback still belongs to the live generation.
func (r *Rebuilder) IsCurrent(generation uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return generation == r.generation && !r.rebuilding
}

// ForceRebuild re-arms the session and requires a fresh hello plus snapshot.
func (r *Rebuilder) ForceRebuild(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.generation++
	r.rebuilding = false
	return nil
}

// Rebuilding reports whether a fresh snapshot is still required.
func (r *Rebuilder) Rebuilding() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rebuilding
}
