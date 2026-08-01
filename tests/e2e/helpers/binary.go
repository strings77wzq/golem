// Package helpers provides black-box-test utilities for the Golem E2E
// suite. It MUST NOT import any package whose path begins with
// "github.com/strings77wzq/golem/" outside this module — see the
// architectural rule documented in tests/e2e/doc.go.
package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// RepoRoot returns the absolute path to the Golem repository root.
// It walks up from the test file's directory until it finds a go.mod
// that does NOT declare module github.com/strings77wzq/golem/tests/e2e.
func RepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		gm := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(gm); err == nil {
			data, readErr := os.ReadFile(gm)
			if readErr == nil {
				content := string(data)
				// Skip the E2E module's own go.mod
				if !strings.Contains(content, "golem/tests/e2e") {
					return dir
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root (no go.mod found)")
		}
		dir = parent
	}
}

// BuildGolem compiles the production golem binary and returns its path.
// The binary is built with CGO_ENABLED=0 for Termux compatibility.
// Cleanup is registered via t.Cleanup.
//
// Package-level tests run in parallel processes (helpers and e2e), so the
// build is serialized with a flock and the already-built binary is reused:
// without this, two concurrent `go build -o` invocations race on the same
// output path and one process can observe a half-built/missing binary.
func BuildGolem(t *testing.T) string {
	t.Helper()
	root := RepoRoot(t)
	binDir := filepath.Join(root, "tests", "e2e", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	binPath := filepath.Join(binDir, "golem")

	// Serialize concurrent builds across test processes.
	lock, err := os.OpenFile(filepath.Join(binDir, ".build.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("open build lock: %v", err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatalf("acquire build lock: %v", err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) //nolint:errcheck

	// Another process may have built it while we waited for the lock.
	if _, err := os.Stat(binPath); err == nil {
		return binPath
	}

	cmd := exec.Command("go", "build", "-trimpath", "-o", binPath, "./cmd/golem")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	t.Cleanup(func() {
		os.Remove(binPath)
	})

	return binPath
}
