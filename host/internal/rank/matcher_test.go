package rank

import (
	"reflect"
	"testing"
)

func TestBuildCandidateIndexUsesCanonicalFieldOrder(t *testing.T) {
	candidate := Candidate{
		TabID:    1,
		WindowID: 2,
		Pinned:   true,
		Active:   true,
		Fields: []FieldText{
			{Field: FieldWindow, Text: "window", Tokens: []string{"window"}},
			{Field: FieldURL, Text: "example.com/page", Tokens: []string{"example", "com", "page"}},
			{Field: FieldTitle, Text: "Example Page", Tokens: []string{"example", "page"}},
		},
	}
	index := buildCandidateIndex(candidate)
	got := make([]Field, 0, len(index.fields))
	for _, field := range index.fields {
		got = append(got, field.field)
	}
	want := []Field{FieldTitle, FieldURL, FieldWindow, FieldState}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("field order = %v, want %v", got, want)
	}
	if index.fields[3].text != "pinned active" {
		t.Fatalf("state text = %q", index.fields[3].text)
	}
	if !reflect.DeepEqual(buildCandidateIndex(candidate), index) {
		t.Fatal("repeated index build is not equal")
	}
}

func TestBuildCandidateIndexOmitsAbsentAndEmptyFields(t *testing.T) {
	index := buildCandidateIndex(Candidate{
		Fields: []FieldText{{Field: FieldTitle, Text: "", Tokens: nil}},
	})
	if len(index.fields) != 0 {
		t.Fatalf("fields = %v, want empty", index.fields)
	}
}
