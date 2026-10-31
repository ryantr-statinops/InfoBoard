package privacy

import (
	"fmt"
	"net/url"
	"unicode/utf8"

	"github.com/ryantr-statinops/InfoBoard/host/diagnostics"
	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
	"github.com/ryantr-statinops/InfoBoard/host/storage"
)

const (
	// OperatingTabLimit is the product operating envelope. IP-07 still owns the
	// larger wire-level snapshot cap; IP-17 refuses to allocate/index beyond this
	// declared operating bound.
	OperatingTabLimit = 1000
	// MinResultLimit is the smallest accepted result limit.
	MinResultLimit = 1
)

// ValidateFrameBytes applies the frame cap before envelope decoding.
func ValidateFrameBytes(size int) *protocol.Error {
	if size < 0 || size > protocol.MaxFrameBytes {
		return protocol.NewError(protocol.CodePayloadLimit).WithBound("frame_bytes", protocol.MaxFrameBytes, size)
	}
	return nil
}

// ValidatePayloadBytes applies the encoded JSON payload cap.
func ValidatePayloadBytes(size int, limits protocol.Limits) *protocol.Error {
	if size < 0 || size > limits.MaxPayloadBytes {
		return protocol.NewError(protocol.CodePayloadLimit).WithBound("payload_bytes", limits.MaxPayloadBytes, size)
	}
	return nil
}

// ValidateResultLimit resolves an optional result limit and enforces 1..50.
func ValidateResultLimit(limit int, hasLimit bool, defaultLimit int) (int, *protocol.Error) {
	if !hasLimit {
		if defaultLimit < MinResultLimit || defaultLimit > protocol.MaxResults {
			return 0, protocol.NewError(protocol.CodePayloadLimit).WithBound("result_limit", protocol.MaxResults, defaultLimit)
		}
		return defaultLimit, nil
	}
	if limit < MinResultLimit || limit > protocol.MaxResults {
		return 0, protocol.NewError(protocol.CodePayloadLimit).WithBound("result_limit", protocol.MaxResults, limit)
	}
	return limit, nil
}

// ValidateOperatingTabCount rejects snapshots above the product operating bound.
func ValidateOperatingTabCount(count int) *protocol.Error {
	if count < 0 || count > OperatingTabLimit {
		return protocol.NewError(protocol.CodePayloadLimit).WithBound("operating_tabs", OperatingTabLimit, count)
	}
	return nil
}

// ValidateDeltaOperationCount rejects an over-bound delta before application.
func ValidateDeltaOperationCount(count int, limits protocol.Limits) *protocol.Error {
	if count < 0 || count > limits.MaxDeltaOperations {
		return protocol.NewError(protocol.CodePayloadLimit).WithBound("delta_operations", limits.MaxDeltaOperations, count)
	}
	return nil
}

// ValidateQuery applies the IP-07 query scalar and context bounds.
func ValidateQuery(query protocol.Query, limits protocol.Limits) *protocol.Error {
	if !utf8.ValidString(query.Text) {
		return protocol.NewError(protocol.CodeInvalidFrame)
	}
	if utf8.RuneCountInString(query.Text) > limits.MaxQueryScalars {
		return protocol.NewError(protocol.CodePayloadLimit).WithBound("query_scalars", limits.MaxQueryScalars, utf8.RuneCountInString(query.Text))
	}
	if query.CurrentWindowID < 0 || (query.CurrentGroupID != nil && *query.CurrentGroupID < 0) {
		return protocol.NewError(protocol.CodeInvalidFrame)
	}
	if _, failure := ValidateResultLimit(query.ResultLimit, query.HasResultLimit, storage.DefaultResultLimit); failure != nil {
		return failure
	}
	return nil
}

// ValidateRecord applies protocol field bounds plus the IP-17 URL shape rule.
func ValidateRecord(record protocol.TabRecord, limits protocol.Limits) *protocol.Error {
	if failure := record.Validate(limits); failure != nil {
		return failure
	}
	if !validHTTPURL(record.URLDisplay) || !validHTTPURL(record.URLSearch) {
		return protocol.NewError(protocol.CodeInvalidFrame)
	}
	return nil
}

func validHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

// ValidateDiagnosticRecordBytes bounds one diagnostic row before insertion.
func ValidateDiagnosticRecordBytes(record storage.DiagnosticRecord) *protocol.Error {
	size := len(record.RedactedPayload)
	if size > diagnostics.MaxDiagnosticRecordBytes {
		return protocol.NewError(protocol.CodePayloadLimit).WithBound("diagnostic_record_bytes", diagnostics.MaxDiagnosticRecordBytes, size)
	}
	return nil
}

// ValidateDiagnosticPayload validates UTF-8 and scrubs a payload before storage.
func ValidateDiagnosticPayload(value string) (string, *protocol.Error) {
	if !utf8.ValidString(value) {
		return "", protocol.NewError(protocol.CodeInvalidFrame)
	}
	scrubbed := storage.RedactDiagnosticPayload(value)
	if len(scrubbed) > diagnostics.MaxDiagnosticRecordBytes {
		return "", protocol.NewError(protocol.CodePayloadLimit).WithBound("diagnostic_record_bytes", diagnostics.MaxDiagnosticRecordBytes, len(scrubbed))
	}
	return scrubbed, nil
}

// BoundDescription is a safe human-readable description for tests and audit
// output. It contains no input value.
func BoundDescription(name string, bound int) string {
	return fmt.Sprintf("%s<=%d", name, bound)
}
