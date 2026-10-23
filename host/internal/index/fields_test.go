package index

import "testing"

func TestFieldTableIsClosed(t *testing.T) {
	for _, field := range []Field{FieldTitle, FieldURL, FieldDomain, FieldWindowLabel, FieldGroupLabel, FieldPinned, FieldActive} {
		if _, ok := LookupField(field); !ok {
			t.Fatalf("missing field: %s", field)
		}
	}
	if _, ok := LookupField(Field("unknown")); ok {
		t.Fatalf("unknown field accepted")
	}
}

func TestOptionalLabelDoesNotFabricateToken(t *testing.T) {
	spec, _ := LookupField(FieldGroupLabel)
	if !spec.Optional {
		t.Fatalf("group label should be optional")
	}
	form, err := BuildForm(FieldGroupLabel, "", spec.MaxScalars)
	if err != nil {
		t.Fatalf("empty optional label: %v", err)
	}
	if form.Search != "" {
		t.Fatalf("empty optional label fabricated search text: %q", form.Search)
	}
}

func TestEligibilityGatesPostings(t *testing.T) {
	if (Eligibility{Eligible: false}).AllowsPostings() {
		t.Fatalf("ineligible record allowed postings")
	}
	if !(Eligibility{Eligible: true}).AllowsPostings() {
		t.Fatalf("eligible record blocked")
	}
}
