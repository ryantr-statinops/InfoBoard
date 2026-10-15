package runtime

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// allowedHealthKeys is the closed set of JSON keys a diagnostic export may
// contain. T10 requires health to expose only counts, durations, versions,
// revisions, and error classes: no title, URL, query, token, SQL, or path can
// be named by these keys.
var allowedHealthKeys = map[string]struct{}{
	"uptime_ms": {}, "session_id": {}, "level": {}, "event": {},
	"state": {}, "availability": {}, "failure": {}, "retryable": {},
	"duration_ms": {}, "count": {}, "host_version": {}, "protocol_version": {},
	"persistence": {}, "transitions": {}, "dropped": {}, "projection_count": {},
}

// excludedContentSentinels mirrors the storage-layer T10 exclusion classes.
// NFR-009 requires that none of these appear in diagnostic exports; the
// runtime Diagnostic struct carries them structurally absent, and this test
// pins that property against the marshalled export form.
var excludedContentSentinels = []string{
	"My Secret Tab Title",
	"https://example.test/search?q=private+query",
	"SELECT * FROM diagnostics",
	"session_token=abc123-secret",
	"/home/user/.config/infoboard.sqlite",
}

func TestDiagnosticExportContainsOnlyBoundedKeys(t *testing.T) {
	diags := NewDiagnostics(NewBoundedSink(io.Discard, 4096), 8, 4096)
	snapshot := HealthSnapshot{
		SessionID: "0123456789abcdef0123456789abcdef", State: StateReady,
		Availability: AvailabilityHealthy, HostVersion: "0.1.0",
		ProtocolVersion: 1, Persistence: PersistenceReady,
		ProjectionCount: 3, Transitions: 1,
	}
	if !diags.Observe(snapshot, LevelInfo, EventPersistenceSettled, 3, 5) {
		t.Fatal("valid diagnostic observation rejected")
	}
	entries := diags.Entries()
	if len(entries) != 1 {
		t.Fatalf("diagnostic entries = %d, want 1", len(entries))
	}
	encoded, err := json.Marshal(entries[0])
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	for key := range decoded {
		if _, allowed := allowedHealthKeys[key]; !allowed {
			t.Fatalf("diagnostic export key %q is outside the bounded allowlist", key)
		}
	}
	for _, sentinel := range excludedContentSentinels {
		if strings.Contains(string(encoded), sentinel) {
			t.Fatalf("diagnostic export contains excluded content %q: %s", sentinel, encoded)
		}
	}
}
