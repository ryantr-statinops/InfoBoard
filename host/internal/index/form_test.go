package index

import "testing"

func TestBuildFormSeparatesDisplayAndSearch(t *testing.T) {
	form, err := BuildForm(FieldTitle, "  CAFÉ  Menu ", MaxTitleScalars)
	if err != nil {
		t.Fatalf("build form: %v", err)
	}
	if form.Display != "CAFÉ Menu" {
		t.Fatalf("display: %q", form.Display)
	}
	if form.Search != "café menu" {
		t.Fatalf("search: %q", form.Search)
	}
}

func TestBuildFormRejectsInvalidInput(t *testing.T) {
	if _, err := BuildForm(FieldTitle, string([]byte{0xff}), MaxTitleScalars); err == nil {
		t.Fatalf("invalid utf8 accepted")
	}
}

func TestProvenanceAppendsInOrder(t *testing.T) {
	var provenance Provenance
	provenance.Append(TokenRef{Field: FieldTitle, Position: 0, Source: "title"})
	provenance.Append(TokenRef{Field: FieldDomain, Position: 0, Source: "domain"})
	if len(provenance.Refs) != 2 || provenance.Refs[1].Field != FieldDomain {
		t.Fatalf("unexpected provenance: %+v", provenance.Refs)
	}
}
