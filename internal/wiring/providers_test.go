package wiring

import (
	"testing"

	"github.com/strings77wzq/golem/core/config"
)

func TestRegisterProviders(t *testing.T) {
	cfg := &config.Config{
		ModelList: []config.ModelEntry{
			{ModelName: "gpt4", Model: "openai/gpt-4o", APIKey: "sk-test"},
			{ModelName: "claude", Model: "anthropic/claude-sonnet-4", APIKey: "sk-ant-test"},
			{ModelName: "deepseek", Model: "deepseek/deepseek-chat", APIKey: "sk-ds-test"},
		},
	}

	factory := RegisterProviders(cfg)
	if factory == nil {
		t.Fatal("expected non-nil factory")
	}

	// Verify providers are registered
	for _, vendor := range []string{"openai", "anthropic", "deepseek"} {
		if _, err := factory.GetProvider(vendor); err != nil {
			t.Errorf("expected provider %q to be registered: %v", vendor, err)
		}
	}
}

func TestRegisterProvidersMockDefault(t *testing.T) {
	cfg := &config.Config{
		ModelList: []config.ModelEntry{},
	}

	factory := RegisterProviders(cfg)
	if factory == nil {
		t.Fatal("expected non-nil factory")
	}

	// Mock should always be registered
	if _, err := factory.GetProvider("mock"); err != nil {
		t.Errorf("expected mock provider to be registered: %v", err)
	}
}

func TestRegisterProvidersWrappedWithRetry(t *testing.T) {
	cfg := &config.Config{
		ModelList: []config.ModelEntry{
			{ModelName: "gpt4", Model: "openai/gpt-4o", APIKey: "sk-test"},
		},
	}

	factory := RegisterProviders(cfg)

	// The provider should be wrapped with retry logic
	// We can't directly check the type, but we can verify it works
	provider, err := factory.GetProvider("openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}

	// Verify the provider name is preserved
	if provider.Name() != "openai" {
		t.Errorf("expected name 'openai', got %q", provider.Name())
	}
}
