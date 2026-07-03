// Package ollama implements the [providers.LLMProvider] and
// [providers.StreamingProvider] interfaces for Ollama's local API.
// Ollama exposes an OpenAI-compatible endpoint at http://localhost:11434/v1,
// so this adapter wraps the openai adapter with Ollama-specific defaults:
// no API key required, localhost base URL, and Ollama model naming.
package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/providers/openai"
	"github.com/strings77wzq/golem/core/tools"
)

const (
	defaultAPIBase = "http://localhost:11434"
)

// Provider wraps the OpenAI adapter with Ollama-specific defaults.
type Provider struct {
	inner *openai.Provider
	base  string
}

// Option configures the Ollama provider.
type Option func(*Provider)

// WithAPIBase sets a custom Ollama API base URL.
func WithAPIBase(base string) Option {
	return func(p *Provider) {
		p.base = base
	}
}

// New creates a new Ollama provider. No API key is required.
func New(opts ...Option) *Provider {
	p := &Provider{
		base: defaultAPIBase,
	}

	for _, opt := range opts {
		opt(p)
	}

	// Ollama doesn't require an API key; use a dummy value.
	p.inner = openai.New("ollama", openai.WithAPIBase(p.base+"/v1"))
	return p
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "ollama"
}

// Chat delegates to the OpenAI adapter.
func (p *Provider) Chat(
	ctx context.Context,
	messages []providers.Message,
	toolDefs []tools.ToolDefinition,
	model string,
	opts *providers.ChatOptions,
) (*providers.LLMResponse, error) {
	return p.inner.Chat(ctx, messages, toolDefs, model, opts)
}

// ChatStream delegates to the OpenAI adapter.
func (p *Provider) ChatStream(
	ctx context.Context,
	messages []providers.Message,
	toolDefs []tools.ToolDefinition,
	model string,
	opts *providers.ChatOptions,
	onToken func(token string),
) (*providers.LLMResponse, error) {
	return p.inner.ChatStream(ctx, messages, toolDefs, model, opts, onToken)
}

// HealthCheck queries Ollama's /api/tags endpoint to verify connectivity
// and list available models.
func (p *Provider) HealthCheck(ctx context.Context) (*providers.HealthStatus, error) {
	start := time.Now()

	url := p.base + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &providers.HealthStatus{
			Provider:  p.Name(),
			Status:    "unhealthy",
			Latency:   time.Since(start).Milliseconds(),
			Error:     fmt.Sprintf("failed to create request: %v", err),
			CheckedAt: time.Now().Unix(),
		}, nil
	}

	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &providers.HealthStatus{
			Provider:  p.Name(),
			Status:    "unhealthy",
			Latency:   latency,
			Error:     fmt.Sprintf("request failed: %v", err),
			CheckedAt: time.Now().Unix(),
		}, nil
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &providers.HealthStatus{
			Provider:  p.Name(),
			Status:    "unhealthy",
			Latency:   latency,
			Error:     fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(body)),
			CheckedAt: time.Now().Unix(),
		}, nil
	}

	// Parse response to count available models
	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err == nil && len(tagsResp.Models) == 0 {
		return &providers.HealthStatus{
			Provider:  p.Name(),
			Status:    "degraded",
			Latency:   latency,
			Error:     "no models installed",
			CheckedAt: time.Now().Unix(),
		}, nil
	}

	status := "healthy"
	if latency > 2000 {
		status = "degraded"
	}

	return &providers.HealthStatus{
		Provider:  p.Name(),
		Status:    status,
		Latency:   latency,
		CheckedAt: time.Now().Unix(),
	}, nil
}

// ListModels queries Ollama's /api/tags to return installed model names.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	url := p.base + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(tagsResp.Models))
	for _, m := range tagsResp.Models {
		names = append(names, m.Name)
	}
	return names, nil
}
