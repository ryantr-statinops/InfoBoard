// Package recovery owns reconnect/resync/full-rebuild coordination (IP-09).
package recovery

import "context"

// Rebuilder coordinates reconnect and full rebuild without reusing stale state.
type Rebuilder struct{}

func NewRebuilder() *Rebuilder { return &Rebuilder{} }

func (r *Rebuilder) InvalidateSession(ctx context.Context) error { return nil }
func (r *Rebuilder) ForceRebuild(ctx context.Context) error      { return nil }
