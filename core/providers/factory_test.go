package providers

import (
	"context"
	"fmt"
	"testing"

	"github.com/strings77wzq/golem/core/tools"
)

func TestGetProviderForModelWithFallback_PrimarySucceeds(t *testing.T) {
	f := NewFactory()
	primary := NewMockProvider("openai")
	primary.AddResponse(&LLMResponse{Content: "hello"})
	f.Register("openai", primary)

	provider, modelName, usedModel, err := f.GetProviderForModelWithFallback(
		"openai/gpt-4o",
		[]string{"anthropic/claude-3-haiku"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != primary {
		t.Error("expected primary provider")
	}
	if modelName != "gpt-4o" {
		t.Errorf("expected 'gpt-4o', got %q", modelName)
	}
	if usedModel != "openai/gpt-4o" {
		t.Errorf("expected 'openai/gpt-4o', got %q", usedModel)
	}
}

func TestGetProviderForModelWithFallback_PrimaryFailsFallbackSucceeds(t *testing.T) {
	f := NewFactory()

	// Primary vendor not registered — should fail
	fallback := NewMockProvider("anthropic")
	fallback.AddResponse(&LLMResponse{Content: "fallback response"})
	f.Register("anthropic", fallback)

	provider, modelName, usedModel, err := f.GetProviderForModelWithFallback(
		"openai/gpt-4o",
		[]string{"anthropic/claude-3-haiku"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != fallback {
		t.Error("expected fallback provider")
	}
	if modelName != "claude-3-haiku" {
		t.Errorf("expected 'claude-3-haiku', got %q", modelName)
	}
	if usedModel != "anthropic/claude-3-haiku" {
		t.Errorf("expected 'anthropic/claude-3-haiku', got %q", usedModel)
	}
}

func TestGetProviderForModelWithFallback_AllFail(t *testing.T) {
	f := NewFactory()
	// No providers registered

	_, _, _, err := f.GetProviderForModelWithFallback(
		"openai/gpt-4o",
		[]string{"anthropic/claude-3-haiku"},
	)
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
}

func TestGetProviderForModelWithFallback_NoFallbacks(t *testing.T) {
	f := NewFactory()
	// No providers registered, no fallbacks

	_, _, _, err := f.GetProviderForModelWithFallback("openai/gpt-4o", nil)
	if err == nil {
		t.Fatal("expected error when primary fails and no fallbacks")
	}
}

func TestGetProviderForModelWithFallback_EmptyFallbacks(t *testing.T) {
	f := NewFactory()
	// No providers registered

	_, _, _, err := f.GetProviderForModelWithFallback("openai/gpt-4o", []string{})
	if err == nil {
		t.Fatal("expected error when primary fails and empty fallbacks")
	}
}

func TestGetProviderForModelWithFallback_MultipleFallbacks(t *testing.T) {
	f := NewFactory()

	// Second fallback succeeds
	fallback2 := NewMockProvider("ollama")
	fallback2.AddResponse(&LLMResponse{Content: "local response"})
	f.Register("ollama", fallback2)

	provider, modelName, usedModel, err := f.GetProviderForModelWithFallback(
		"openai/gpt-4o",
		[]string{"anthropic/claude-3-haiku", "ollama/qwen3"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != fallback2 {
		t.Error("expected second fallback provider")
	}
	if modelName != "qwen3" {
		t.Errorf("expected 'qwen3', got %q", modelName)
	}
	if usedModel != "ollama/qwen3" {
		t.Errorf("expected 'ollama/qwen3', got %q", usedModel)
	}
}

// errorProvider always returns errors
type errorProvider struct {
	name string
}

func (p *errorProvider) Name() string { return p.name }
func (p *errorProvider) Chat(_ context.Context, _ []Message, _ []tools.ToolDefinition, _ string, _ *ChatOptions) (*LLMResponse, error) {
	return nil, fmt.Errorf("%s: simulated failure", p.name)
}

func TestGetProviderForModelWithFallback_PrimaryProviderError(t *testing.T) {
	f := NewFactory()

	// Primary provider registered but will fail at Chat time
	// The fallback resolution only checks if vendor is registered, not if Chat succeeds
	// So this test verifies that the Factory resolves the correct provider
	// The actual retry-on-error logic is in the agent layer
	primary := &errorProvider{name: "openai"}
	f.Register("openai", primary)

	fallback := NewMockProvider("anthropic")
	fallback.AddResponse(&LLMResponse{Content: "fallback"})
	f.Register("anthropic", fallback)

	// Factory resolves primary successfully (vendor exists)
	provider, modelName, usedModel, err := f.GetProviderForModelWithFallback(
		"openai/gpt-4o",
		[]string{"anthropic/claude-3-haiku"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != primary {
		t.Error("expected primary provider (factory resolves by vendor, not by health)")
	}
	if modelName != "gpt-4o" {
		t.Errorf("expected 'gpt-4o', got %q", modelName)
	}
	if usedModel != "openai/gpt-4o" {
		t.Errorf("expected 'openai/gpt-4o', got %q", usedModel)
	}
}
