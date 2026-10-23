package index

import "testing"

func TestValidateRejectsInvalidUTF8(t *testing.T) {
	if err := validate(string([]byte{0xff, 0xfe})); err == nil || err.Code != ErrInvalidUTF8 {
		t.Fatalf("invalid utf8 accepted: %+v", err)
	}
}

func TestValidateRejectsControlCharacters(t *testing.T) {
	if err := validate("safe\x00value"); err == nil || err.Code != ErrControlChar {
		t.Fatalf("control character accepted: %+v", err)
	}
	if err := validate("tab\tnewline\ncarriage\r"); err != nil {
		t.Fatalf("policy whitespace rejected: %v", err)
	}
}

func TestNormalizeFormFoldsCaseAndCompatibility(t *testing.T) {
	if got := normalizeForm("CAFÉ"); got != "café" {
		t.Fatalf("case fold: %q", got)
	}
	if got := normalizeForm("ﬁle"); got != "file" {
		t.Fatalf("nfkc: %q", got)
	}
}

func TestCollapseWhitespace(t *testing.T) {
	if got := collapseWhitespace("  a \t b\n\nc  "); got != "a b c" {
		t.Fatalf("whitespace: %q", got)
	}
}

func TestTruncateScalarsNeverSplitsRunes(t *testing.T) {
	got := truncateScalars("héllo", 2)
	if got != "hé" {
		t.Fatalf("truncate: %q", got)
	}
	if got := truncateScalars("abc", 0); got != "" {
		t.Fatalf("zero bound: %q", got)
	}
}
