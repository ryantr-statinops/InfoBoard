package privacy

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

func TestSecretRejectionCoversClosedShapes(t *testing.T) {
	var fixture struct {
		Cases []struct {
			CaseID       string `json:"case_id"`
			Field        string `json:"field"`
			Value        string `json:"value"`
			ExpectedKind string `json:"expected_kind"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(privacyFixture(t, "secret-rejection.json"), &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) != 6 {
		t.Fatalf("secret cases = %d", len(fixture.Cases))
	}
	for _, testCase := range fixture.Cases {
		findings := DetectSecret(testCase.Field, testCase.Value)
		if len(findings) == 0 {
			t.Fatalf("%s was not rejected", testCase.CaseID)
		}
		found := false
		for _, finding := range findings {
			if string(finding.Kind) == testCase.ExpectedKind {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s kinds = %v, want %s", testCase.CaseID, findings, testCase.ExpectedKind)
		}
		safe := SecretFindingsSafe(findings)
		serialized, err := json.Marshal(safe)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(serialized), testCase.Value) {
			t.Fatalf("%s echoed the raw secret", testCase.CaseID)
		}
	}
}

func TestInputLimitsRejectBeforeWork(t *testing.T) {
	limits := protocol.DefaultLimits()

	if failure := ValidateFrameBytes(protocol.MaxFrameBytes + 1); failure == nil || failure.Code != protocol.CodePayloadLimit || failure.LimitName != "frame_bytes" {
		t.Fatalf("frame limit = %+v", failure)
	}
	if failure := ValidatePayloadBytes(limits.MaxPayloadBytes+1, limits); failure == nil || failure.Code != protocol.CodePayloadLimit || failure.LimitName != "payload_bytes" {
		t.Fatalf("payload limit = %+v", failure)
	}
	if _, failure := ValidateResultLimit(0, true, 10); failure == nil || failure.Code != protocol.CodePayloadLimit {
		t.Fatalf("zero result limit = %+v", failure)
	}
	if _, failure := ValidateResultLimit(51, true, 10); failure == nil || failure.Code != protocol.CodePayloadLimit {
		t.Fatalf("high result limit = %+v", failure)
	}
	if limit, failure := ValidateResultLimit(50, true, 10); failure != nil || limit != 50 {
		t.Fatalf("bound result limit = %d, %+v", limit, failure)
	}
	if failure := ValidateOperatingTabCount(OperatingTabLimit + 1); failure == nil || failure.LimitName != "operating_tabs" {
		t.Fatalf("operating tab limit = %+v", failure)
	}
	if failure := ValidateDeltaOperationCount(limits.MaxDeltaOperations+1, limits); failure == nil || failure.LimitName != "delta_operations" {
		t.Fatalf("delta limit = %+v", failure)
	}
	longQuery := protocol.Query{Text: strings.Repeat("a", limits.MaxQueryScalars+1)}
	if failure := ValidateQuery(longQuery, limits); failure == nil || failure.Code != protocol.CodePayloadLimit {
		t.Fatalf("query limit = %+v", failure)
	}
	negativeWindow := protocol.Query{Text: "go", CurrentWindowID: -1}
	if failure := ValidateQuery(negativeWindow, limits); failure == nil || failure.Code != protocol.CodeInvalidFrame {
		t.Fatalf("negative window = %+v", failure)
	}
	validQuery := protocol.Query{Text: strings.Repeat("a", limits.MaxQueryScalars), CurrentWindowID: 1}
	if failure := ValidateQuery(validQuery, limits); failure != nil {
		t.Fatalf("valid query = %+v", failure)
	}
	malformed := protocol.TabRecord{
		ProfileID:          scopeProfileA,
		ContextKind:        "normal",
		TabID:              1,
		WindowID:           1,
		TitleDisplay:       "Example",
		TitleSearch:        "example",
		URLDisplay:         "not a url",
		URLSearch:          "not a url",
		DomainDisplay:      "example.test",
		DomainSearch:       "example.test",
		Eligible:           true,
		ProjectionEpoch:    scopeEpoch,
		ProjectionRevision: 1,
	}
	if failure := ValidateRecord(malformed, limits); failure == nil || failure.Code != protocol.CodeInvalidFrame {
		t.Fatalf("malformed URL = %+v", failure)
	}
}
