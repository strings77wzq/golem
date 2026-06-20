package helpers

import (
	"os"
	"os/exec"
	"testing"
)

func TestBuildBinary_ProducesExecutable(t *testing.T) {
	binPath := BuildGolem(t)

	// Binary exists
	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("binary not found at %s: %v", binPath, err)
	}
	if info.IsDir() {
		t.Fatalf("expected a file, got directory at %s", binPath)
	}

	// Binary is executable
	if info.Mode().Perm()&0100 == 0 {
		t.Fatalf("binary at %s is not executable", binPath)
	}

	// --version exits 0
	cmd := exec.Command(binPath, "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("golem --version failed: %v\n%s", err, out)
	}
	if len(out) == 0 {
		t.Fatal("golem --version produced no output")
	}
}
