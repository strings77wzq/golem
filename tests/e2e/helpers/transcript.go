package helpers

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Transcript captures stdout/stderr from an E2E binary run to a file
// under a transcript directory. Every line is piped through Redact
// before being written, so no Bearer token, sk- key, or other matched
// credential reaches the on-disk artefact uploaded by CI.
//
// Transcripts are append-only and must be Closed before the file is
// read by an assertion. The zero value is NOT usable — always
// construct via New.
//
// Concurrency: a Transcript is safe for concurrent calls to Write and
// Close. Calls after Close return an error rather than panicking; this
// keeps misbehaving subprocess pipes from crashing the test binary.
type Transcript struct {
	mu      sync.Mutex
	f       *os.File
	pending bytes.Buffer // accumulates partial lines across Write boundaries
	closed  bool
}

// ErrTranscriptClosed is returned by Write/Close on a Transcript that
// has already been closed. Callers can use errors.Is for comparison.
var ErrTranscriptClosed = errors.New("transcript already closed")

// New creates a transcript file at <dir>/<name>.log. The directory is
// created if it does not exist. The file is opened in truncate mode so
// reruns produce a clean artefact. A header line of the form
// "# <name> @ <RFC3339-timestamp>" is written before any caller content.
//
// `name` is used verbatim as the file stem; callers SHOULD pre-sanitise
// it for OS path safety. Forward slashes inside `name` are rejected to
// catch path-injection bugs early.
func New(dir, name string) (*Transcript, error) {
	if strings.ContainsAny(name, "/\\") {
		return nil, fmt.Errorf("transcript: name %q must not contain path separators", name)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("transcript: mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, name+".log")
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("transcript: create %s: %w", path, err)
	}
	header := fmt.Sprintf("# %s @ %s\n", name, time.Now().UTC().Format(time.RFC3339))
	if _, err := f.WriteString(header); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("transcript: write header: %w", err)
	}
	return &Transcript{f: f}, nil
}

// Write implements io.Writer. Bytes are accumulated in an internal
// buffer; whenever the buffer contains a '\n', complete lines are
// passed through Redact and flushed to the file. Partial lines remain
// buffered until the next Write that completes them, or until Close.
//
// Write returns the length of the input it accepted (always len(p) on
// success), per the io.Writer contract. The actual bytes-on-disk count
// will differ when Redact substitutes or drops a line.
func (t *Transcript) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return 0, ErrTranscriptClosed
	}

	t.pending.Write(p)
	for {
		buf := t.pending.Bytes()
		idx := bytes.IndexByte(buf, '\n')
		if idx < 0 {
			break
		}
		line := string(buf[:idx])
		t.pending.Next(idx + 1) // drop the line + the newline

		out := Redact(line)
		if out == "" {
			// Line was dropped by the conservative heuristic. Skip the
			// newline too so the on-disk artefact stays compact.
			continue
		}
		if _, err := t.f.WriteString(out + "\n"); err != nil {
			return 0, fmt.Errorf("transcript: write line: %w", err)
		}
	}
	return len(p), nil
}

// Close flushes any buffered partial line (passing it through Redact)
// and closes the underlying file. Subsequent Writes return
// ErrTranscriptClosed.
func (t *Transcript) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return ErrTranscriptClosed
	}
	t.closed = true

	if t.pending.Len() > 0 {
		out := Redact(t.pending.String())
		if out != "" {
			if _, err := t.f.WriteString(out); err != nil {
				_ = t.f.Close()
				return fmt.Errorf("transcript: flush partial: %w", err)
			}
		}
		t.pending.Reset()
	}
	return t.f.Close()
}

// Path returns the absolute path of the on-disk transcript file.
// Useful for assertions and for the CI artefact uploader.
func (t *Transcript) Path() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.f == nil {
		return ""
	}
	return t.f.Name()
}
