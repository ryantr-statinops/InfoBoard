package sync

import (
	"context"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// IndexPublisher is the IP-10 seam. IP-09 stages a validated snapshot/delta and
// publishes the matching index revision only after projection commit succeeds.
type IndexPublisher interface {
	Publish(ctx context.Context, profileID string, kind domain.ContextKind, epoch string, revision uint64, records []domain.EligibleTabRecord) error
	Ready() (revision uint64, ok bool)
	Rebuilding() bool
}

// NoopPublisher is the default seam until IP-10 supplies the in-memory index.
type NoopPublisher struct {
	revision   uint64
	rebuilding bool
}

func NewNoopPublisher() *NoopPublisher { return &NoopPublisher{} }

func (p *NoopPublisher) Publish(_ context.Context, _ string, _ domain.ContextKind, _ string, revision uint64, _ []domain.EligibleTabRecord) error {
	p.revision = revision
	p.rebuilding = false
	return nil
}

func (p *NoopPublisher) Ready() (uint64, bool) { return p.revision, p.revision != 0 && !p.rebuilding }
func (p *NoopPublisher) Rebuilding() bool      { return p.rebuilding }
