package wiring

import (
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/providers/anthropic"
	"github.com/strings77wzq/golem/core/providers/ollama"
	"github.com/strings77wzq/golem/core/providers/openai"
)

// RegisterProviders creates a provider factory from config ModelList.
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

		var provider providers.LLMProvider

		switch vendor {
		case "openai":
			var opts []openai.Option
			if entry.APIBase != "" {
				opts = append(opts, openai.WithAPIBase(entry.APIBase))
			}
			provider = openai.New(entry.APIKey, opts...)
		case "anthropic":
			var opts []anthropic.Option
			if entry.APIBase != "" {
				opts = append(opts, anthropic.WithAPIBase(entry.APIBase))
			}
			provider = anthropic.New(entry.APIKey, opts...)
		case "deepseek":
			base := entry.APIBase
			if base == "" {
				base = "https://api.deepseek.com"
			}
			provider = openai.New(entry.APIKey, openai.WithAPIBase(base))
		case "moonshot":
			base := entry.APIBase
			if base == "" {
				base = "https://api.moonshot.cn"
			}
			provider = openai.New(entry.APIKey, openai.WithAPIBase(base))
		case "zhipu":
			base := entry.APIBase
			if base == "" {
				base = "https://open.bigmodel.cn/api/paas"
			}
			provider = openai.New(entry.APIKey, openai.WithAPIBase(base))
		case "minimax":
			base := entry.APIBase
			if base == "" {
				base = "https://api.minimax.chat"
			}
			provider = openai.New(entry.APIKey, openai.WithAPIBase(base))
		case "dashscope":
			base := entry.APIBase
			if base == "" {
				base = "https://dashscope.aliyuncs.com/compatible-mode"
			}
			provider = openai.New(entry.APIKey, openai.WithAPIBase(base))
		case "ollama":
			var opts []ollama.Option
			if entry.APIBase != "" {
				opts = append(opts, ollama.WithAPIBase(entry.APIBase))
			}
			provider = ollama.New(opts...)
		case "mimo":
			base := entry.APIBase
			if base == "" {
				base = "https://api.xiaomimimo.com/v1"
			}
			provider = openai.New(entry.APIKey, openai.WithAPIBase(base))
		case "mock":
			provider = providers.NewMockProvider("mock")
		}

		// Wrap with retry logic for resilience
		if provider != nil && vendor != "mock" {
			provider = providers.NewRetryProvider(provider, providers.RetryConfig{})
		}

		factory.Register(vendor, provider)
	}

	if !registered["mock"] {
		factory.Register("mock", providers.NewMockProvider("mock"))
	}

	return factory
}
