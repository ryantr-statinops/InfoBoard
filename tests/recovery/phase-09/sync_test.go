package phase09

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/projection"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	syncer "github.com/ryantr-statinops/InfoBoard/host/internal/sync"
)

const fixtureDir = "../../../fixtures/recovery/phase-09"

type fixture struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Expected struct {
		Accepted bool   `json:"accepted"`
		State    string `json:"state"`
		Error    string `json:"error"`
	} `json:"expected"`
}

func loadFixtures(t *testing.T) []fixture {
	t.Helper()
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
	fixtures := make([]fixture, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		body, err := os.ReadFile(filepath.Join(fixtureDir, entry.Name()))
		if err != nil {
			t.Fatalf("read fixture %s: %v", entry.Name(), err)
		}
		var parsed fixture
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("parse fixture %s: %v", entry.Name(), err)
		}
		fixtures = append(fixtures, parsed)
	}
	return fixtures
}

func newOrchestrator(t *testing.T) *syncer.Orchestrator {
	t.Helper()
	partition := projection.Partition{
		ProfileID:   domain.ProfileID("123e4567-e89b-42d3-a456-426614174000"),
		ContextKind: domain.ContextNormal,
	}
	orchestrator, err := syncer.NewOrchestrator(partition, syncer.NewNoopPublisher())
	if err != nil {
		t.Fatalf("new orchestrator: %v", err)
	}
	return orchestrator
}

func TestFixturesAreComplete(t *testing.T) {
	fixtures := loadFixtures(t)
	if len(fixtures) != 8 {
		t.Fatalf("expected 8 fixtures, got %d", len(fixtures))
	}
	for _, item := range fixtures {
		if item.ID == "" || item.Name == "" {
			t.Fatalf("incomplete fixture: %+v", item)
		}
	}
}

func TestHelloToReadyThenRevisionChange(t *testing.T) {
	orchestrator := newOrchestrator(t)
	ctx := context.Background()
	outcome, err := orchestrator.ApplySnapshot(ctx, protocol.SyncRequest{
		Context:  protocol.RequestContext{ProfileID: "123e4567-e89b-42d3-a456-426614174000", ContextKind: domain.ContextNormal},
		Epoch:    "123e4567-e89b-42d3-a456-426614174001",
		Revision: 1,
		Sequence: 1,
		Snapshot: &protocol.Snapshot{},
	})
	if err != nil || !outcome.Accepted {
		t.Fatalf("snapshot: outcome=%+v err=%v", outcome, err)
	}
	if allowed, _ := orchestrator.CheckQuery(1); !allowed {
		t.Fatalf("current revision refused")
	}
	if allowed, code := orchestrator.CheckQuery(2); allowed || code != protocol.CodeRevisionMismatch {
		t.Fatalf("unknown revision allowed: %s", code)
	}
}

func TestDegradedPersistenceKeepsSyncQueryable(t *testing.T) {
	orchestrator := newOrchestrator(t)
	orchestrator.SetPersistenceDegraded(true)
	outcome, err := orchestrator.ApplySnapshot(context.Background(), protocol.SyncRequest{
		Context:  protocol.RequestContext{ProfileID: "123e4567-e89b-42d3-a456-426614174000", ContextKind: domain.ContextNormal},
		Epoch:    "123e4567-e89b-42d3-a456-426614174001",
		Revision: 1,
		Sequence: 1,
		Snapshot: &protocol.Snapshot{},
	})
	if err != nil || !outcome.Accepted {
		t.Fatalf("degraded snapshot: outcome=%+v err=%v", outcome, err)
	}
	health := orchestrator.PersistenceHealth()
	if !health.Degraded || !health.Queryable {
		t.Fatalf("unexpected health: %+v", health)
	}
}
