// Package privacy owns the IP-17 policy boundary between browser metadata, the
// local protocol, the in-memory projection, SQLite, diagnostics, and exports.
//
// The package is a policy and validation layer. It consumes the IP-07 protocol,
// IP-08 storage, and IP-16 diagnostics seams rather than redefining their wire
// shapes, tables, or state machines.
package privacy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// FieldClass is the closed IP-17 data-class enum.
type FieldClass string

const (
	FieldClassTransientSearch     FieldClass = "transient_search"
	FieldClassBoundedDurable      FieldClass = "bounded_durable"
	FieldClassDiagnosticAllowlist FieldClass = "diagnostic_allowlist"
	FieldClassForbidden           FieldClass = "forbidden"
)

// FieldClasses is the complete field-class vocabulary.
var FieldClasses = []FieldClass{
	FieldClassTransientSearch,
	FieldClassBoundedDurable,
	FieldClassDiagnosticAllowlist,
	FieldClassForbidden,
}

var (
	ErrInvalidFieldCatalog = errors.New("privacy: invalid field catalog")
	ErrUnclassifiedField   = errors.New("privacy: unclassified field")
	ErrForbiddenField      = errors.New("privacy: forbidden field")
)

// FieldPolicy records the source, owner, boundary, storage class, retention,
// display policy, and redaction rule for one field.
type FieldPolicy struct {
	Field      string     `json:"field"`
	FieldClass FieldClass `json:"field_class"`
	Source     string     `json:"source"`
	Owner      string     `json:"owner"`
	Boundary   string     `json:"boundary"`
	Storage    string     `json:"storage"`
	Retention  string     `json:"retention"`
	Display    string     `json:"display"`
	Redaction  string     `json:"redaction"`
}

// FieldCatalog is an immutable lookup over a validated field matrix.
type FieldCatalog struct {
	byName map[string]FieldPolicy
}

// NewFieldCatalog validates a matrix before it can be used at a boundary.
func NewFieldCatalog(policies []FieldPolicy) (*FieldCatalog, error) {
	if len(policies) == 0 {
		return nil, fmt.Errorf("%w: empty matrix", ErrInvalidFieldCatalog)
	}
	validClasses := make(map[FieldClass]bool, len(FieldClasses))
	for _, class := range FieldClasses {
		validClasses[class] = true
	}
	byName := make(map[string]FieldPolicy, len(policies))
	for index, policy := range policies {
		if policy.Field == "" || !validClasses[policy.FieldClass] {
			return nil, fmt.Errorf("%w: entry %d has an invalid field or class", ErrInvalidFieldCatalog, index)
		}
		if policy.Source == "" || policy.Owner == "" || policy.Boundary == "" ||
			policy.Storage == "" || policy.Retention == "" || policy.Display == "" || policy.Redaction == "" {
			return nil, fmt.Errorf("%w: field %q is incomplete", ErrInvalidFieldCatalog, policy.Field)
		}
		if _, exists := byName[policy.Field]; exists {
			return nil, fmt.Errorf("%w: duplicate field %q", ErrInvalidFieldCatalog, policy.Field)
		}
		byName[policy.Field] = policy
	}
	return &FieldCatalog{byName: byName}, nil
}

// LoadFieldCatalogJSON decodes the contract fixture into an immutable catalog.
func LoadFieldCatalogJSON(data []byte) (*FieldCatalog, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var fixture struct {
		Fields []FieldPolicy `json:"fields"`
	}
	if err := decoder.Decode(&fixture); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFieldCatalog, err)
	}
	return NewFieldCatalog(fixture.Fields)
}

// Classify returns the policy for a known field.
func (catalog *FieldCatalog) Classify(field string) (FieldPolicy, bool) {
	if catalog == nil {
		return FieldPolicy{}, false
	}
	policy, ok := catalog.byName[field]
	return policy, ok
}

// IsForbidden reports whether a field is explicitly outside every boundary.
func (catalog *FieldCatalog) IsForbidden(field string) bool {
	policy, ok := catalog.Classify(field)
	return ok && policy.FieldClass == FieldClassForbidden
}

// AssertClassified rejects an unknown or forbidden field.
func (catalog *FieldCatalog) AssertClassified(field string) (FieldPolicy, error) {
	policy, ok := catalog.Classify(field)
	if !ok {
		return FieldPolicy{}, fmt.Errorf("%w: %s", ErrUnclassifiedField, field)
	}
	if policy.FieldClass == FieldClassForbidden {
		return FieldPolicy{}, fmt.Errorf("%w: %s", ErrForbiddenField, field)
	}
	return policy, nil
}

// Count returns the number of fields in the matrix.
func (catalog *FieldCatalog) Count() int {
	if catalog == nil {
		return 0
	}
	return len(catalog.byName)
}
