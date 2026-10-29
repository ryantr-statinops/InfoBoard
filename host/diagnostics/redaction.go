package diagnostics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

// Redaction is the single allowlist boundary for persisted diagnostics, local
// exports, and UI-safe status payloads. Unknown keys and values that do not fit
// a bounded field kind are dropped rather than recursively serialized.
type Redaction struct{}

// SanitizeMap returns a new map containing only allowlisted diagnostic fields
// with validated values. It never copies an unknown key or an arbitrary nested
// object.
func (Redaction) SanitizeMap(input map[string]any) map[string]any {
	output := make(map[string]any)
	for key, value := range input {
		if sanitized, ok := sanitizeField(key, value); ok {
			output[key] = sanitized
		}
	}
	return output
}

// SanitizeRecords applies the same boundary to a slice of context maps.
func (Redaction) SanitizeRecords(input []map[string]any) []map[string]any {
	output := make([]map[string]any, 0, len(input))
	for _, record := range input {
		output = append(output, Redaction{}.SanitizeMap(record))
	}
	return output
}

// RedactEvent projects an event through the allowlist. Sensitive or malformed
// version fields are omitted rather than exported.
func (Redaction) RedactEvent(event Event) map[string]any {
	return Redaction{}.SanitizeMap(map[string]any{
		"schema_version":        event.SchemaVersion,
		"event_id":              event.EventID,
		"event_time":            event.EventTime,
		"event_sequence":        event.EventSequence,
		"session_id":            event.SessionID,
		"component":             string(event.Component),
		"from_state":            string(event.FromState),
		"to_state":              string(event.ToState),
		"reason_code":           string(event.ReasonCode),
		"error_class":           string(event.ErrorClass),
		"retryable":             event.Retryable,
		"recommended_action":    string(event.RecommendedAction),
		"protocol_version":      event.ProtocolVersion,
		"ranking_model_version": event.RankingModelVersion,
		"store_schema_version":  event.StoreSchemaVersion,
		"projection_revision":   event.ProjectionRevision,
		"indexed_revision":      event.IndexedRevision,
		"projection_sequence":   event.ProjectionSequence,
		"counts":                countsMap(event.Counts),
		"duration_ms":           event.Timing.DurationMillis,
		"queue_ms":              event.Timing.QueueMillis,
		"latency_bucket":        string(event.Timing.Latency),
	})
}

// RedactPayload projects a health payload through the allowlist. The active
// profile is represented only by a stable local digest; raw profile IDs and
// browser fields are never exported.
func (Redaction) RedactPayload(payload HealthPayload) map[string]any {
	counts := map[string]any{
		"frames_read":        payload.Counters.FramesRead,
		"frames_written":     payload.Counters.FramesWritten,
		"requests_started":   payload.Counters.RequestsStarted,
		"requests_completed": payload.Counters.RequestsCompleted,
		"requests_rejected":  payload.Counters.RequestsRejected,
		"transitions":        payload.Counters.Transitions,
		"protocol_errors":    payload.Counters.ProtocolErrors,
		"dropped":            payload.Counters.DiagnosticsDropped,
	}
	return Redaction{}.SanitizeMap(map[string]any{
		"host_version":          payload.HostVersion,
		"protocol_version":      payload.ProtocolVersion,
		"ranking_model_version": payload.RankingModelVersion,
		"store_schema_version":  payload.SchemaVersion,
		"profile_scope":         ProfileScope(payload.ProfileID),
		"context_kind":          string(payload.ContextKind),
		"state":                 string(payload.State),
		"host":                  string(payload.Host),
		"protocol":              string(payload.Protocol),
		"projection":            string(payload.Projection),
		"index":                 string(payload.Index),
		"storage":               string(payload.Storage),
		"projection_freshness":  string(payload.ProjectionFreshness),
		"index_freshness":       string(payload.IndexFreshness),
		"projection_revision":   payload.ProjectionRevision,
		"indexed_revision":      payload.IndexedRevision,
		"projection_sequence":   payload.ProjectionSequence,
		"last_handshake_at":     payload.LastHandshakeAt,
		"updated_at":            payload.UpdatedAt,
		"diagnostic_age_ms":     payload.DiagnosticAgeMillis,
		"retryable":             payload.Retryable,
		"counts":                counts,
	})
}

