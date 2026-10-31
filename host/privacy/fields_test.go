package privacy

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func privacyFixture(t *testing.T, name string) []byte {
	t.Helper()
	root := os.Getenv("INFOBOARD_PRIVACY_FIXTURES")
	if root == "" {
		root = filepath.Join("..", "..", "fixtures", "privacy", "phase-17")
	}
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("read privacy fixture %s: %v", name, err)
	}
	return data
}

func TestProfileFieldClassMatrix(t *testing.T) {
	data := privacyFixture(t, "field-classes.json")
	catalog, err := LoadFieldCatalogJSON(data)
	if err != nil {
		t.Fatalf("load field catalog: %v", err)
	}
	if catalog.Count() < 50 {
		t.Fatalf("field catalog too small: %d", catalog.Count())
	}

	var fixture struct {
		Fields          []FieldPolicy `json:"fields"`
		ForbiddenFields []string      `json:"forbidden_fields"`
		Classes         []FieldClass  `json:"classes"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Classes) != len(FieldClasses) {
		t.Fatalf("class count = %d, want %d", len(fixture.Classes), len(FieldClasses))
	}
	for _, class := range fixture.Classes {
		found := false
		for _, expected := range FieldClasses {
			if class == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("unknown field class %q", class)
		}
	}
	for _, field := range fixture.ForbiddenFields {
		if !catalog.IsForbidden(field) {
			t.Fatalf("forbidden field %q was not marked forbidden", field)
		}
		if _, err := catalog.AssertClassified(field); !errors.Is(err, ErrForbiddenField) {
			t.Fatalf("assert forbidden %q = %v", field, err)
		}
	}
	if _, err := catalog.AssertClassified("unclassified_sentinel"); !errors.Is(err, ErrUnclassifiedField) {
		t.Fatalf("unclassified field error = %v", err)
	}
}

func TestProfileFieldCatalogRejectsMalformedEntries(t *testing.T) {
	cases := [][]FieldPolicy{
		nil,
		{{Field: "profile_id", FieldClass: FieldClassTransientSearch}},
		{{Field: "", FieldClass: FieldClassForbidden, Source: "x", Owner: "x", Boundary: "x", Storage: "x", Retention: "x", Display: "x", Redaction: "x"}},
		{{Field: "profile_id", FieldClass: "unknown", Source: "x", Owner: "x", Boundary: "x", Storage: "x", Retention: "x", Display: "x", Redaction: "x"}},
	}
	for index, policies := range cases {
		if _, err := NewFieldCatalog(policies); !errors.Is(err, ErrInvalidFieldCatalog) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}
