package privacy

import (
	"regexp"
	"strings"
)

// SecretKind is the closed secret-shape vocabulary. Findings never carry the
// original value.
type SecretKind string

const (
	SecretKindAuthHeader  SecretKind = "auth_header"
	SecretKindCookie      SecretKind = "cookie"
	SecretKindToken       SecretKind = "token"
	SecretKindPrivateKey  SecretKind = "private_key"
	SecretKindCredential  SecretKind = "credential"
	SecretKindEnvironment SecretKind = "environment_secret"
)

// SecretFinding names the field and secret shape only.
type SecretFinding struct {
	Field string
	Kind  SecretKind
}

var secretFieldNames = map[string]SecretKind{
	"authorization":       SecretKindAuthHeader,
	"auth_header":         SecretKindAuthHeader,
	"proxy-authorization": SecretKindAuthHeader,
	"cookie":              SecretKindCookie,
	"set-cookie":          SecretKindCookie,
	"token":               SecretKindToken,
	"access_token":        SecretKindToken,
	"refresh_token":       SecretKindToken,
	"session_token":       SecretKindToken,
	"secret":              SecretKindCredential,
	"password":            SecretKindCredential,
	"credential":          SecretKindCredential,
	"api_key":             SecretKindCredential,
	"apikey":              SecretKindCredential,
	"private_key":         SecretKindPrivateKey,
	"environment_secret":  SecretKindEnvironment,
	"env_secret":          SecretKindEnvironment,
}

var (
	authHeaderValue = regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]{8,}`)
	jwtValue        = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`)
	knownTokenValue = regexp.MustCompile(`\b(?:ghp_|github_pat_|sk-|AKIA)[A-Za-z0-9_-]{8,}\b`)
	pemPrivateKey   = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)
	passwordPair    = regexp.MustCompile(`(?i)\b(?:password|passwd|token|secret)=[^&\s]+`)
)

// DetectSecret reports a field/value secret shape. The value is never returned.
func DetectSecret(field string, value any) []SecretFinding {
	findings := make([]SecretFinding, 0, 2)
	if kind, ok := secretFieldNames[strings.ToLower(field)]; ok {
		findings = append(findings, SecretFinding{Field: field, Kind: kind})
	}
	text, ok := value.(string)
	if !ok {
		return findings
	}
	switch {
	case authHeaderValue.MatchString(text):
		findings = append(findings, SecretFinding{Field: field, Kind: SecretKindAuthHeader})
	case jwtValue.MatchString(text), knownTokenValue.MatchString(text):
		findings = append(findings, SecretFinding{Field: field, Kind: SecretKindToken})
	case pemPrivateKey.MatchString(text):
		findings = append(findings, SecretFinding{Field: field, Kind: SecretKindPrivateKey})
	case passwordPair.MatchString(text):
		findings = append(findings, SecretFinding{Field: field, Kind: SecretKindCredential})
	}
	return findings
}

// RejectSecrets returns every secret finding in a bounded map.
func RejectSecrets(fields map[string]any) []SecretFinding {
	findings := make([]SecretFinding, 0)
	for field, value := range fields {
		findings = append(findings, DetectSecret(field, value)...)
	}
	return findings
}

// ContainsSecretShapedValue reports whether a string matches a high-confidence
// secret shape. It never returns or logs the value.
func ContainsSecretShapedValue(value string) bool {
	return authHeaderValue.MatchString(value) ||
		jwtValue.MatchString(value) ||
		knownTokenValue.MatchString(value) ||
		pemPrivateKey.MatchString(value) ||
		passwordPair.MatchString(value)
}

// SecretFindingsSafe reports only field/kind pairs for audit output.
func SecretFindingsSafe(findings []SecretFinding) []map[string]string {
	safe := make([]map[string]string, 0, len(findings))
	for _, finding := range findings {
		safe = append(safe, map[string]string{"field": finding.Field, "kind": string(finding.Kind)})
	}
	return safe
}
