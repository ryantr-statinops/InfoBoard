// Package phase10 exercises the IP-10 lexical index through its public seams.
package phase10

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
	"github.com/ryantr-statinops/InfoBoard/host/internal/index"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

// TestLexicalQuerySurvivesPersistenceFailure proves durable persistence is
// optional for the lexical hot path: a storage open failure leaves the
// in-memory index and prepared query fully usable.
func TestLexicalQuerySurvivesPersistenceFailure(t *testing.T) {
	ctx := context.Background()

	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, []byte("occupied"), 0o600); err != nil {
		t.Fatalf("prepare blocked path: %v", err)
	}
	repository, result := storage.Open(ctx, blocked)
	if repository != nil {
		_ = repository.Close()
	}
	if result.Status == storage.StatusReady {
		t.Fatalf("storage unexpectedly ready at %s", blocked)
	}

	publisher := index.NewPublisher()
	records := []domain.EligibleTabRecord{{
		ProfileID:    "profile",
		ContextKind:  domain.ContextNormal,
		TabIdentity:  domain.TabIdentity{ProfileID: "profile", ContextKind: domain.ContextNormal, TabID: 1},
		TitleSearch:  "Example Page",
		DomainSearch: "example.test",
		Eligible:     true,
	}}
	if err := publisher.Publish(ctx, "profile", domain.ContextNormal, "epoch", 1, records); err != nil {
		t.Fatalf("publish with storage unavailable: %v", err)
	}
	if status := publisher.QueryGate(1); status != index.QueryReady {
		t.Fatalf("lexical index not ready: %s", status)
	}
	state := publisher.State()
	if state == nil || len(state.Postings["example"]) != 1 {
		t.Fatalf("lexical postings unavailable: %+v", state)
	}

	prepared, err := index.PrepareQuery("Example")
	if err != nil {
		t.Fatalf("prepare query: %v", err)
	}
	if prepared.Empty || len(prepared.Tokens) != 1 || prepared.Tokens[0] != "example" {
		t.Fatalf("unexpected prepared query: %+v", prepared)
	}
}
