package index

import "testing"

func TestPrepareQueryTreatsEmptyAndWhitespaceEqually(t *testing.T) {
	for _, raw := range []string{"", "   ", "\t\n"} {
		prepared, err := PrepareQuery(raw)
		if err != nil {
			t.Fatalf("prepare %q: %v", raw, err)
		}
		if !prepared.Empty || len(prepared.Tokens) != 0 {
			t.Fatalf("not empty: %+v", prepared)
		}
	}
}

func TestPrepareQueryNormalizesTokens(t *testing.T) {
	prepared, err := PrepareQuery("  CAFÉ  Menu ")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if len(prepared.Tokens) != 2 || prepared.Tokens[0] != "café" || prepared.Tokens[1] != "menu" {
		t.Fatalf("tokens: %v", prepared.Tokens)
	}
}

func TestPrepareQueryRejectsMalformedAndOverlong(t *testing.T) {
	if _, err := PrepareQuery(string([]byte{0xff})); err == nil {
		t.Fatalf("invalid utf8 accepted")
	}
	long := make([]rune, MaxQueryScalars+1)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := PrepareQuery(string(long)); err == nil || err.Code != ErrQueryTooLong {
		t.Fatalf("overlong query accepted: %+v", err)
	}
}

func TestPrepareQueryPhraseBoundaries(t *testing.T) {
	prepared, err := PrepareQuery(`alpha "beta gamma"`)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if len(prepared.Phrases) != 2 {
		t.Fatalf("phrases: %+v", prepared.Phrases)
	}
}
