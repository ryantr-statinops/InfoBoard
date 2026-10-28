package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func mustManifest(t *testing.T, root string, entries []Entry) Manifest {
	t.Helper()
	manifest, err := NewManifest(root, entries)
	if err != nil {
		t.Fatalf("new manifest: %v", err)
	}
	return manifest
}

func TestNewManifestRejectsUnsafeEntries(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name    string
		root    string
		entries []Entry
	}{
		{"relative root", "data", []Entry{{ID: "db", RelativePath: "db.sqlite", Kind: KindFile}}},
		{"volume root", filepath.VolumeName(root) + string(filepath.Separator), []Entry{{ID: "db", RelativePath: "db.sqlite", Kind: KindFile}}},
		{"no entries", root, nil},
		{"empty id", root, []Entry{{ID: "", RelativePath: "db.sqlite", Kind: KindFile}}},
		{"bad id", root, []Entry{{ID: "bad/id", RelativePath: "db.sqlite", Kind: KindFile}}},
		{"absolute entry", root, []Entry{{ID: "db", RelativePath: filepath.Join(root, "db.sqlite"), Kind: KindFile}}},
		{"parent entry", root, []Entry{{ID: "db", RelativePath: "../db.sqlite", Kind: KindFile}}},
		{"nested parent", root, []Entry{{ID: "db", RelativePath: "data/../../db.sqlite", Kind: KindFile}}},
		{"root entry", root, []Entry{{ID: "db", RelativePath: ".", Kind: KindDirectory}}},
		{"wildcard", root, []Entry{{ID: "db", RelativePath: "*.sqlite", Kind: KindFile}}},
		{"bad kind", root, []Entry{{ID: "db", RelativePath: "db.sqlite", Kind: "socket"}}},
		{"duplicate id", root, []Entry{{ID: "db", RelativePath: "a.sqlite", Kind: KindFile}, {ID: "db", RelativePath: "b.sqlite", Kind: KindFile}}},
		{"duplicate path", root, []Entry{{ID: "a", RelativePath: "db.sqlite", Kind: KindFile}, {ID: "b", RelativePath: "db.sqlite", Kind: KindFile}}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := NewManifest(testCase.root, testCase.entries); !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("expected invalid manifest, got %v", err)
			}
		})
	}
}

func TestCleanerRemovesOnlyManifestAndTreatsMissingAsSuccess(t *testing.T) {
	root := t.TempDir()
	unrelated := filepath.Join(root, "unrelated.txt")
	if err := os.WriteFile(unrelated, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	database := filepath.Join(root, "data", "db.sqlite")
	if err := os.MkdirAll(filepath.Dir(database), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(database, []byte("db"), 0o600); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "cache")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "entry"), []byte("cache"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := mustManifest(t, root, []Entry{
		{ID: "database", RelativePath: "data/db.sqlite", Kind: KindFile},
		{ID: "cache", RelativePath: "cache", Kind: KindDirectory},
		{ID: "missing", RelativePath: "missing.bin", Kind: KindFile},
	})
	cleaner, err := NewCleaner(manifest)
	if err != nil {
		t.Fatal(err)
	}
	result, err := cleaner.Remove(context.Background())
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !result.Verified || result.Retryable || len(result.Removed) != 2 || len(result.Missing) != 1 || len(result.Failed) != 0 {
		t.Fatalf("cleanup result = %+v", result)
	}
	for _, path := range []string{database, cache} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s remains: %v", path, err)
		}
	}
	if contents, err := os.ReadFile(unrelated); err != nil || string(contents) != "keep" {
		t.Fatalf("unrelated file changed: %q err=%v", contents, err)
	}
}

func TestCleanerRefusesSymlinkTarget(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "owned-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	manifest := mustManifest(t, root, []Entry{{ID: "link", RelativePath: "owned-link", Kind: KindFile}})
	cleaner, err := NewCleaner(manifest)
	if err != nil {
		t.Fatal(err)
	}
	result, err := cleaner.Remove(context.Background())
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if result.Verified || !result.Retryable || result.ErrorClass != ErrorPathSafety || len(result.Failed) != 1 {
		t.Fatalf("symlink result = %+v", result)
	}
	if contents, err := os.ReadFile(outside); err != nil || string(contents) != "outside" {
		t.Fatalf("outside target changed: %q err=%v", contents, err)
	}
}

func TestCleanerHonorsCancellation(t *testing.T) {
	root := t.TempDir()
	manifest := mustManifest(t, root, []Entry{{ID: "missing", RelativePath: "missing.bin", Kind: KindFile}})
	cleaner, err := NewCleaner(manifest)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cleaner.Remove(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestDefaultEntriesStayInsideRoot(t *testing.T) {
	root := t.TempDir()
	manifest := mustManifest(t, root, DefaultEntries())
	if len(manifest.Entries()) != len(DefaultEntries()) {
		t.Fatalf("default entries changed: %v", manifest.Entries())
	}
	for _, target := range manifest.Targets() {
		relative, err := filepath.Rel(root, target)
		if err != nil || relative == ".." || filepath.IsAbs(relative) {
			t.Fatalf("default target escaped root: %s", target)
		}
	}
}
