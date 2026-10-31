package privacy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProfilePermissionMatrixIsComplete(t *testing.T) {
	data := privacyFixture(t, "permission-matrix.json")
	var fixture struct {
		Matrix []PermissionMatrixRow `json:"matrix"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePermissionMatrix(fixture.Matrix); err != nil {
		t.Fatalf("permission matrix: %v", err)
	}
	if len(fixture.Matrix) != 8 {
		t.Fatalf("capability count = %d, want 8", len(fixture.Matrix))
	}
}

func TestProfileManifestAuditAcceptsApprovedPackage(t *testing.T) {
	path := filepath.Join("..", "..", "extension", "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	audit := AuditManifestJSON(data)
	if !audit.OK {
		t.Fatalf("approved manifest findings: %v", audit.Findings)
	}
	if len(audit.Permissions) != 4 {
		t.Fatalf("permissions = %v", audit.Permissions)
	}
}

func TestProfileManifestAuditRejectsPermissionCreep(t *testing.T) {
	audit := AuditManifestJSON(privacyFixture(t, "manifest-denied.json"))
	if audit.OK {
		t.Fatal("denied manifest unexpectedly passed")
	}
	want := map[string]bool{
		"permissions_not_approved_set": true,
		"denied_permission:history":    true,
		"denied_permission:cookies":    true,
		"denied_permission:scripting":  true,
		"host_permissions_present":     true,
		"optional_permissions_present": true,
		"broad_host_pattern":           true,
		"content_scripts_present":      true,
		"commands_not_approved_set":    true,
	}
	for _, finding := range audit.Findings {
		delete(want, finding)
	}
	if len(want) != 0 {
		t.Fatalf("missing findings: %v (got %v)", want, audit.Findings)
	}
}
