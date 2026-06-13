package ollama

import (
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
}
