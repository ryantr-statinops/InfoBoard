package index

// QueryStatus is the bounded query gate result.
type QueryStatus string

const (
	QueryReady      QueryStatus = "READY"
	QueryRebuilding QueryStatus = "INDEX_REBUILDING"
	QueryUnknown    QueryStatus = "REVISION_MISMATCH"
)

// BeginRebuild marks the index as rebuilding before staged work starts.
func (p *Publisher) BeginRebuild() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rebuilding = true
}

// AbortRebuild returns to the previous published state after a failed build.
func (p *Publisher) AbortRebuild() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rebuilding = false
}

// QueryGate accepts only the current published revision.
func (p *Publisher) QueryGate(revision uint64) QueryStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.rebuilding {
		return QueryRebuilding
	}
	if p.state == nil || p.state.Revision != revision {
		return QueryUnknown
	}
	return QueryReady
}
