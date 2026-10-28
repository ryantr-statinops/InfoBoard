package protocol

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

const (
	maxLifecycleOperationBytes      = 16
	maxRankingModelVersionBytes     = 128
	minLifecycleResultLimit         = 1
	maxLifecycleResultLimit         = 50
	lifecycleOperationLoad          = "load"
	lifecycleOperationUpdate        = "update"
	lifecycleOperationReset         = "reset"
	lifecycleOperationUninstall     = "uninstall"
	lifecycleStatusComplete         = "complete"
	lifecycleStatusDegraded         = "degraded"
	lifecycleStatusRetryable        = "retryable"
	lifecycleStatusIncomplete       = "incomplete"
	lifecyclePersistenceReady       = "ready"
	lifecyclePersistenceDegraded    = "degraded"
	lifecyclePersistenceUnknown     = "unknown"
	lifecyclePersistenceUnavailable = "unavailable"
)

// LifecycleRequest is the validated host-bound lifecycle payload. The
// configuration bytes are present only for update and have already passed the
// strict contract decoder, including the bound-profile check.
type LifecycleRequest struct {
	Operation     string
	Configuration []byte
}

// LifecycleConfiguration is the profile-scoped settings shape returned with a
// lifecycle result. Profile identity and protocol versions are host-supplied.
type LifecycleConfiguration struct {
	ProfileID           string `json:"profile_id"`
	ResultLimit         int    `json:"result_limit"`
	Density             string `json:"density"`
	Theme               string `json:"theme"`
	ContextLabels       bool   `json:"context_labels"`
	RecencyEnabled      bool   `json:"recency_enabled"`
	ProtocolVersion     int    `json:"protocol_version"`
	RankingModelVersion string `json:"ranking_model_version"`
}

// LifecycleResult is the host-bound result plus the runtime resync action.
// It is a closed, redacted object: it carries no titles, URLs, queries, paths,
// SQLite text, or caller-supplied diagnostics.
type LifecycleResult struct {
	Operation          string                 `json:"operation"`
	ProfileID          domain.ProfileID       `json:"profile_id"`
	Status             string                 `json:"status"`
	Retryable          bool                   `json:"retryable"`
	Persistence        string                 `json:"persistence"`
	Durable            bool                   `json:"durable"`
	ErrorClass         string                 `json:"error_class"`
	CompletedSteps     []string               `json:"completed_steps"`
	OwnedStateVerified bool                   `json:"owned_state_verified"`
	ProjectionRevision uint64                 `json:"projection_revision"`
	Configuration      LifecycleConfiguration `json:"configuration"`
	ResyncRequired     bool                   `json:"resync_required"`
}

// DecodeLifecycle validates one lifecycle request before the hook or durable
// store is touched. Unknown keys, unknown operations, trailing bytes, and
// profile mismatches are rejected with the exact wire error literals.
func DecodeLifecycle(payload []byte, profile domain.ProfileID) (LifecycleRequest, *Error) {
	var operation string
	var configuration []byte
	failure := decodePayload(payload, map[string]fieldDecoder{
		"operation": decodeStringField(&operation, maxLifecycleOperationBytes, false),
		"configuration": func(decoder *json.Decoder) *Error {
			raw, err := decodeObject(decoder)
			if err != nil {
				return NewError(CodeInvalidFrame)
			}
			if len(raw) > MaxPayloadBytes {
				return NewError(CodePayloadLimit).WithBound("lifecycle_configuration_bytes", MaxPayloadBytes, len(raw))
			}
			configuration = append([]byte(nil), raw...)
			return nil
		},
	}, []string{"operation"})
	if failure != nil {
		return LifecycleRequest{}, failure
	}

	switch operation {
	case lifecycleOperationLoad, lifecycleOperationReset, lifecycleOperationUninstall:
		if configuration != nil {
			return LifecycleRequest{}, NewError(CodeInvalidFrame)
		}
	case lifecycleOperationUpdate:
		if configuration == nil {
			return LifecycleRequest{}, NewError(CodeInvalidFrame)
		}
		if failure := validateLifecycleConfiguration(configuration, profile); failure != nil {
			return LifecycleRequest{}, failure
		}
	default:
		return LifecycleRequest{}, NewError(CodeInvalidFrame)
	}

	return LifecycleRequest{Operation: operation, Configuration: configuration}, nil
}

