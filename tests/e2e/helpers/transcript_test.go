package helpers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTranscript_FileCreated verifies that calling New("foo") followed
// by Close() produces a file under the directory configured by the
// helper containing a header line of the form "# foo @ <RFC3339>".
//
// The transcript helper accepts an explicit directory so each test can
// isolate its output via t.TempDir(); production callers (the
// behavioural tests in tests/e2e/) use a default that lives under
// tests/e2e/transcripts/.
func TestTranscript_FileCreated(t *testing.T) {
	dir := t.TempDir()
	tr, err := New(dir, "foo")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	path := filepath.Join(dir, "foo.log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("transcript file missing at %s: %v", path, err)
	}
	contents := string(data)
	lines := strings.SplitN(contents, "\n", 2)
	header := lines[0]
	if !strings.HasPrefix(header, "# foo @ ") {
		t.Fatalf("transcript: expected header `# foo @ <RFC3339>`, got %q", header)
	}
	tsPart := strings.TrimPrefix(header, "# foo @ ")
	if _, err := time.Parse(time.RFC3339, tsPart); err != nil {
		t.Fatalf("transcript: header timestamp %q is not RFC3339: %v", tsPart, err)
	}
}

// TestTranscript_RedactsOnWrite verifies that everything written through
// Transcript.Write is piped through Redact line-by-line, so a Bearer
// token never reaches the on-disk artefact.
func TestTranscript_RedactsOnWrite(t *testing.T) {
	dir := t.TempDir()
	tr, err := New(dir, "redacted")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const secret = "sk-abcdefghijklmnopqrstuvwxyz1234"
	payload := "GET /v1/foo\nAuthorization: Bearer " + secret + "\nstatus=200\n"
	if _, err := tr.Write([]byte(payload)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "redacted.log"))
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	contents := string(data)
	if strings.Contains(contents, secret) {
		t.Fatalf("transcript leaked secret token; contents=%q", contents)
	}
	if !strings.Contains(contents, "[REDACTED]") {
		t.Fatalf("transcript missing [REDACTED] marker; contents=%q", contents)
	}
	// Non-secret context lines MUST survive so the artefact is still
	// useful for diagnostics.
	if !strings.Contains(contents, "GET /v1/foo") {
		t.Fatalf("transcript dropped non-secret line; contents=%q", contents)
	}
	if !strings.Contains(contents, "status=200") {
		t.Fatalf("transcript dropped non-secret line; contents=%q", contents)
	}
}

// TestTranscript_HandlesPartialLine verifies that Write may be called
// with an incomplete trailing line (no '\n') and that the line still
// reaches the file once Close flushes the buffer.
func TestTranscript_HandlesPartialLine(t *testing.T) {
	dir := t.TempDir()
	tr, err := New(dir, "partial")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := tr.Write([]byte("first part ")); err != nil {
		t.Fatalf("Write 1: %v", err)
	}
	if _, err := tr.Write([]byte("second part\n")); err != nil {
		t.Fatalf("Write 2: %v", err)
	}
	if _, err := tr.Write([]byte("trailing without newline")); err != nil {
		t.Fatalf("Write 3: %v", err)
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "partial.log"))
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	contents := string(data)
	if !strings.Contains(contents, "first part second part") {
		t.Fatalf("transcript: expected joined line, got %q", contents)
	}
	if !strings.Contains(contents, "trailing without newline") {
		t.Fatalf("transcript: missing trailing partial line, got %q", contents)
	}
}