// RedactMapping projects an error mapping without any raw message or cause.
func (Redaction) RedactMapping(mapping Mapping) map[string]any {
	return Redaction{}.SanitizeMap(map[string]any{
		"error_class":        string(mapping.Class),
		"status":             string(mapping.Status),
		"retryable":          mapping.Retryable,
		"recommended_action": string(mapping.Action),
		"exhausted_action":   string(mapping.ExhaustedAction),
		"automatic":          mapping.Policy.Automatic,
		"max_attempts":       mapping.Policy.MaxAttempts,
		"backoff_ms":         mapping.Policy.BackoffMillis,
		"total_budget_ms":    mapping.Policy.TotalBudgetMillis,
		"cooldown_ms":        mapping.Policy.CooldownMillis,
	})
}

// ExportEvents serializes only redacted event maps.
func (Redaction) ExportEvents(events []Event) ([]byte, error) {
	records := make([]map[string]any, 0, len(events))
	for _, event := range events {
		records = append(records, Redaction{}.RedactEvent(event))
	}
	return json.Marshal(records)
}

// ExportPayload serializes only a redacted health payload.
func (Redaction) ExportPayload(payload HealthPayload) ([]byte, error) {
	return json.Marshal(Redaction{}.RedactPayload(payload))
}

// ExportRecords serializes only redacted context maps.
func (Redaction) ExportRecords(records []map[string]any) ([]byte, error) {
	return json.Marshal(Redaction{}.SanitizeRecords(records))
}

// ProfileScope returns a stable, non-reversible local digest of the active
// profile. The digest is scoped to this local diagnostics boundary and is not
// a browser or network identifier.
func ProfileScope(profile domain.ProfileID) string {
	if profile == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("infoboard-diagnostics-profile:" + string(profile)))
	return "local-sha256:" + hex.EncodeToString(sum[:8])
}

// SafeError is a typed diagnostic error whose Error method exposes only the
// mapped user-safe status, never the underlying cause or field values.
type SafeError struct {
	Class ErrorClass
	Cause error
}

func (safe SafeError) Error() string { return string(Map(safe.Class).Status) }
func (safe SafeError) Unwrap() error { return safe.Cause }

// NewSafeError wraps a cause without allowing it to reach serialized output.
func NewSafeError(class ErrorClass, cause error) error {
	return SafeError{Class: class, Cause: cause}
}

// RedactError returns a user-safe status for any error. It never returns the
// original error text.
func RedactError(err error) string {
	if err == nil {
		return ""
	}
	var safe SafeError
	if errors.As(err, &safe) {
		return string(Map(safe.Class).Status)
	}
	return string(StatusRuntimeRecoverableError)
}

func countsMap(counts EventCounts) map[string]any {
	return map[string]any{
		"records":  counts.RecordCount,
		"dropped":  counts.DroppedCount,
		"retries":  counts.RetryCount,
		"failures": counts.FailureCount,
	}
}

func sanitizeField(key string, value any) (any, bool) {
	switch key {
	case "schema_version":
		return boundedInt(value, 0, int64(EventSchemaVersion))
	case "event_id":
		text, ok := value.(string)
		if !ok || !validOpaqueID(text, 96) {
			return nil, false
		}
		return text, true
	case "event_time", "last_handshake_at", "updated_at":
		return safeTimestamp(value)
	case "event_sequence", "store_schema_version", "projection_revision",
		"indexed_revision", "projection_sequence", "diagnostic_age_ms",
		"freshness_age_ms", "duration_ms", "queue_ms", "total_budget_ms",
		"cooldown_ms", "max_attempts":
		return boundedInt(value, 0, math.MaxInt64)
	case "protocol_version":
		return boundedInt(value, 0, 1_000_000)
	case "session_id":
		text, ok := value.(string)
		if !ok || !validOpaqueID(text, 64) {
			return nil, false
		}
		return text, true
	case "profile_scope":
		text, ok := value.(string)
		if !ok || !safeDigest(text) {
			return nil, false
		}
		return text, true
	case "ranking_model_version", "host_version":
		text, ok := value.(string)
		if !ok || !safeVersion(text) {
			return nil, false
		}
		return text, true
	case "component":
		return safeEnum(value, func(text string) bool { return Component(text).Valid() })
	case "from_state":
		return safeOptionalEnum(value, func(text string) bool { return State(text).Valid() })
	case "to_state", "state":
		return safeEnum(value, func(text string) bool { return State(text).Valid() })
	case "reason_code":
		return safeEnum(value, func(text string) bool { return ReasonCode(text).Valid() })
	case "error_class":
		return safeOptionalEnum(value, func(text string) bool { return ErrorClass(text).Valid() })
	case "status":
		return safeEnum(value, func(text string) bool { return Status(text).Valid() })
	case "recommended_action", "exhausted_action":
		return safeEnum(value, func(text string) bool { return RecommendedAction(text).Valid() })
	case "latency_bucket":
		return safeEnum(value, func(text string) bool { return LatencyBucket(text).Valid() })
	case "context_kind":
		return safeEnum(value, func(text string) bool {
			return text == string(domain.ContextNormal) || text == string(domain.ContextPrivate)
		})
	case "host":
		return safeEnum(value, func(text string) bool { return HostState(text).Valid() })
	case "protocol":
		return safeEnum(value, func(text string) bool { return ProtocolState(text).Valid() })
	case "projection":
		return safeEnum(value, func(text string) bool { return ProjectionState(text).Valid() })
	case "index":
		return safeEnum(value, func(text string) bool { return IndexState(text).Valid() })
	case "storage":
		return safeEnum(value, func(text string) bool { return StorageState(text).Valid() })
	case "projection_freshness", "index_freshness":
		return safeEnum(value, func(text string) bool { return Freshness(text).Valid() })
	case "retryable", "automatic":
		boolean, ok := value.(bool)
		if !ok {
			return nil, false
		}
		return boolean, true
	case "backoff_ms":
		return safeBackoff(value)
	case "counts":
		return safeCounts(value)
	default:
		return nil, false
	}
}

