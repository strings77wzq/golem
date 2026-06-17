package helpers

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testModel = "qwen3:0.5b"

// TestDetectOllama_NotRunning verifies that when the Ollama HTTP endpoint is
// unreachable (server closed), Detect returns the sentinel ErrOllamaUnavailable.
func TestDetectOllama_NotRunning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // close immediately so subsequent requests fail at the transport layer.

	err := Detect(srv.URL, testModel)
	if err == nil {
		t.Fatalf("Detect: expected error against closed server, got nil")
	}
	if !errors.Is(err, ErrOllamaUnavailable) {
		t.Fatalf("Detect: expected ErrOllamaUnavailable, got %v", err)
	}
}

// TestDetectOllama_ModelMissing verifies that when Ollama is reachable but
// /api/tags does not contain the required model, Detect returns the
// sentinel ErrModelNotPulled (not ErrOllamaUnavailable).
func TestDetectOllama_ModelMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"models":[]}`)
	}))
	defer srv.Close()

	err := Detect(srv.URL, testModel)
	if err == nil {
		t.Fatalf("Detect: expected ErrModelNotPulled when models list is empty, got nil")
	}
	if !errors.Is(err, ErrModelNotPulled) {
		t.Fatalf("Detect: expected ErrModelNotPulled, got %v", err)
	}
	if errors.Is(err, ErrOllamaUnavailable) {
		t.Fatalf("Detect: must distinguish missing-model from unavailable; got both, err=%v", err)
	}
}

// TestDetectOllama_HappyPath verifies that when /api/tags includes the
// required model, Detect returns nil.
func TestDetectOllama_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// Realistic shape: Ollama returns more fields per model; only "name"
		// is required by Detect, so the extra fields exercise the
		// permissive JSON decoder.
		_, _ = io.WriteString(w, `{"models":[{"name":"`+testModel+`","size":1234},{"name":"other:latest"}]}`)
	}))
	defer srv.Close()

	if err := Detect(srv.URL, testModel); err != nil {
		t.Fatalf("Detect: expected nil with model present, got %v", err)
	}
}

// TestDetectOllama_Non200 verifies that a reachable endpoint returning a
// non-2xx status surfaces ErrOllamaUnavailable (not nil, not ModelNotPulled).
// This locks in the contract for the GREEN step in task 2.2.
func TestDetectOllama_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := Detect(srv.URL, testModel)
	if !errors.Is(err, ErrOllamaUnavailable) {
		t.Fatalf("Detect: expected ErrOllamaUnavailable on 500, got %v", err)
	}
}

// recordingT is a minimal sink for Skipf calls that records whether Skip
// fired and what message was used. It deliberately does not embed
// *testing.T because the public testing.T API surface is wide and
// intercepting Skipf is enough to verify SkipIfUnavailable's contract.
//
// We rely on the fact that SkipIfUnavailable only calls t.Helper() and
// t.Skipf on its argument. To avoid coupling to *testing.T we use a
// child test (t.Run) and observe its Skipped() flag.

// TestSkipIfUnavailable_Skips verifies SkipIfUnavailable skips the test
// (via t.Skipf) when Detect would return an error. We observe this by
// running the helper inside a sub-test and asserting the sub-test was
// marked Skipped.
func TestSkipIfUnavailable_Skips(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	skipped := t.Run("inner", func(inner *testing.T) {
		SkipIfUnavailable(inner, srv.URL, testModel)
		inner.Fatalf("SkipIfUnavailable: expected Skipf to short-circuit the test, but execution continued")
	})

	// t.Run returns true if the subtest did not fail. A skipped subtest
	// returns true. To distinguish "passed" from "skipped" we inspect the
	// post-run state: a passing subtest would have hit the Fatalf above
	// and t.Run would return false. So `skipped == true` here means the
	// sub-test did not fail, and the only way to do that given the
	// Fatalf is to have called t.Skipf first.
	if !skipped {
		t.Fatalf("SkipIfUnavailable: subtest unexpectedly failed; expected a skip path")
	}
}

// TestSkipIfUnavailable_NoSkipWhenHealthy verifies SkipIfUnavailable does
// NOT call Skip when Detect would succeed. We observe this by running a
// sub-test that does succeed and asserting it ran to completion.
func TestSkipIfUnavailable_NoSkipWhenHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"models":[{"name":"`+testModel+`"}]}`)
	}))
	defer srv.Close()

	ran := false
	ok := t.Run("inner", func(inner *testing.T) {
		SkipIfUnavailable(inner, srv.URL, testModel)
		ran = true
	})
	if !ok || !ran {
		t.Fatalf("SkipIfUnavailable: expected pass-through on healthy endpoint; ok=%v ran=%v", ok, ran)
	}
}
