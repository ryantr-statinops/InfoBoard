package diagnostics

import (
	"fmt"

	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

// Status is the stable, user-safe status vocabulary exposed by diagnostics and
// health responses. It never contains an error message or a raw field value.
type Status string

const (
	StatusNone                    Status = "none"
	StatusRuntimeProtocolError    Status = "runtime_protocol_error"
	StatusRuntimeIncompatible     Status = "runtime_incompatible"
	StatusProfileRefreshRequired  Status = "profile_refresh_required"
	StatusInputTooLarge           Status = "input_too_large"
	StatusProjectionRefresh       Status = "projection_refresh_required"
	StatusResultsRefresh          Status = "results_refresh_required"
	StatusIndexRebuilding         Status = "index_rebuilding"
	StatusQueryTimedOut           Status = "query_timed_out"
	StatusPersistenceDegraded     Status = "persistence_degraded"
	StatusRuntimeRestarting       Status = "runtime_restarting"
	StatusRuntimeRecoverableError Status = "runtime_recoverable_error"
	StatusRuntimeUnavailable      Status = "runtime_unavailable"
	StatusPermissionLimited       Status = "permission_limited"
	StatusPersistenceQuarantined  Status = "persistence_quarantined"
	StatusPersistenceUpgrade      Status = "persistence_upgrade_required"
)

func (status Status) Valid() bool {
	switch status {
	case StatusNone, StatusRuntimeProtocolError, StatusRuntimeIncompatible,
		StatusProfileRefreshRequired, StatusInputTooLarge, StatusProjectionRefresh,
		StatusResultsRefresh, StatusIndexRebuilding, StatusQueryTimedOut,
		StatusPersistenceDegraded, StatusRuntimeRestarting,
		StatusRuntimeRecoverableError, StatusRuntimeUnavailable,
		StatusPermissionLimited, StatusPersistenceQuarantined,
		StatusPersistenceUpgrade:
		return true
	default:
		return false
	}
}

// RetryPolicy is the explicit budget attached to a retryable outcome. It is
// never inferred from UI text. A zero MaxAttempts means automatic retry is not
// permitted for the class.
type RetryPolicy struct {
	Automatic         bool    `json:"automatic"`
	MaxAttempts       int     `json:"max_attempts"`
	BackoffMillis     []int64 `json:"backoff_ms,omitempty"`
	TotalBudgetMillis int64   `json:"total_budget_ms"`
	CooldownMillis    int64   `json:"cooldown_ms"`
}

// Mapping is the complete user-safe outcome for one error class.
type Mapping struct {
	Class           ErrorClass        `json:"error_class"`
	Status          Status            `json:"status"`
	Retryable       bool              `json:"retryable"`
	Policy          RetryPolicy       `json:"retry_policy"`
	Action          RecommendedAction `json:"recommended_action"`
	ExhaustedAction RecommendedAction `json:"exhausted_action"`
}

// Validate rejects a mapping that could cause an unbounded or contradictory
// retry path.
func (mapping Mapping) Validate() error {
	if !mapping.Class.Valid() || !mapping.Status.Valid() ||
		!mapping.Action.Valid() || !mapping.ExhaustedAction.Valid() {
		return fmt.Errorf("%w: mapping vocabulary", ErrInvalidEvent)
	}
	if mapping.Retryable != (mapping.Policy.MaxAttempts > 0) {
		return fmt.Errorf("%w: retryability budget mismatch", ErrInvalidEvent)
	}
	if mapping.Policy.MaxAttempts < 0 || mapping.Policy.TotalBudgetMillis < 0 ||
		mapping.Policy.CooldownMillis < 0 || len(mapping.Policy.BackoffMillis) > mapping.Policy.MaxAttempts {
		return fmt.Errorf("%w: retry policy bounds", ErrInvalidEvent)
	}
	if mapping.Retryable && (mapping.Policy.TotalBudgetMillis <= 0 || mapping.ExhaustedAction == ActionNone) {
		return fmt.Errorf("%w: retry policy incomplete", ErrInvalidEvent)
	}
	var total int64
	for _, backoff := range mapping.Policy.BackoffMillis {
		if backoff < 0 {
			return fmt.Errorf("%w: negative backoff", ErrInvalidEvent)
		}
		total += backoff
	}
	if total > mapping.Policy.TotalBudgetMillis {
		return fmt.Errorf("%w: backoff exceeds budget", ErrInvalidEvent)
	}
	if mapping.Policy.Automatic && !mapping.Retryable {
		return fmt.Errorf("%w: automatic retry without budget", ErrInvalidEvent)
	}
	return nil
}

var errorMappings = map[ErrorClass]Mapping{
	ErrorNone: {
		Class: ErrorNone, Status: StatusNone, Retryable: false,
		Action: ActionNone, ExhaustedAction: ActionNone,
	},
	ErrorInvalidFrame: {
		Class: ErrorInvalidFrame, Status: StatusRuntimeProtocolError, Retryable: false,
		Action: ActionReconnect, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorProtocolMismatch: {
		Class: ErrorProtocolMismatch, Status: StatusRuntimeIncompatible, Retryable: false,
		Action: ActionUpdateHost, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorProfileMismatch: {
		Class: ErrorProfileMismatch, Status: StatusProfileRefreshRequired, Retryable: true,
		Policy: RetryPolicy{Automatic: false, MaxAttempts: 1, BackoffMillis: []int64{0}, TotalBudgetMillis: 2_000, CooldownMillis: 500},
		Action: ActionRefreshProfile, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorPayloadLimit: {
		Class: ErrorPayloadLimit, Status: StatusInputTooLarge, Retryable: false,
		Action: ActionShortenQuery, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorSnapshotRequired: {
		Class: ErrorSnapshotRequired, Status: StatusProjectionRefresh, Retryable: true,
		Policy: RetryPolicy{Automatic: true, MaxAttempts: 1, BackoffMillis: []int64{0}, TotalBudgetMillis: 2_000, CooldownMillis: 250},
		Action: ActionRefreshProjection, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorRevisionMismatch: {
		Class: ErrorRevisionMismatch, Status: StatusResultsRefresh, Retryable: true,
		Policy: RetryPolicy{Automatic: true, MaxAttempts: 1, BackoffMillis: []int64{0}, TotalBudgetMillis: 2_000, CooldownMillis: 250},
		Action: ActionRefreshProjection, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorIndexRebuilding: {
		Class: ErrorIndexRebuilding, Status: StatusIndexRebuilding, Retryable: true,
		Policy: RetryPolicy{Automatic: false, MaxAttempts: 1, BackoffMillis: []int64{0}, TotalBudgetMillis: 2_000, CooldownMillis: 500},
		Action: ActionWait, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorQueryTimeout: {
		Class: ErrorQueryTimeout, Status: StatusQueryTimedOut, Retryable: true,
		Policy: RetryPolicy{Automatic: true, MaxAttempts: 1, BackoffMillis: []int64{0}, TotalBudgetMillis: 1_000, CooldownMillis: 250},
		Action: ActionRetry, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorPersistenceDegraded: {
		Class: ErrorPersistenceDegraded, Status: StatusPersistenceDegraded, Retryable: true,
		Policy: RetryPolicy{Automatic: true, MaxAttempts: 3, BackoffMillis: []int64{100, 500, 1_000}, TotalBudgetMillis: 5_000, CooldownMillis: 1_000},
		Action: ActionRepairPersistence, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorHostShutdown: {
		Class: ErrorHostShutdown, Status: StatusRuntimeRestarting, Retryable: true,
		Policy: RetryPolicy{Automatic: true, MaxAttempts: 3, BackoffMillis: []int64{100, 500, 1_000}, TotalBudgetMillis: 5_000, CooldownMillis: 500},
		Action: ActionReconnect, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorInternalFailure: {
		Class: ErrorInternalFailure, Status: StatusRuntimeRecoverableError, Retryable: true,
		Policy: RetryPolicy{Automatic: true, MaxAttempts: 1, BackoffMillis: []int64{0}, TotalBudgetMillis: 1_000, CooldownMillis: 500},
		Action: ActionRetry, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorHostMissing: {
		Class: ErrorHostMissing, Status: StatusRuntimeUnavailable, Retryable: true,
		Policy: RetryPolicy{Automatic: true, MaxAttempts: 3, BackoffMillis: []int64{100, 500, 1_000}, TotalBudgetMillis: 5_000, CooldownMillis: 500},
		Action: ActionReconnect, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorPermissionLimited: {
		Class: ErrorPermissionLimited, Status: StatusPermissionLimited, Retryable: true,
		Policy: RetryPolicy{Automatic: false, MaxAttempts: 1, BackoffMillis: []int64{0}, TotalBudgetMillis: 1_000, CooldownMillis: 1_000},
		Action: ActionRetry, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorStorageQuarantined: {
		Class: ErrorStorageQuarantined, Status: StatusPersistenceQuarantined, Retryable: true,
		Policy: RetryPolicy{Automatic: false, MaxAttempts: 1, BackoffMillis: []int64{0}, TotalBudgetMillis: 10_000, CooldownMillis: 1_000},
		Action: ActionRepairPersistence, ExhaustedAction: ActionOpenDiagnostics,
	},
	ErrorSchemaUpgradeRequired: {
		Class: ErrorSchemaUpgradeRequired, Status: StatusPersistenceUpgrade, Retryable: false,
		Action: ActionUpdateHost, ExhaustedAction: ActionOpenDiagnostics,
	},
}

// Lookup returns a copy of the mapping for a known class. Unknown classes are
// never silently treated as retryable.
func Lookup(class ErrorClass) (Mapping, bool) {
	mapping, ok := errorMappings[class]
	if !ok {
		return Mapping{}, false
	}
	mapping.Policy.BackoffMillis = append([]int64(nil), mapping.Policy.BackoffMillis...)
	return mapping, true
}

// Map returns the mapping for a class, falling back to INTERNAL_FAILURE for an
// unknown value while preserving a non-empty, bounded status.
func Map(class ErrorClass) Mapping {
	if mapping, ok := Lookup(class); ok {
		return mapping
	}
	mapping, _ := Lookup(ErrorInternalFailure)
	return mapping
}

// MapProtocolCode maps an IP-07 protocol error literal without redefining the
// protocol vocabulary.
func MapProtocolCode(code protocol.ErrorCode) (Mapping, bool) {
	return Lookup(ErrorClass(code))
}

// AllErrorClasses returns a stable copy of the mapped classes for fixture and
// completeness checks.
func AllErrorClasses() []ErrorClass {
	return []ErrorClass{
		ErrorNone, ErrorInvalidFrame, ErrorProtocolMismatch, ErrorProfileMismatch,
		ErrorPayloadLimit, ErrorSnapshotRequired, ErrorRevisionMismatch,
		ErrorIndexRebuilding, ErrorQueryTimeout, ErrorPersistenceDegraded,
		ErrorHostShutdown, ErrorInternalFailure, ErrorHostMissing,
		ErrorPermissionLimited, ErrorStorageQuarantined, ErrorSchemaUpgradeRequired,
	}
}
