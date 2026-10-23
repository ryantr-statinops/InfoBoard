package index

import (
	"context"
	"sync"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// Publisher implements the IP-09 index publication seam. It publishes one
// immutable state per revision and reports rebuilding while a build is staged.
type Publisher struct {
	mu          sync.RWMutex
	state       *State
	rebuilding  bool
	lastFailure string
}

func NewPublisher() *Publisher { return &Publisher{} }

// Publish stages and commits one revision. The previous state remains readable
// until the new state is fully built.
func (p *Publisher) Publish(ctx context.Context, profileID string, kind domain.ContextKind, epoch string, revision uint64, records []domain.EligibleTabRecord) error {
	p.mu.Lock()
	p.rebuilding = true
	p.mu.Unlock()

	converted := make([]Record, 0, len(records))
	for _, record := range records {
		converted = append(converted, Record{
			TabID:       record.TabIdentity.TabID,
			ProfileID:   string(record.ProfileID),
			Revision:    revision,
			Eligible:    record.Eligible,
			Pinned:      record.Pinned,
			Active:      record.Active,
			Title:       record.TitleSearch,
			URL:         record.URLSearch,
			Domain:      record.DomainSearch,
			WindowLabel: deref(record.WindowLabelSearch),
			GroupLabel:  deref(record.GroupLabelSearch),
		})
	}
	staged, err := Stage(profileID, revision, converted)
	if err != nil {
		p.mu.Lock()
		p.rebuilding = false
		p.lastFailure = err.Error()
		p.mu.Unlock()
		return err
	}
	published := staged.Commit()

	p.mu.Lock()
	p.state = published
	p.rebuilding = false
	p.lastFailure = ""
	p.mu.Unlock()
	return nil
}

// Failure returns the bounded last build failure code, or an empty string.
func (p *Publisher) Failure() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastFailure
}

// Ready returns the published revision and whether it is queryable.
func (p *Publisher) Ready() (uint64, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.state == nil || p.rebuilding {
		return 0, false
	}
	return p.state.Revision, true
}

// Rebuilding reports whether a staged build is in progress.
func (p *Publisher) Rebuilding() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.rebuilding
}

// State returns the published immutable state.
func (p *Publisher) State() *State {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
