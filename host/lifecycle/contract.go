// Package lifecycle owns profile-scoped settings and reset/uninstall
// orchestration around the IP-08 repository. It does not own SQLite schema or
// the browser projection.
package lifecycle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

const (
	MinResultLimit             = 1
	MaxResultLimit             = 50
	DefaultResultLimit         = 10
	MaxRankingModelVersionSize = 128
)

// Density is the bounded display density.
type Density string

const (
	DensityCompact     Density = "compact"
	DensityComfortable Density = "comfortable"
)

// Theme is the bounded display theme.
type Theme string

const (
	ThemeSystem Theme = "system"
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
)

// PersistenceState describes whether the effective configuration came from a
// healthy durable store. It never carries SQLite text or paths.
type PersistenceState string

const (
	PersistenceReady       PersistenceState = "ready"
	PersistenceDegraded    PersistenceState = "degraded"
	PersistenceUnavailable PersistenceState = "unavailable"
	PersistenceUnknown     PersistenceState = "unknown"
)

// ConfigKey identifies one user-editable setting.
type ConfigKey string

const (
	ConfigResultLimit   ConfigKey = "result_limit"
	ConfigDensity       ConfigKey = "density"
	ConfigTheme         ConfigKey = "theme"
	ConfigContextLabels ConfigKey = "context_labels"
	ConfigRecency       ConfigKey = "recency_enabled"
)

// ProfileConfig is the exact profile-scoped wire shape. ProfileID and the
// version fields are supplied by the host/session, not chosen by the UI.
type ProfileConfig struct {
	ProfileID           domain.ProfileID `json:"profile_id"`
	ResultLimit         int              `json:"result_limit"`
	Density             Density          `json:"density"`
	Theme               Theme            `json:"theme"`
	ContextLabels       bool             `json:"context_labels"`
	RecencyEnabled      bool             `json:"recency_enabled"`
	ProtocolVersion     int              `json:"protocol_version"`
	RankingModelVersion string           `json:"ranking_model_version"`
}

// EffectiveConfiguration separates the effective values from whether they are
// durable. A degraded read/write returns usable values with Durable=false.
type EffectiveConfiguration struct {
	Configuration ProfileConfig    `json:"configuration"`
	Persistence   PersistenceState `json:"persistence"`
	Durable       bool             `json:"durable"`
	Retryable     bool             `json:"retryable"`
	ErrorClass    ErrorClass       `json:"error_class"`
}

var (
	ErrInvalidConfiguration = errors.New("invalid configuration")
	ErrProfileMismatch      = errors.New("profile mismatch")
)

// ValidationError is a closed validation failure. Field is a fixed key name,
// never caller data.
type ValidationError struct {
	Field ConfigKey
	Err   error
}

func (failure *ValidationError) Error() string {
	if failure == nil {
		return "invalid configuration"
	}
	return fmt.Sprintf("%s: %s", failure.Field, failure.Err)
}

func (failure *ValidationError) Unwrap() error {
	if failure == nil {
		return ErrInvalidConfiguration
	}
	return failure.Err
}

func invalid(field ConfigKey) error {
	return &ValidationError{Field: field, Err: ErrInvalidConfiguration}
}

// DefaultProfileConfig returns the contract defaults for one validated profile.
func DefaultProfileConfig(profileID domain.ProfileID, protocolVersion int, rankingModelVersion string) ProfileConfig {
	return ProfileConfig{
		ProfileID:           profileID,
		ResultLimit:         DefaultResultLimit,
		Density:             DensityComfortable,
		Theme:               ThemeSystem,
		ContextLabels:       true,
		RecencyEnabled:      true,
		ProtocolVersion:     protocolVersion,
		RankingModelVersion: rankingModelVersion,
	}
}

// Validate rejects empty/mismatched profiles, out-of-range limits, invalid
// enums, malformed booleans, and oversized version strings. expectedProfile may
// be empty only when the caller has already authenticated the profile.
func (configuration ProfileConfig) Validate(expectedProfile domain.ProfileID) error {
	profile, err := domain.ParseProfileID(string(configuration.ProfileID))
	if err != nil {
		return invalid("profile_id")
	}
	if expectedProfile != "" && profile != expectedProfile {
		return &ValidationError{Field: "profile_id", Err: ErrProfileMismatch}
	}
	if configuration.ResultLimit < MinResultLimit || configuration.ResultLimit > MaxResultLimit {
		return invalid(ConfigResultLimit)
	}
	if configuration.Density != DensityCompact && configuration.Density != DensityComfortable {
		return invalid(ConfigDensity)
	}
	if configuration.Theme != ThemeSystem && configuration.Theme != ThemeLight && configuration.Theme != ThemeDark {
		return invalid(ConfigTheme)
	}
	if configuration.ProtocolVersion <= 0 {
		return invalid("protocol_version")
	}
	if !utf8.ValidString(configuration.RankingModelVersion) ||
		len(configuration.RankingModelVersion) > MaxRankingModelVersionSize {
		return invalid("ranking_model_version")
	}
	return nil
}

// DecodeProfileConfig strictly decodes one wire object. Unknown keys, trailing
// bytes, malformed JSON, and any validation failure are rejected before the
// caller can write or delete anything.
func DecodeProfileConfig(raw []byte, expectedProfile domain.ProfileID) (ProfileConfig, error) {
	if len(raw) == 0 {
		return ProfileConfig{}, ErrInvalidConfiguration
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var configuration ProfileConfig
	if err := decoder.Decode(&configuration); err != nil {
		return ProfileConfig{}, ErrInvalidConfiguration
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return ProfileConfig{}, ErrInvalidConfiguration
	}
	if err := configuration.Validate(expectedProfile); err != nil {
		return ProfileConfig{}, err
	}
	return configuration, nil
}

// EncodeProfileConfig serializes a validated configuration without adding
// caller-controlled fields.
func EncodeProfileConfig(configuration ProfileConfig) ([]byte, error) {
	if err := configuration.Validate(configuration.ProfileID); err != nil {
		return nil, err
	}
	return json.Marshal(configuration)
}
