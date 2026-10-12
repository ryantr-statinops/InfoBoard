package projection

// RequireSnapshot quarantines the committed lineage after reconnect or uncertainty.
// The last complete records remain available for inspection, but the partition is
// not queryable until a new authoritative snapshot establishes a fresh epoch.
func (state *State) RequireSnapshot() {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.committed == nil || state.committed.Status == StatusResyncRequired {
		return
	}
	current := *state.committed
	current.Status = StatusResyncRequired
	if current.Counters.ResyncRequests < ^uint64(0) {
		current.Counters.ResyncRequests++
	}
	state.committed = &current
}

// RequireSnapshot fences the reconciler at a host-session recovery boundary.
func (reconciler *Reconciler) RequireSnapshot() {
	reconciler.state.RequireSnapshot()
}
