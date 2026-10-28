package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

const (
	ErrorCleanupFailed = "cleanup_failed"
	ErrorPathSafety    = "path_safety"
)

// CleanupResult reports reviewed entry IDs, never absolute paths. Missing
// entries are already-complete outcomes; failed entries are retryable.
type CleanupResult struct {
	Removed    []string
	Missing    []string
	Failed     []string
	Verified   bool
	Retryable  bool
	ErrorClass string
}

// Cleaner removes exactly one validated manifest. It refuses symlinks and
// non-regular files, and never follows a link while removing a directory tree.
type Cleaner struct {
	manifest Manifest
}

// NewCleaner validates that the manifest came from NewManifest and has at
// least one reviewed entry.
func NewCleaner(manifest Manifest) (*Cleaner, error) {
	if manifest.root == "" || len(manifest.entries) == 0 {
		return nil, ErrInvalidManifest
	}
	return &Cleaner{manifest: manifest}, nil
}

// Remove deletes the manifest targets and verifies their absence. It returns a
// renderable result for path/permission failures; cancellation is returned as
// an error after the partial result is populated.
func (cleaner *Cleaner) Remove(ctx context.Context) (CleanupResult, error) {
	if cleaner == nil {
		return CleanupResult{}, ErrInvalidManifest
	}
	if ctx == nil {
		ctx = context.Background()
	}
	result := CleanupResult{}
	for _, entry := range cleaner.manifest.entries {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		target := filepath.Join(cleaner.manifest.root, entry.RelativePath)
		info, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			result.Missing = append(result.Missing, entry.ID)
			continue
		}
		if err != nil {
			markFailure(&result, entry.ID, ErrorCleanupFailed)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			markFailure(&result, entry.ID, ErrorPathSafety)
			continue
		}
		switch entry.Kind {
		case KindFile:
			if !info.Mode().IsRegular() {
				markFailure(&result, entry.ID, ErrorPathSafety)
				continue
			}
			if err := os.Remove(target); err != nil {
				markFailure(&result, entry.ID, ErrorCleanupFailed)
				continue
			}
		case KindDirectory:
			if !info.IsDir() {
				markFailure(&result, entry.ID, ErrorPathSafety)
				continue
			}
			if err := removeTree(target); err != nil {
				class := ErrorCleanupFailed
				if errors.Is(err, ErrUnsafePath) {
					class = ErrorPathSafety
				}
				markFailure(&result, entry.ID, class)
				continue
			}
		default:
			markFailure(&result, entry.ID, ErrorPathSafety)
			continue
		}
		result.Removed = append(result.Removed, entry.ID)
	}
	result.Verified = len(result.Failed) == 0
	result.Retryable = len(result.Failed) > 0
	return result, nil
}

func markFailure(result *CleanupResult, id, class string) {
	result.Failed = append(result.Failed, id)
	if result.ErrorClass != ErrorPathSafety {
		result.ErrorClass = class
	}
}

func removeTree(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return ErrUnsafePath
	}
	if !info.IsDir() {
		if !info.Mode().IsRegular() {
			return ErrUnsafePath
		}
		return os.Remove(path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := removeTree(filepath.Join(path, entry.Name())); err != nil {
			return err
		}
	}
	return os.Remove(path)
}
