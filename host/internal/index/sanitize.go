package index

import (
	"unicode"
	"unicode/utf8"
)

// NormalizeErrorCode is the closed set of bounded normalization failures.
type NormalizeErrorCode string

const (
	ErrInvalidUTF8     NormalizeErrorCode = "INVALID_UTF8"
	ErrControlChar     NormalizeErrorCode = "CONTROL_CHAR"
	ErrPhraseDelimiter NormalizeErrorCode = "PHRASE_DELIMITER"
	ErrQueryTooLong    NormalizeErrorCode = "QUERY_TOO_LONG"
)

// NormalizeError is a typed, redacted failure. It never carries input values.
type NormalizeError struct{ Code NormalizeErrorCode }

func (e *NormalizeError) Error() string { return string(e.Code) }

// validateUTF8 rejects invalid byte sequences before any allocation.
func validateUTF8(input string) *NormalizeError {
	if !utf8.ValidString(input) {
		return &NormalizeError{Code: ErrInvalidUTF8}
	}
	return nil
}

// validateControl rejects C0/C1 controls except tab, newline, and carriage
// return, which the whitespace policy normalizes later.
func validateControl(input string) *NormalizeError {
	for _, r := range input {
		if r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		if unicode.IsControl(r) {
			return &NormalizeError{Code: ErrControlChar}
		}
	}
	return nil
}

// validate runs the bounded pre-normalization checks.
func validate(input string) *NormalizeError {
	if err := validateUTF8(input); err != nil {
		return err
	}
	return validateControl(input)
}
