// Package helpers provides black-box-test utilities for the Golem E2E
// suite. It MUST NOT import any package whose path begins with
// "github.com/strings77wzq/golem/" outside this module — see the
// architectural rule documented in tests/e2e/doc.go.
package helpers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// Sentinel errors returned by Detect. Callers MUST use errors.Is for
// comparison so future wrapping does not break consumers.
var (
	// ErrOllamaUnavailable indicates the Ollama HTTP endpoint could not
	// be reached, returned a non-2xx status, or produced an unparseable
	// response. The E2E suite treats this as a clean skip.
	ErrOllamaUnavailable = errors.New("ollama unavailable")

	// ErrModelNotPulled indicates Ollama is reachable but the required
	// model is not present in /api/tags. The E2E suite treats this as
	// a clean skip with a "run `ollama pull <model>`" hint.
	ErrModelNotPulled = errors.New("ollama model not pulled")
)

// detectTimeout bounds the /api/tags probe. Kept short so a missing or
// hung Ollama daemon does not stall the whole test run.
const detectTimeout = 3 * time.Second

// Detect probes the Ollama HTTP API at baseURL (e.g. "http://127.0.0.1:11434")
// and returns:
//
//   - nil                       — Ollama is reachable and `model` is pulled.
//   - ErrOllamaUnavailable      — connection error or non-2xx status.
//   - ErrModelNotPulled         — reachable but `model` is absent from /api/tags.
//
// It deliberately wraps the underlying error so operators can see why
// Ollama was unavailable; tests should compare with errors.Is.
func Detect(baseURL, model string) error {
	ctx, cancel := context.WithTimeout(context.Background(), detectTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("%w: build request: %v", ErrOllamaUnavailable, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOllamaUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: GET /api/tags returned %s", ErrOllamaUnavailable, resp.Status)
	}

	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("%w: decode /api/tags: %v", ErrOllamaUnavailable, err)
	}

	for _, m := range payload.Models {
		if m.Name == model {
			return nil
		}
	}
	return fmt.Errorf("%w: %q not in /api/tags", ErrModelNotPulled, model)
}

// SkipIfUnavailable calls Detect and, on any error, calls t.Skipf with a
// documented, actionable message. It is the entry point every behavioural
// test uses so the suite remains green for contributors without a local
// Ollama install.
func SkipIfUnavailable(t *testing.T, baseURL, model string) {
	t.Helper()
	switch err := Detect(baseURL, model); {
	case err == nil:
		return
	case errors.Is(err, ErrModelNotPulled):
		t.Skipf("skipping E2E: model %q not pulled — run `ollama pull %s` (detail: %v)", model, model, err)
	case errors.Is(err, ErrOllamaUnavailable):
		t.Skipf("skipping E2E: Ollama unavailable at %s — start it with `ollama serve` (detail: %v)", baseURL, err)
	default:
		t.Skipf("skipping E2E: unexpected Detect error: %v", err)
	}
}