func safeEnum(value any, valid func(string) bool) (any, bool) {
	text, ok := value.(string)
	if !ok || !valid(text) {
		return nil, false
	}
	return text, true
}

func safeOptionalEnum(value any, valid func(string) bool) (any, bool) {
	text, ok := value.(string)
	if !ok || (text != "" && !valid(text)) {
		return nil, false
	}
	return text, true
}

func safeTimestamp(value any) (any, bool) {
	switch timestamp := value.(type) {
	case time.Time:
		if timestamp.IsZero() {
			return nil, false
		}
		return timestamp.UTC().Format(time.RFC3339Nano), true
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, timestamp)
		if err != nil || parsed.IsZero() {
			return nil, false
		}
		return parsed.UTC().Format(time.RFC3339Nano), true
	default:
		return nil, false
	}
}

func safeVersion(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' || char == ':' {
			continue
		}
		return false
	}
	return true
}

func safeDigest(value string) bool {
	if !strings.HasPrefix(value, "local-sha256:") || len(value) != len("local-sha256:")+16 {
		return false
	}
	for _, char := range value[len("local-sha256:"):] {
		if (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') {
			continue
		}
		return false
	}
	return true
}

func safeBackoff(value any) (any, bool) {
	var values []int64
	switch typed := value.(type) {
	case []int64:
		values = append(values, typed...)
	case []any:
		for _, item := range typed {
			number, ok := boundedInt(item, 0, math.MaxInt64)
			if !ok {
				return nil, false
			}
			values = append(values, number.(int64))
		}
	default:
		return nil, false
	}
	if len(values) > 16 {
		return nil, false
	}
	return values, true
}

func safeCounts(value any) (any, bool) {
	input, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	allowed := map[string]bool{
		"records": true, "dropped": true, "retries": true, "failures": true,
		"frames_read": true, "frames_written": true, "requests_started": true,
		"requests_completed": true, "requests_rejected": true, "transitions": true,
		"protocol_errors": true, "inserted": true, "rejected": true,
		"age_evicted": true, "byte_evicted": true, "persistence_failed": true,
	}
	output := make(map[string]any)
	for key, item := range input {
		if !allowed[key] {
			continue
		}
		number, ok := boundedInt(item, 0, math.MaxInt64)
		if !ok {
			continue
		}
		output[key] = number
	}
	return output, true
}

func boundedInt(value any, minimum, maximum int64) (any, bool) {
	var number int64
	switch typed := value.(type) {
	case int:
		number = int64(typed)
	case int64:
		number = typed
	case uint:
		if uint64(typed) > math.MaxInt64 {
			return nil, false
		}
		number = int64(typed)
	case uint64:
		if typed > math.MaxInt64 {
			return nil, false
		}
		number = int64(typed)
	case float64:
		if math.Trunc(typed) != typed || typed > math.MaxInt64 || typed < math.MinInt64 {
			return nil, false
		}
		number = int64(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return nil, false
		}
		number = parsed
	default:
		return nil, false
	}
	if number < minimum || number > maximum {
		return nil, false
	}
	return number, true
}

// ValidateExport proves that a serialized export contains no known sensitive
// markers. It is a final defense; the allowlist above is the primary boundary.
func ValidateExport(export []byte, sensitive ...string) error {
	if !json.Valid(export) {
		return fmt.Errorf("%w: export is not valid JSON", ErrInvalidEvent)
	}
	text := string(export)
	for _, marker := range sensitive {
		if marker != "" && strings.Contains(text, marker) {
			return fmt.Errorf("%w: sensitive marker reached export", ErrInvalidEvent)
		}
	}
	return nil
}
