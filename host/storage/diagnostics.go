package storage

import (
	"strings"
	"unicode/utf8"
)

const (
	// MaxDiagnosticFieldBytes bounds every free-text diagnostic field. It
	// matches MaxStoredVersionBytes so version and identity labels share one
	// budget with the protocol layer's opaque values.
	MaxDiagnosticFieldBytes = 128
)

// RedactDiagnosticPayload replaces every rune outside the safe set with '#',
// so titles, full URLs, queries, cookies, tokens, SQL, and file paths cannot
// survive in stored diagnostics. The output is deterministic for a given input
// and never larger in bytes than the input: every replaced rune becomes one
// ASCII byte. Invalid UTF-8 redacts to empty so undecodable bytes are never
// stored. Callers must still respect MaxDiagnosticRowBytes on the result.
func RedactDiagnosticPayload(input string) string {
	if input == "" {
		return ""
	}
	if !utf8.ValidString(input) {
		return ""
	}
	var redacted strings.Builder
	redacted.Grow(len(input))
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_', r == '-', r == '.':
			redacted.WriteRune(r)
		default:
			redacted.WriteByte('#')
		}
	}
	return redacted.String()
}

// ValidDiagnosticEventKind accepts only lowercase dotted event names such as
// "storage.open_failed". Arbitrary titles, URLs, queries, or sentences never
// match: they contain uppercase letters, slashes, spaces, or query syntax.
func ValidDiagnosticEventKind(kind string) bool {
	return validDiagnosticToken(kind, true)
}

// ValidDiagnosticErrorClass accepts only closed lowercase error classes such
// as "storage_unavailable". Callers must map driver errors, SQL text, and
// browser data to an ErrorClass before recording; raw text never matches.
func ValidDiagnosticErrorClass(class string) bool {
	return validDiagnosticToken(class, false)
}

// ValidDiagnosticVersionLabel accepts only opaque version labels such as
// "rank-1" or "1.4.0+build.2". Paths, SQL fragments, and sentences contain
// characters outside the label set and are rejected.
func ValidDiagnosticVersionLabel(version string) bool {
	if version == "" || len(version) > MaxDiagnosticFieldBytes || !utf8.ValidString(version) {
		return false
	}
	for index := 0; index < len(version); index++ {
		char := version[index]
		switch {
		case char >= 'a' && char <= 'z',
			char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9',
			char == '_', char == '-', char == '.', char == '+':
		default:
			return false
		}
	}
	return version[0] != '.' && version[0] != '-' && version[0] != '+' && version[len(version)-1] != '.'
}

func validDiagnosticToken(token string, dotted bool) bool {
	if token == "" || len(token) > MaxDiagnosticFieldBytes || !utf8.ValidString(token) {
		return false
	}
	segments := 1
	segmentLength := 0
	previousDot := false
	for index := 0; index < len(token); index++ {
		char := token[index]
		if char == '.' {
			if !dotted || segmentLength == 0 || previousDot {
				return false
			}
			segments++
			segmentLength = 0
			previousDot = true
			continue
		}
		previousDot = false
		switch {
		case char >= 'a' && char <= 'z',
			char >= '0' && char <= '9',
			char == '_':
			segmentLength++
		default:
			return false
		}
		if index == 0 && (char < 'a' || char > 'z') {
			return false
		}
	}
	if previousDot || segmentLength == 0 {
		return false
	}
	return !dotted || segments >= 2
}

// ValidateDiagnosticRecord reports the closed error class for a record that
// violates the redaction contract, or "" when the record is storable. A
// non-empty RedactedPayload must already be redacted: values that still carry
// excluded content fail validation instead of being silently rewritten.
func ValidateDiagnosticRecord(record DiagnosticRecord) ErrorClass {
	if !ValidDiagnosticEventKind(record.EventKind) {
		return ErrorInvalidConfiguration
	}
	if !ValidDiagnosticErrorClass(string(record.ErrorClass)) {
		return ErrorInvalidConfiguration
	}
	if !ValidDiagnosticVersionLabel(record.RankingModelVersion) && record.RankingModelVersion != "" {
		return ErrorInvalidConfiguration
	}
	if !utf8.ValidString(record.RedactedPayload) || len(record.RedactedPayload) > MaxDiagnosticRowBytes {
		return ErrorInvalidConfiguration
	}
	if record.RedactedPayload != RedactDiagnosticPayload(record.RedactedPayload) {
		return ErrorInvalidConfiguration
	}
	return ""
}
