package wiring

import (
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"

	// Import provider packages to trigger init() registration
	_ "github.com/strings77wzq/golem/core/providers/anthropic"
	_ "github.com/strings77wzq/golem/core/providers/ollama"
	_ "github.com/strings77wzq/golem/core/providers/openai"
)

// RegisterProviders creates a provider factory from config ModelList.
// Uses the provider registry for dynamic vendor support.
// All providers are wrapped with RetryProvider for exponential backoff retry.
func RegisterProviders(cfg *config.Config) *providers.Factory {
	factory := providers.NewFactory()

	registered := make(map[string]bool)

	for _, entry := range cfg.ModelList {
		vendor := entry.Vendor()
		if registered[vendor] {
			continue
		}
		registered[vendor] = true

		// Skip mock — register directly
		if vendor == "mock" {
			factory.Register(vendor, providers.NewMockProvider("mock"))
			continue
		}

		// Use registry to create provider
		provider, err := providers.GlobalRegistry.Create(providers.ProviderConfig{
			Vendor:  vendor,
			APIKey:  entry.APIKey,
			APIBase: entry.APIBase,
		})
		if err != nil {
			// Unknown vendor — skip
			continue
		}

		// Wrap with retry logic for resilience
		provider = providers.NewRetryProvider(provider, providers.RetryConfig{})
		factory.Register(vendor, provider)
	}

	if !registered["mock"] {
		factory.Register("mock", providers.NewMockProvider("mock"))
	}

	return factory
}
