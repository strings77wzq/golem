package helpers

import (
	"strings"
	"testing"
)

// TestRedact_BearerToken verifies the secret scrubber substitutes a
// Bearer-style API token with the literal "[REDACTED]". This is the
// minimum guarantee the E2E suite makes about transcript artefacts:
// uploading a transcript MUST never leak a long-lived credential.
func TestRedact_BearerToken(t *testing.T) {
	in := "Authorization: Bearer sk-abcdefghijklmnopqrstuvwxyz1234"
	out := Redact(in)

	if strings.Contains(out, "sk-abcdefghijklmnopqrstuvwxyz1234") {
		t.Fatalf("Redact: token leaked into output: %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("Redact: expected output to contain [REDACTED], got %q", out)
	}
	// Surrounding context (the "Authorization:" prefix) MUST be preserved
	// so that operators can still see WHICH header was redacted.
	if !strings.Contains(out, "Authorization:") {
		t.Fatalf("Redact: dropped surrounding context, got %q", out)
	}
}

// TestRedact_DropUncertainLine verifies the conservative drop heuristic:
// a line with a 60+ char unbroken alphanumeric run and no recognised
// structured-log prefix is dropped entirely. This catches secrets we
// forgot to add a pattern for.
func TestRedact_DropUncertainLine(t *testing.T) {
	long := strings.Repeat("a", 60) + "1234567890"
	in := "got token blob: " + long + " trailing"

	out := Redact(in)
	if out != "" {
		t.Fatalf("Redact: expected empty string for uncertain long-run line, got %q", out)
	}
}

// TestRedact_KeepsStructuredLog verifies the drop heuristic does NOT
// fire on lines we recognise as structured Golem output, even if they
// contain a long alphanumeric run. Otherwise we would silently lose
// useful diagnostic data in transcripts.
func TestRedact_KeepsStructuredLog(t *testing.T) {
	long := strings.Repeat("z", 70)
	in := "event=tool_call args=" + long
	// Note: the long run will be partially eaten by rxKVSecret because
	// "args=..." matches the kv pattern; we want to assert the line
	// itself is not dropped wholesale.
	out := Redact(in)
	if out == "" {
		t.Fatalf("Redact: structured-log line MUST NOT be dropped wholesale; got empty")
	}
	if !strings.HasPrefix(out, "event=tool_call") {
		t.Fatalf("Redact: lost structured-log prefix; out=%q", out)
	}
}

// TestRedact_OpenAIKey verifies "sk-..." style keys (without a Bearer
// prefix) are still caught.
func TestRedact_OpenAIKey(t *testing.T) {
	in := "config: sk-abcdefghijklmnopqrstuvwxyz0123456789"
	out := Redact(in)
	if strings.Contains(out, "sk-abcdefghijklmnopqrstuvwxyz0123456789") {
		t.Fatalf("Redact: sk- key leaked, out=%q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("Redact: expected [REDACTED] marker, out=%q", out)
	}
}

// TestRedact_SafeShortLineUnchanged verifies short, ordinary lines pass
// through unchanged so transcripts remain readable.
func TestRedact_SafeShortLineUnchanged(t *testing.T) {
	in := "running tool sql_query rows=5"
	out := Redact(in)
	if out != in {
		t.Fatalf("Redact: short safe line was modified: in=%q out=%q", in, out)
	}
}
