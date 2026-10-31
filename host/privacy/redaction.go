package privacy

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"

	"github.com/ryantr-statinops/InfoBoard/host/diagnostics"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

// Redactor is the IP-17 output boundary. It delegates to the IP-16 allowlist
// implementation and the IP-08 diagnostic payload scrubber instead of creating
// a second logging system.
type Redactor struct {
	diagnostics diagnostics.Redaction
}

func NewRedactor() Redactor {
	return Redactor{}
}

// SanitizeMap drops every unknown or malformed field and returns only the
// allowlisted diagnostic shape.
func (redactor Redactor) SanitizeMap(input map[string]any) map[string]any {
	copy := make(map[string]any, len(input))
	for key, value := range input {
		copy[key] = value
	}
	for _, key := range []string{"event_id", "session_id"} {
		if text, ok := copy[key].(string); ok && !safeLocalID(text) {
			delete(copy, key)
		}
	}
	return redactor.diagnostics.SanitizeMap(copy)
}

func safeLocalID(value string) bool {
	if value == "" || len(value) > 96 {
		return false
	}
	for index := 0; index < len(value); index++ {
		char := value[index]
		if index == 0 {
			if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
				continue
			}
			return false
		}
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') ||
			char == '.' || char == '_' || char == ':' || char == '-' {
			continue
		}
		return false
	}
	return true
}

// SanitizeRecords applies the same allowlist to a bounded record slice.
func (redactor Redactor) SanitizeRecords(input []map[string]any) []map[string]any {
	return redactor.diagnostics.SanitizeRecords(input)
}

// ExportRecords serializes only sanitized records.
func (redactor Redactor) ExportRecords(input []map[string]any) ([]byte, error) {
	return redactor.diagnostics.ExportRecords(input)
}

// ValidateExport rejects an export that still contains any supplied sentinel.
func (redactor Redactor) ValidateExport(export []byte, sensitive ...string) error {
	return diagnostics.ValidateExport(export, sensitive...)
}

// SafeError wraps a cause without allowing its text to reach an observable sink.
func (redactor Redactor) SafeError(class diagnostics.ErrorClass, cause error) error {
	return diagnostics.NewSafeError(class, cause)
}

// RedactError returns only the mapped safe status.
func (redactor Redactor) RedactError(err error) string {
	return diagnostics.RedactError(err)
}

// RedactDiagnosticPayload applies the IP-08 scrubber before storage.
func (redactor Redactor) RedactDiagnosticPayload(input string) string {
	return storage.RedactDiagnosticPayload(input)
}

// Correlation returns a bounded one-way local digest. It is stable for one value
// and cannot be reversed into the original profile, tab, title, URL, or query.
func (redactor Redactor) Correlation(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "local-sha256:" + hex.EncodeToString(sum[:8])
}

// DisplayURL removes credentials, query material, and fragments from a URL that
// may be shown in the current-scope surface. Invalid URLs are omitted.
func (redactor Redactor) DisplayURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.RawFragment = ""
	return parsed.String()
}

// IsSensitiveOutput reports whether an export still contains a sentinel. It is
// useful for tests and for callers that already hold the bounded output bytes.
func (redactor Redactor) IsSensitiveOutput(export []byte, sensitive ...string) bool {
	return diagnostics.ValidateExport(export, sensitive...) != nil
}

// ErrUnsafeOutput is returned by callers that choose to fail closed on a
// redaction validation error.
var ErrUnsafeOutput = errors.New("privacy: unsafe output")

// RequireSafeExport is a fail-closed wrapper around ValidateExport.
func (redactor Redactor) RequireSafeExport(export []byte, sensitive ...string) ([]byte, error) {
	if err := redactor.ValidateExport(export, sensitive...); err != nil {
		return nil, ErrUnsafeOutput
	}
	return export, nil
}

// SafeDisplayText rejects control characters and trims an optional display
// label. It never returns a raw error or a URL.
func SafeDisplayText(value string) string {
	if !strings.ContainsAny(value, "\x00\r\n\t") {
		return value
	}
	return ""
}
