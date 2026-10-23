package index

import (
	"context"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

func TestQueryGateRefusesWhileRebuilding(t *testing.T) {
	publisher := NewPublisher()
	if status := publisher.QueryGate(1); status != QueryUnknown {
		t.Fatalf("unknown revision: %s", status)
	}
	publisher.BeginRebuild()
	if status := publisher.QueryGate(1); status != QueryRebuilding {
		t.Fatalf("rebuilding: %s", status)
	}
	publisher.AbortRebuild()
	if status := publisher.QueryGate(1); status != QueryUnknown {
		t.Fatalf("aborted: %s", status)
	}
}

func TestEmptySnapshotPublishesReady(t *testing.T) {
	publisher := NewPublisher()
	if err := publisher.Publish(context.Background(), "p", domain.ContextNormal, "e", 1, nil); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if status := publisher.QueryGate(1); status != QueryReady {
		t.Fatalf("empty snapshot not ready: %s", status)
	}
	if publisher.Failure() != "" {
		t.Fatalf("unexpected failure: %s", publisher.Failure())
	}
}

func TestFailedBuildKeepsPreviousState(t *testing.T) {
	publisher := NewPublisher()
	if err := publisher.Publish(context.Background(), "p", domain.ContextNormal, "e", 1, nil); err != nil {
		t.Fatalf("publish: %v", err)
	}
	records := []domain.EligibleTabRecord{
		{TabIdentity: domain.TabIdentity{TabID: 1}},
		{TabIdentity: domain.TabIdentity{TabID: 1}},
	}
	if err := publisher.Publish(context.Background(), "p", domain.ContextNormal, "e", 2, records); err == nil {
		t.Fatalf("duplicate build accepted")
	}
	if publisher.Failure() == "" {
		t.Fatalf("failure not recorded")
	}
	if status := publisher.QueryGate(1); status != QueryReady {
		t.Fatalf("previous state lost: %s", status)
	}
}
