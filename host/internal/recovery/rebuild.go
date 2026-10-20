package recovery

import "context"

// Rebuilder coordinates reconnect and full rebuild without reusing stale state.
type Rebuilder struct {
	sessionValid bool
}

func NewRebuilder() *Rebuilder { return &Rebuilder{sessionValid: true} }

// T06: invalidate session-scoped work on disconnect; force new hello + snapshot.
func (r *Rebuilder) InvalidateSession(ctx context.Context) error {
	r.sessionValid = false
	return nil
}

func (r *Rebuilder) IsSessionValid() bool { return r.sessionValid }

func (r *Rebuilder) ForceRebuild(ctx context.Context) error {
	r.sessionValid = true
	return nil
}
