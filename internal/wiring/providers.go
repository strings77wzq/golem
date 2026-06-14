package wiring

import (
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/providers/anthropic"
	"github.com/strings77wzq/golem/core/providers/ollama"
	"github.com/strings77wzq/golem/core/providers/openai"
)

// RegisterProviders creates a provider factory from config ModelList.
func RegisterProviders(cfg *config.Config) *providers.Factory {
	factory := providers.NewFactory()

	registered := make(map[string]bool)

	for _, entry := range cfg.ModelList {
		vendor := entry.Vendor()
		if registered[vendor] {
			continue
		}
		registered[vendor] = true

		switch vendor {
		case "openai":
			var opts []openai.Option
			if entry.APIBase != "" {
				opts = append(opts, openai.WithAPIBase(entry.APIBase))
			}
			factory.Register(vendor, openai.New(entry.APIKey, opts...))
		case "anthropic":
			var opts []anthropic.Option
			if entry.APIBase != "" {
				opts = append(opts, anthropic.WithAPIBase(entry.APIBase))
			}
			factory.Register(vendor, anthropic.New(entry.APIKey, opts...))
		case "deepseek":
			base := entry.APIBase
			if base == "" {
				base = "https://api.deepseek.com"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "moonshot":
			base := entry.APIBase
			if base == "" {
				base = "https://api.moonshot.cn"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "zhipu":
			base := entry.APIBase
			if base == "" {
				base = "https://open.bigmodel.cn/api/paas"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "minimax":
			base := entry.APIBase
			if base == "" {
				base = "https://api.minimax.chat"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "dashscope":
			base := entry.APIBase
			if base == "" {
				base = "https://dashscope.aliyuncs.com/compatible-mode"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "ollama":
			var opts []ollama.Option
			if entry.APIBase != "" {
				opts = append(opts, ollama.WithAPIBase(entry.APIBase))
			}
			factory.Register(vendor, ollama.New(opts...))
		case "mimo":
			base := entry.APIBase
			if base == "" {
				base = "https://api.xiaomimimo.com/v1"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "mock":
			factory.Register(vendor, providers.NewMockProvider("mock"))
		}
	}

	if !registered["mock"] {
		factory.Register("mock", providers.NewMockProvider("mock"))
	}

	return factory
}
