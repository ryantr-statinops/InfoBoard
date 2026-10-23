package recovery

import (
	"context"
	"testing"
)

func TestInvalidateRetiresOldGeneration(t *testing.T) {
	rebuilder := NewRebuilder()
	old := rebuilder.Generation()
	if err := rebuilder.InvalidateSession(context.Background()); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	if rebuilder.IsCurrent(old) {
		t.Fatalf("old generation still current")
	}
	if !rebuilder.Rebuilding() {
		t.Fatalf("rebuilder should require a new snapshot")
	}
	if rebuilder.IsCurrent(rebuilder.Generation()) {
		t.Fatalf("session must stay invalid until rebuild completes")
	}
}

func TestForceRebuildReArmsSession(t *testing.T) {
	rebuilder := NewRebuilder()
	if err := rebuilder.InvalidateSession(context.Background()); err != nil {
		t.Fatalf("invalidate: %v", err)
	}
	if err := rebuilder.ForceRebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if rebuilder.Rebuilding() {
		t.Fatalf("rebuild flag still set")
	}
	if !rebuilder.IsCurrent(rebuilder.Generation()) {
		t.Fatalf("new generation should be current")
	}
}
