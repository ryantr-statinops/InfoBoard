package privacy

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

const (
	scopeProfileA = "123e4567-e89b-42d3-a456-426614174000"
	scopeProfileB = "123e4567-e89b-42d3-a456-426614174001"
	scopeEpoch    = "123e4567-e89b-42d3-a456-426614174010"
)

func scopeA(t *testing.T) Scope {
	t.Helper()
	scope, err := ParseScope("chrome", scopeProfileA, "normal", "")
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func frame(t *testing.T, messageType protocol.MessageType, profileID string, payload any) []byte {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	encoded, failure := protocol.EncodeEnvelope(protocol.Envelope{
		Protocol:  protocol.CurrentVersion,
		Type:      messageType,
		RequestID: "request-1",
		ProfileID: profileID,
		Payload:   raw,
	}, protocol.DefaultLimits())
	if failure != nil {
		t.Fatal(failure)
	}
	return encoded
}

func tabRecord(profileID, contextKind string) map[string]any {
	return map[string]any{
		"profile_id":          profileID,
		"context_kind":        contextKind,
		"tab_id":              7,
		"window_id":           2,
		"title_display":       "Example",
		"title_search":        "example",
		"url_display":         "https://example.test",
		"url_search":          "https://example.test/path",
		"domain_display":      "example.test",
		"domain_search":       "example.test",
		"pinned":              false,
		"active":              false,
		"eligible":            true,
		"observed_at":         1,
		"projection_epoch":    scopeEpoch,
		"projection_revision": 7,
	}
}

func TestProfileScopeParsingRejectsForeignOrMalformedIdentity(t *testing.T) {
	valid := scopeA(t)
	if valid.Key() == "" {
		t.Fatal("empty scope key")
	}
	cases := []struct {
		family  string
		profile string
		context string
		nonce   string
	}{
		{"firefox", scopeProfileA, "normal", ""},
		{"chrome", "not-a-profile", "normal", ""},
		{"chrome", scopeProfileA, "unknown", ""},
		{"chrome", scopeProfileA, "normal", "nonce"},
		{"chrome", scopeProfileA, "private", string(make([]byte, 129))},
	}
	for index, testCase := range cases {
		if _, err := ParseScope(testCase.family, testCase.profile, testCase.context, testCase.nonce); !errors.Is(err, ErrInvalidScope) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}

func TestProfileCrossReadRejectedBeforePayloadUse(t *testing.T) {
	bound := scopeA(t)
	limits := protocol.DefaultLimits()

	validQuery := frame(t, protocol.TypeQuery, scopeProfileA, map[string]any{"query": "go", "current_window_id": 1})
	if _, failure := ValidateFrame(validQuery, bound, limits); failure != nil {
		t.Fatalf("valid query: %v", failure)
	}

	foreignEnvelope := frame(t, protocol.TypeQuery, scopeProfileB, map[string]any{"query": "go", "current_window_id": 1})
	if _, failure := ValidateFrame(foreignEnvelope, bound, limits); failure == nil || failure.Code != protocol.CodeProfileMismatch {
		t.Fatalf("foreign envelope = %+v", failure)
	}

	foreignSnapshot := frame(t, protocol.TypeSnapshot, scopeProfileA, map[string]any{
		"tabs":     []any{tabRecord(scopeProfileB, "normal")},
		"sequence": 1,
	})
	if _, failure := ValidateFrame(foreignSnapshot, bound, limits); failure == nil || failure.Code != protocol.CodeProfileMismatch {
		t.Fatalf("foreign snapshot = %+v", failure)
	}

	foreignDelta := frame(t, protocol.TypeDelta, scopeProfileA, map[string]any{
		"base_revision":  1,
		"sequence_start": 2,
		"sequence_end":   2,
		"operations": []any{map[string]any{
			"operation": "update",
			"tab_id":    7,
			"record":    tabRecord(scopeProfileB, "normal"),
		}},
	})
	if _, failure := ValidateFrame(foreignDelta, bound, limits); failure == nil || failure.Code != protocol.CodeProfileMismatch {
		t.Fatalf("foreign delta = %+v", failure)
	}

	foreignHello := frame(t, protocol.TypeHello, scopeProfileA, map[string]any{
		"extension_version":           "0.1.0",
		"browser_family":              "chrome",
		"browser_version":             "140",
		"context_kind":                "private",
		"capabilities":                []string{},
		"supported_protocol_versions": []int{1},
	})
	if _, failure := ValidateFrame(foreignHello, bound, limits); failure == nil || failure.Code != protocol.CodeProfileMismatch {
		t.Fatalf("foreign hello = %+v", failure)
	}
}

func TestProfilePrivateDurableWriteRejected(t *testing.T) {
	normal := scopeA(t)
	if failure := RejectPrivateDurableWrite(normal); failure != nil {
		t.Fatalf("normal write: %v", failure)
	}
	privateScope, err := ParseScope("chrome", scopeProfileA, "private", "nonce-1")
	if err != nil {
		t.Fatal(err)
	}
	if failure := RejectPrivateDurableWrite(privateScope); failure == nil || failure.Code != protocol.CodeProfileMismatch {
		t.Fatalf("private write = %+v", failure)
	}
}
