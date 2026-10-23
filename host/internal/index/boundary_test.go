package index

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// allowedImportPrefixes is the closed dependency surface of the lexical index.
// net/url is parsing-only; there is no network client, SQL/storage, browser API,
// page evaluation, or process-execution dependency.
var allowedImportPrefixes = []string{
	"context",
	"crypto/",
	"encoding/",
	"fmt",
	"go/",
	"golang.org/x/text/",
	"github.com/ryantr-statinops/InfoBoard/host/internal/domain",
	"net/url",
	"reflect",
	"sort",
	"strings",
	"sync",
	"testing",
	"unicode",
	"unicode/utf8",
}

func TestLexicalPackageBoundary(t *testing.T) {
	packages, err := parser.ParseDir(token.NewFileSet(), ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}
	for name, pkg := range packages {
		if name != "index" {
			continue
		}
		for file, parsed := range pkg.Files {
			for _, spec := range parsed.Imports {
				path := strings.Trim(spec.Path.Value, `"`)
				if !allowedImport(path) {
					t.Fatalf("%s imports forbidden dependency %q", file, path)
				}
			}
		}
	}
}

func allowedImport(path string) bool {
	for _, prefix := range allowedImportPrefixes {
		if path == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
