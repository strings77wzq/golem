// Package architecture provides tests that enforce architectural constraints.
package architecture

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLayerDependencies enforces AGENTS.md §3 layer dependency rules.
// This test scans all .go files and verifies no forbidden imports exist.
//
// Rules:
//   - foundation/ → stdlib ONLY (exception: mattn/go-isatty)
//   - core/ → foundation/ ONLY (never internal/, feature/, cmd/)
//   - feature/ → core/ + foundation/ ONLY (never internal/, cmd/)
//   - internal/ → core/ + foundation/ ONLY (never feature/, cmd/)
//   - cmd/ → imports ALL layers (composition root)
func TestLayerDependencies(t *testing.T) {
	root := findProjectRoot(t)

	rules := []struct {
		name          string
		dir           string
		forbidden     []string
		allowedExcept []string // exceptions to forbidden patterns
	}{
		{
			name:          "foundation_must_not_import_project",
			dir:           "foundation",
			forbidden:     []string{"github.com/strings77wzq/golem/core/", "github.com/strings77wzq/golem/internal/", "github.com/strings77wzq/golem/feature/", "github.com/strings77wzq/golem/cmd/"},
			allowedExcept: []string{"github.com/strings77wzq/golem/foundation/"},
		},
		{
			name:      "core_must_not_import_internal_feature_cmd",
			dir:       "core",
			forbidden: []string{"github.com/strings77wzq/golem/internal/", "github.com/strings77wzq/golem/feature/", "github.com/strings77wzq/golem/cmd/"},
		},
		{
			name:      "feature_must_not_import_internal_cmd",
			dir:       "feature",
			forbidden: []string{"github.com/strings77wzq/golem/internal/", "github.com/strings77wzq/golem/cmd/"},
		},
		{
			name:      "internal_must_not_import_feature_cmd",
			dir:       "internal",
			forbidden: []string{"github.com/strings77wzq/golem/feature/", "github.com/strings77wzq/golem/cmd/"},
		},
	}

	for _, rule := range rules {
		t.Run(rule.name, func(t *testing.T) {
			dirPath := filepath.Join(root, rule.dir)
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				t.Skipf("directory %s does not exist", rule.dir)
			}

			violations := scanImports(t, dirPath, rule.forbidden, rule.allowedExcept)
			if len(violations) > 0 {
				t.Errorf("layer dependency violations found:\n%s", strings.Join(violations, "\n"))
			}
		})
	}
}

// TestFoundationOnlyImportsStdlib verifies foundation/ only imports stdlib + mattn/go-isatty.
func TestFoundationOnlyImportsStdlib(t *testing.T) {
	root := findProjectRoot(t)
	dirPath := filepath.Join(root, "foundation")

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Skip("foundation directory does not exist")
	}

	// Allowed non-stdlib imports (documented exceptions)
	allowed := map[string]bool{
		"github.com/mattn/go-isatty": true, // TTY detection in term/
		"modernc.org/sqlite":         true, // SQLite driver in store/
	}

	violations := []string{}
	fset := token.NewFileSet()

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return nil // skip unparseable files
		}

		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			// Allow stdlib (no dots in first segment) and allowed exceptions
			if isStdlib(importPath) || allowed[importPath] {
				continue
			}
			relPath, _ := filepath.Rel(root, path)
			violations = append(violations,
				"  "+relPath+" imports "+importPath+" (foundation must only import stdlib + mattn/go-isatty)")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk foundation directory: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("foundation layer violations:\n%s", strings.Join(violations, "\n"))
	}
}

func scanImports(t *testing.T, dirPath string, forbidden []string, allowedExcept []string) []string {
	t.Helper()
	violations := []string{}
	fset := token.NewFileSet()

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return nil
		}

		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			// Check if this import is in the allowed exceptions
			isAllowed := false
			for _, exc := range allowedExcept {
				if strings.HasPrefix(importPath, exc) {
					isAllowed = true
					break
				}
			}
			if isAllowed {
				continue
			}

			// Check if this import matches any forbidden pattern
			for _, forbiddenPrefix := range forbidden {
				if strings.HasPrefix(importPath, forbiddenPrefix) {
					relPath, _ := filepath.Rel(filepath.Dir(dirPath), path)
					violations = append(violations,
						"  "+relPath+" imports "+importPath+" (forbidden: "+forbiddenPrefix+"...)")
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk directory %s: %v", dirPath, err)
	}

	return violations
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from current directory to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

func isStdlib(importPath string) bool {
	// Stdlib packages don't have dots in the first path segment
	firstSlash := strings.Index(importPath, "/")
	firstSegment := importPath
	if firstSlash > 0 {
		firstSegment = importPath[:firstSlash]
	}
	return !strings.Contains(firstSegment, ".")
}
