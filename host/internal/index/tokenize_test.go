package index

import (
	"reflect"
	"testing"
)

func TestTokenizeSplitsOnWhitespaceAndPunctuation(t *testing.T) {
	got := tokenize("alpha-beta, gamma.delta")
	want := []string{"alpha", "beta", "gamma", "delta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens: %v", got)
	}
}

func TestTokenizeSearchMatchesIndexTokenization(t *testing.T) {
	got := TokenizeSearch("alpha-beta, gamma.delta")
	want := []string{"alpha", "beta", "gamma", "delta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens: %v", got)
	}
}

func TestParseURLSegments(t *testing.T) {
	segments, err := ParseURLSegments("https://Docs.Example.test/guide/start")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !reflect.DeepEqual(segments.Host, []string{"docs", "example", "test"}) {
		t.Fatalf("host: %v", segments.Host)
	}
	if !reflect.DeepEqual(segments.Path, []string{"guide", "start"}) {
		t.Fatalf("path: %v", segments.Path)
	}
}

func TestStateLabels(t *testing.T) {
	labels := StateLabels(true, false)
	if len(labels) != 1 || labels[0].Field != FieldPinned || labels[0].Search != "pinned" {
		t.Fatalf("labels: %+v", labels)
	}
}
