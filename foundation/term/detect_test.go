package term_test

import (
	"testing"

	"github.com/strings77wzq/golem/foundation/term"
)

func TestIsPiped(t *testing.T) {
	// In test environment, stdin/stdout are typically not TTYs
	// so IsPiped() should return true
	result := term.IsPiped()
	if !result {
		t.Log("IsPiped returned false; test is likely running in a TTY")
	}
}

func TestIsInputTTY(t *testing.T) {
	// Just verify it doesn't panic and returns a bool
	_ = term.IsInputTTY()
}

func TestIsOutputTTY(t *testing.T) {
	_ = term.IsOutputTTY()
}

func TestReadStdin_WhenTTY(t *testing.T) {
	if !term.IsInputTTY() {
		t.Skip("stdin is not a TTY in this test environment")
	}
	s, err := term.ReadStdin()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "" {
		t.Fatalf("expected empty string from TTY stdin, got %q", s)
	}
}

func TestIsPipedConsistency(t *testing.T) {
	// IsPiped should be consistent with IsInputTTY and IsOutputTTY
	piped := term.IsPiped()
	inputTTY := term.IsInputTTY()
	outputTTY := term.IsOutputTTY()

	// IsPiped returns true when either is not a TTY
	expected := !inputTTY || !outputTTY
	if piped != expected {
		t.Errorf("IsPiped=%v but IsInputTTY=%v, IsOutputTTY=%v (expected %v)",
			piped, inputTTY, outputTTY, expected)
	}
}

func TestIsInputTTYIdempotent(t *testing.T) {
	// Call multiple times, should return same value (sync.OnceValue)
	v1 := term.IsInputTTY()
	v2 := term.IsInputTTY()
	v3 := term.IsInputTTY()

	if v1 != v2 || v2 != v3 {
		t.Errorf("IsInputTTY not idempotent: %v, %v, %v", v1, v2, v3)
	}
}

func TestIsOutputTTYIdempotent(t *testing.T) {
	v1 := term.IsOutputTTY()
	v2 := term.IsOutputTTY()
	v3 := term.IsOutputTTY()

	if v1 != v2 || v2 != v3 {
		t.Errorf("IsOutputTTY not idempotent: %v, %v, %v", v1, v2, v3)
	}
}

func TestReadStdinWhenPiped(t *testing.T) {
	if term.IsInputTTY() {
		t.Skip("stdin is a TTY in this test environment")
	}
	// When piped, ReadStdin should read from stdin
	// In test environment, stdin is typically empty or closed
	s, err := term.ReadStdin()
	// We don't check the value because stdin state varies
	// Just verify it doesn't panic
	_ = s
	_ = err
}
