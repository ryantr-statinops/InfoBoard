package index

import (
	"context"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

func TestPublisherRetainsQueryMetadata(t *testing.T) {
	groupID := int64(7)
	windowLabel := "Work"
	groupLabel := "Research"
	source := domain.EligibleTabRecord{
		ProfileID:          domain.ProfileID("profile"),
		TabIdentity:        domain.TabIdentity{ProfileID: domain.ProfileID("profile"), TabID: 11},
		WindowID:           3,
		GroupID:            &groupID,
		TitleDisplay:       "Example Docs",
		TitleSearch:        "example docs",
		URLSearch:          "https://example.test/docs",
		URLDisplay:         "https://example.test/docs",
		DomainDisplay:      "example.test",
		DomainSearch:       "example.test",
		WindowLabelDisplay: &windowLabel,
		WindowLabelSearch:  &windowLabel,
		GroupLabelDisplay:  &groupLabel,
		GroupLabelSearch:   &groupLabel,
		Eligible:           true,
	}

	publisher := NewPublisher()
	if err := publisher.Publish(context.Background(), "profile", domain.ContextNormal, "epoch", 4, []domain.EligibleTabRecord{source}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	got, ok := publisher.State().Records[11]
	if !ok {
		t.Fatal("published record missing")
	}
	if got.WindowID != 3 || got.GroupID == nil || *got.GroupID != 7 {
		t.Fatalf("identity metadata = %+v", got)
	}
	if got.TitleDisplay != "Example Docs" || got.URLDisplay != "https://example.test/docs" || got.DomainDisplay != "example.test" {
		t.Fatalf("display metadata = %+v", got)
	}
	if got.WindowLabelDisplay == nil || *got.WindowLabelDisplay != "Work" || got.GroupLabelDisplay == nil || *got.GroupLabelDisplay != "Research" {
		t.Fatalf("label metadata = %+v", got)
	}

	groupID = 99
	windowLabel = "Changed"
	groupLabel = "Changed"
	if *got.GroupID != 7 || *got.WindowLabelDisplay != "Work" || *got.GroupLabelDisplay != "Research" {
		t.Fatalf("published state aliases caller pointers: %+v", got)
	}
}
