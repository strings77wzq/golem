package ollama

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/strings77wzq/golem/core/providers"
)

// Compile-time interface checks.
var _ providers.LLMProvider = (*Provider)(nil)
var _ providers.StreamingProvider = (*Provider)(nil)
var _ providers.HealthChecker = (*Provider)(nil)

func TestNew(t *testing.T) {
	p := New()
	if p.Name() != "ollama" {
		t.Errorf("expected name 'ollama', got %q", p.Name())
	}
	if p.base != defaultAPIBase {
		t.Errorf("expected default base %q, got %q", defaultAPIBase, p.base)
	}
}

func TestNewWithAPIBase(t *testing.T) {
	p := New(WithAPIBase("http://custom:11434"))
	if p.base != "http://custom:11434" {
		t.Errorf("expected base 'http://custom:11434', got %q", p.base)
	}
}

func TestHealthCheckUnreachable(t *testing.T) {
	p := New(WithAPIBase("http://localhost:19999"))
	status, err := p.HealthCheck(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %q", status.Status)
	}
	if status.Provider != "ollama" {
		t.Errorf("expected provider 'ollama', got %q", status.Provider)
	}
}

func TestHealthCheckServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	p := New(WithAPIBase(server.URL))
	status, err := p.HealthCheck(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %q", status.Status)
	}
}

func TestHealthCheckNoModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []interface{}{},
		})
	}))
	defer server.Close()

	p := New(WithAPIBase(server.URL))
	status, err := p.HealthCheck(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "degraded" {
		t.Errorf("expected degraded, got %q", status.Status)
	}
}

func TestHealthCheckHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]string{
				{"name": "qwen3:0.5b"},
			},
		})
	}))
	defer server.Close()

	p := New(WithAPIBase(server.URL))
	status, err := p.HealthCheck(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "healthy" {
		t.Errorf("expected healthy, got %q", status.Status)
	}
}

func TestListModelsUnreachable(t *testing.T) {
	p := New(WithAPIBase("http://localhost:19999"))
	models, err := p.ListModels(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(models) != 0 {
		t.Errorf("expected empty models, got %v", models)
	}
}

func TestListModelsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]string{
				{"name": "qwen3:0.5b"},
				{"name": "llama3:8b"},
			},
		})
	}))
	defer server.Close()

	p := New(WithAPIBase(server.URL))
	models, err := p.ListModels(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}
	if models[0] != "qwen3:0.5b" {
		t.Errorf("expected 'qwen3:0.5b', got %q", models[0])
	}
}
