// Package lifecycle owns the fixed InfoBoard cleanup manifest and the
// path-safe removal adapter used by uninstall. It never derives deletion
// targets from request input.
package lifecycle

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidManifest = errors.New("invalid lifecycle manifest")
	ErrUnsafePath      = errors.New("unsafe lifecycle path")
)

// EntryKind is the expected filesystem type of one owned artifact.
type EntryKind string

const (
	KindFile      EntryKind = "file"
	KindDirectory EntryKind = "directory"
)

// Entry is one reviewed, root-relative artifact.
type Entry struct {
	ID           string
	RelativePath string
	Kind         EntryKind
}

// Manifest is a validated root plus root-relative owned entries. The fields are
// intentionally private so callers must pass through NewManifest validation.
type Manifest struct {
	root    string
	entries []Entry
}

// DefaultEntries returns the database artifacts known to the host. Platform
// install manifests and binaries are added by the packaging owner when their
// locations are frozen; they are not inferred here.
func DefaultEntries() []Entry {
	return []Entry{
		{ID: "database", RelativePath: "infoboard.sqlite", Kind: KindFile},
		{ID: "database-journal", RelativePath: "infoboard.sqlite-journal", Kind: KindFile},
		{ID: "database-wal", RelativePath: "infoboard.sqlite-wal", Kind: KindFile},
		{ID: "database-shm", RelativePath: "infoboard.sqlite-shm", Kind: KindFile},
	}
}

// NewManifest validates an absolute trusted root and root-relative entries.
// Empty, absolute, parent-traversing, wildcard, duplicate, and root-level
// targets are rejected before any cleanup can run.
func NewManifest(root string, entries []Entry) (Manifest, error) {
	if root == "" || !filepath.IsAbs(root) {
		return Manifest{}, fmt.Errorf("%w: root must be absolute", ErrInvalidManifest)
	}
	root = filepath.Clean(root)
	if filepath.Dir(root) == root {
		return Manifest{}, fmt.Errorf("%w: root must not be a volume root", ErrInvalidManifest)
	}
	if len(entries) == 0 {
		return Manifest{}, fmt.Errorf("%w: at least one entry is required", ErrInvalidManifest)
	}

	validated := make([]Entry, 0, len(entries))
	ids := make(map[string]struct{}, len(entries))
	paths := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !validID(entry.ID) {
			return Manifest{}, fmt.Errorf("%w: invalid entry id", ErrInvalidManifest)
		}
		if _, exists := ids[entry.ID]; exists {
			return Manifest{}, fmt.Errorf("%w: duplicate entry id %q", ErrInvalidManifest, entry.ID)
		}
		ids[entry.ID] = struct{}{}
		if entry.Kind != KindFile && entry.Kind != KindDirectory {
			return Manifest{}, fmt.Errorf("%w: invalid entry kind", ErrInvalidManifest)
		}
		if entry.RelativePath == "" ||
			filepath.IsAbs(entry.RelativePath) ||
			filepath.VolumeName(entry.RelativePath) != "" ||
			strings.ContainsAny(entry.RelativePath, "*?[") {
			return Manifest{}, fmt.Errorf("%w: invalid relative path", ErrInvalidManifest)
		}
		cleaned := filepath.Clean(entry.RelativePath)
		if filepath.ToSlash(cleaned) != filepath.ToSlash(entry.RelativePath) ||
			cleaned == "." ||
			cleaned == ".." ||
			strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
			return Manifest{}, fmt.Errorf("%w: path escapes root", ErrInvalidManifest)
		}
		if _, exists := paths[cleaned]; exists {
			return Manifest{}, fmt.Errorf("%w: duplicate relative path", ErrInvalidManifest)
		}
		paths[cleaned] = struct{}{}
		target := filepath.Join(root, cleaned)
		relative, err := filepath.Rel(root, target)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return Manifest{}, fmt.Errorf("%w: target escapes root", ErrInvalidManifest)
		}
		validated = append(validated, Entry{ID: entry.ID, RelativePath: cleaned, Kind: entry.Kind})
	}
	return Manifest{root: root, entries: validated}, nil
}

func validID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, character := range id {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '.' || character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

// Root returns the validated absolute root.
func (manifest Manifest) Root() string { return manifest.root }

// Entries returns a detached copy of the reviewed entries.
func (manifest Manifest) Entries() []Entry {
	return append([]Entry(nil), manifest.entries...)
}

// Targets returns the absolute targets in manifest order.
func (manifest Manifest) Targets() []string {
	targets := make([]string, 0, len(manifest.entries))
	for _, entry := range manifest.entries {
		targets = append(targets, filepath.Join(manifest.root, entry.RelativePath))
	}
	return targets
}
