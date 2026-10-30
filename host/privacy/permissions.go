package privacy

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ApprovedManifestPermissions is the only permission set the extension package
// may request. The order is normalized before comparison.
var ApprovedManifestPermissions = []string{"nativeMessaging", "storage", "tabGroups", "tabs"}

// ApprovedCommands is the only command set the extension package may declare.
var ApprovedCommands = []string{"close-search", "open-search"}

// DeniedManifestPermissions are outside the product boundary and must never be
// requested, even when a browser would allow them.
var DeniedManifestPermissions = []string{
	"bookmarks", "browsingData", "contentSettings", "cookies", "debugger",
	"declarativeNetRequest", "downloads", "history", "management", "privacy",
	"proxy", "scripting", "sessions", "topSites", "webNavigation", "webRequest",
}

// ManifestAudit is the safe result of a static manifest inspection. Findings
// contain policy identifiers, never raw manifest values.
type ManifestAudit struct {
	OK          bool
	Findings    []string
	Permissions []string
	Commands    []string
}

type manifestShape struct {
	ManifestVersion        int                 `json:"manifest_version"`
	Permissions            []string            `json:"permissions"`
	HostPermissions        []string            `json:"host_permissions"`
	OptionalPermissions    []string            `json:"optional_permissions"`
	OptionalHostPermission []string            `json:"optional_host_permissions"`
	ContentScripts         []json.RawMessage   `json:"content_scripts"`
	WebAccessibleResources []json.RawMessage   `json:"web_accessible_resources"`
	Commands               map[string]struct{} `json:"commands"`
}

// AuditManifestJSON performs the static permission audit for one manifest.
func AuditManifestJSON(data []byte) ManifestAudit {
	var manifest manifestShape
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ManifestAudit{Findings: []string{"manifest_not_object"}}
	}
	findings := make([]string, 0)
	if manifest.ManifestVersion != 3 {
		findings = append(findings, "manifest_version_not_3")
	}
	permissions := append([]string(nil), manifest.Permissions...)
	if !sameStrings(permissions, ApprovedManifestPermissions) {
		findings = append(findings, "permissions_not_approved_set")
	}
	for _, denied := range DeniedManifestPermissions {
		for _, permission := range permissions {
			if permission == denied {
				findings = append(findings, "denied_permission:"+denied)
			}
		}
	}
	if len(manifest.HostPermissions) > 0 {
		findings = append(findings, "host_permissions_present")
	}
	if len(manifest.OptionalPermissions) > 0 {
		findings = append(findings, "optional_permissions_present")
	}
	if len(manifest.OptionalHostPermission) > 0 {
		findings = append(findings, "optional_host_permissions_present")
	}
	for _, pattern := range append(append([]string(nil), manifest.HostPermissions...), manifest.OptionalHostPermission...) {
		if pattern == "<all_urls>" || containsWildcard(pattern) {
			findings = append(findings, "broad_host_pattern")
		}
	}
	if len(manifest.ContentScripts) > 0 {
		findings = append(findings, "content_scripts_present")
	}
	if len(manifest.WebAccessibleResources) > 0 {
		findings = append(findings, "web_accessible_resources_present")
	}
	commands := make([]string, 0, len(manifest.Commands))
	for command := range manifest.Commands {
		commands = append(commands, command)
	}
	if !sameStrings(commands, ApprovedCommands) {
		findings = append(findings, "commands_not_approved_set")
	}
	sort.Strings(permissions)
	sort.Strings(commands)
	return ManifestAudit{
		OK:          len(findings) == 0,
		Findings:    findings,
		Permissions: permissions,
		Commands:    commands,
	}
}

func containsWildcard(value string) bool {
	for _, r := range value {
		if r == '*' {
			return true
		}
	}
	return false
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}
	return true
}

// PermissionMatrixJSON is a stable machine-readable matrix row set. The fixture
// is the source of truth; this helper only validates that every row has an owner
// and at least one requirement mapping.
type PermissionMatrixRow struct {
	Capability         string   `json:"capability"`
	ProductUse         string   `json:"product_use"`
	Owner              string   `json:"owner"`
	RequirementIDs     []string `json:"requirement_ids"`
	ManifestPermission *string  `json:"manifest_permission"`
	DenialBehavior     string   `json:"denial_behavior"`
	Evidence           string   `json:"evidence"`
}

// ValidatePermissionMatrix rejects an incomplete or duplicated capability map.
func ValidatePermissionMatrix(rows []PermissionMatrixRow) error {
	if len(rows) == 0 {
		return fmt.Errorf("privacy: empty permission matrix")
	}
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row.Capability == "" || row.ProductUse == "" || row.Owner == "" ||
			len(row.RequirementIDs) == 0 || row.DenialBehavior == "" || row.Evidence == "" {
			return fmt.Errorf("privacy: incomplete permission row %q", row.Capability)
		}
		if seen[row.Capability] {
			return fmt.Errorf("privacy: duplicate capability %q", row.Capability)
		}
		seen[row.Capability] = true
	}
	return nil
}
