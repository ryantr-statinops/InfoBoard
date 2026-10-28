package lifecycle

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/domain"
)

const (
	profileA = domain.ProfileID("11111111-1111-4111-8111-111111111111")
	profileB = domain.ProfileID("22222222-2222-4222-8222-222222222222")
)

func validConfig(profile domain.ProfileID) ProfileConfig {
	return ProfileConfig{
		ProfileID:           profile,
		ResultLimit:         12,
		Density:             DensityCompact,
		Theme:               ThemeDark,
		ContextLabels:       false,
		RecencyEnabled:      false,
		ProtocolVersion:     1,
		RankingModelVersion: "rank-1",
	}
}

func TestDefaultProfileConfigMatchesContract(t *testing.T) {
	configuration := DefaultProfileConfig(profileA, 1, "rank-1")
	if configuration.ResultLimit != 10 ||
		configuration.Density != DensityComfortable ||
		configuration.Theme != ThemeSystem ||
		!configuration.ContextLabels ||
		!configuration.RecencyEnabled ||
		configuration.ProtocolVersion != 1 ||
		configuration.RankingModelVersion != "rank-1" {
		t.Fatalf("unexpected defaults: %+v", configuration)
	}
	if err := configuration.Validate(profileA); err != nil {
		t.Fatalf("defaults must validate: %v", err)
	}
}

func TestValidateProfileConfigRejectsClosedFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ProfileConfig)
		field  ConfigKey
		target error
	}{
		{"empty profile", func(c *ProfileConfig) { c.ProfileID = "" }, "profile_id", ErrInvalidConfiguration},
		{"mismatched profile", func(c *ProfileConfig) { c.ProfileID = profileB }, "profile_id", ErrProfileMismatch},
		{"low limit", func(c *ProfileConfig) { c.ResultLimit = 0 }, ConfigResultLimit, ErrInvalidConfiguration},
		{"high limit", func(c *ProfileConfig) { c.ResultLimit = 51 }, ConfigResultLimit, ErrInvalidConfiguration},
		{"density", func(c *ProfileConfig) { c.Density = "wide" }, ConfigDensity, ErrInvalidConfiguration},
		{"theme", func(c *ProfileConfig) { c.Theme = "neon" }, ConfigTheme, ErrInvalidConfiguration},
		{"protocol", func(c *ProfileConfig) { c.ProtocolVersion = 0 }, "protocol_version", ErrInvalidConfiguration},
		{"ranking", func(c *ProfileConfig) { c.RankingModelVersion = string(make([]byte, MaxRankingModelVersionSize+1)) }, "ranking_model_version", ErrInvalidConfiguration},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			configuration := validConfig(profileA)
			testCase.mutate(&configuration)
			err := configuration.Validate(profileA)
			if !errors.Is(err, testCase.target) {
				t.Fatalf("expected %v, got %v", testCase.target, err)
			}
			var validation *ValidationError
			if !errors.As(err, &validation) || validation.Field != testCase.field {
				t.Fatalf("expected field %q, got %#v", testCase.field, validation)
			}
		})
	}
}

func TestDecodeProfileConfigIsStrict(t *testing.T) {
	valid, err := json.Marshal(validConfig(profileA))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeProfileConfig(valid, profileA); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	unknown := []byte(`{"profile_id":"11111111-1111-4111-8111-111111111111","result_limit":10,"density":"comfortable","theme":"system","context_labels":true,"recency_enabled":true,"protocol_version":1,"ranking_model_version":"","extra":1}`)
	if _, err := DecodeProfileConfig(unknown, profileA); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("unknown field must fail closed, got %v", err)
	}

	trailing := append(append([]byte(nil), valid...), []byte(`{}`)...)
	if _, err := DecodeProfileConfig(trailing, profileA); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("trailing JSON must fail closed, got %v", err)
	}

	if _, err := DecodeProfileConfig(valid, profileB); !errors.Is(err, ErrProfileMismatch) {
		t.Fatalf("profile mismatch must fail closed, got %v", err)
	}
}

func TestEncodeProfileConfigRevalidates(t *testing.T) {
	encoded, err := EncodeProfileConfig(validConfig(profileA))
	if err != nil {
		t.Fatalf("encode valid config: %v", err)
	}
	decoded, err := DecodeProfileConfig(encoded, profileA)
	if err != nil {
		t.Fatalf("decode encoded config: %v", err)
	}
	if decoded != validConfig(profileA) {
		t.Fatalf("round trip changed config: %+v", decoded)
	}
	invalidConfig := validConfig(profileA)
	invalidConfig.Density = "wide"
	if _, err := EncodeProfileConfig(invalidConfig); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("invalid config must not encode, got %v", err)
	}
}