func validateLifecycleConfiguration(raw []byte, profile domain.ProfileID) *Error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var configuration LifecycleConfiguration
	if err := decoder.Decode(&configuration); err != nil {
		return NewError(CodeInvalidFrame)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return NewError(CodeInvalidFrame)
	}
	if configuration.ProfileID != string(profile) {
		return NewError(CodeProfileMismatch)
	}
	if configuration.ResultLimit < minLifecycleResultLimit ||
		configuration.ResultLimit > maxLifecycleResultLimit ||
		(configuration.Density != "compact" && configuration.Density != "comfortable") ||
		(configuration.Theme != "system" && configuration.Theme != "light" && configuration.Theme != "dark") ||
		configuration.ProtocolVersion <= 0 ||
		!utf8.ValidString(configuration.RankingModelVersion) ||
		len(configuration.RankingModelVersion) > maxRankingModelVersionBytes {
		return NewError(CodeInvalidFrame)
	}
	return nil
}

// decodeLifecycleResult validates the lifecycle owner's encoded result before
// it reaches the extension. The hook is trusted, but a malformed or
// profile-crossing result is still a host failure rather than a wire leak.
func decodeLifecycleResult(payload []byte, profile domain.ProfileID, operation string) (LifecycleResult, *Error) {
	if len(payload) == 0 {
		return LifecycleResult{}, NewError(CodeInternalFailure)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var result LifecycleResult
	if err := decoder.Decode(&result); err != nil {
		return LifecycleResult{}, NewError(CodeInternalFailure)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return LifecycleResult{}, NewError(CodeInternalFailure)
	}
	if result.Operation != operation ||
		result.ProfileID != profile ||
		!validLifecycleStatus(result.Status) ||
		!validLifecycleErrorClass(result.ErrorClass) ||
		!validLifecyclePersistence(result.Persistence) ||
		!validLifecycleSteps(result.CompletedSteps) ||
		result.Configuration.ProfileID != string(profile) ||
		result.Configuration.ResultLimit < minLifecycleResultLimit ||
		result.Configuration.ResultLimit > maxLifecycleResultLimit ||
		(result.Configuration.Density != "compact" && result.Configuration.Density != "comfortable") ||
		(result.Configuration.Theme != "system" && result.Configuration.Theme != "light" && result.Configuration.Theme != "dark") ||
		result.Configuration.ProtocolVersion <= 0 ||
		!utf8.ValidString(result.Configuration.RankingModelVersion) ||
		len(result.Configuration.RankingModelVersion) > maxRankingModelVersionBytes {
		return LifecycleResult{}, NewError(CodeInternalFailure)
	}
	return result, nil
}

func validLifecycleStatus(status string) bool {
	switch status {
	case lifecycleStatusComplete, lifecycleStatusDegraded, lifecycleStatusRetryable, lifecycleStatusIncomplete:
		return true
	default:
		return false
	}
}

func validLifecyclePersistence(persistence string) bool {
	switch persistence {
	case lifecyclePersistenceReady, lifecyclePersistenceDegraded, lifecyclePersistenceUnavailable, lifecyclePersistenceUnknown:
		return true
	default:
		return false
	}
}

func validLifecycleSteps(steps []string) bool {
	for _, step := range steps {
		switch step {
		case "storage_reset", "index_cleared", "snapshot_requested", "session_closed", "storage_uninstall", "paths_removed":
		default:
			return false
		}
	}
	return true
}

func validLifecycleErrorClass(class string) bool {
	switch class {
	case "",
		"invalid_configuration",
		"profile_mismatch",
		"storage_unavailable",
		"storage_timeout",
		"storage_failure",
		"schema_upgrade_required",
		"database_quarantined",
		"database_corrupt",
		"schema_integrity_failed",
		"quarantine_failed",
		"recovery_failed",
		"unsafe_database_path",
		"compatibility_mismatch",
		"reset_runtime_unavailable",
		"projection_reset_failed",
		"projection_reset_incomplete",
		"cancelled",
		"uninstall_not_configured",
		"session_close_failed",
		"cleanup_manifest_invalid",
		"cleanup_cancelled",
		"cleanup_failed",
		"path_safety",
		"unknown":
		return true
	default:
		return false
	}
}
