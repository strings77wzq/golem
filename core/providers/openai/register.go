package openai

import (
	"github.com/strings77wzq/golem/core/providers"
)

func init() {
	providers.GlobalRegistry.Register("openai", func(cfg providers.ProviderConfig) providers.LLMProvider {
		var opts []Option
		if cfg.APIBase != "" {
			opts = append(opts, WithAPIBase(cfg.APIBase))
		}
		return New(cfg.APIKey, opts...)
	})

	// OpenAI-compatible vendors use the same adapter with different base URLs
	providers.GlobalRegistry.Register("deepseek", func(cfg providers.ProviderConfig) providers.LLMProvider {
		base := cfg.APIBase
		if base == "" {
			base = "https://api.deepseek.com"
		}
		return New(cfg.APIKey, WithAPIBase(base))
	})

	providers.GlobalRegistry.Register("moonshot", func(cfg providers.ProviderConfig) providers.LLMProvider {
		base := cfg.APIBase
		if base == "" {
			base = "https://api.moonshot.cn"
		}
		return New(cfg.APIKey, WithAPIBase(base))
	})

	providers.GlobalRegistry.Register("zhipu", func(cfg providers.ProviderConfig) providers.LLMProvider {
		base := cfg.APIBase
		if base == "" {
			base = "https://open.bigmodel.cn/api/paas"
		}
		return New(cfg.APIKey, WithAPIBase(base))
	})

	providers.GlobalRegistry.Register("minimax", func(cfg providers.ProviderConfig) providers.LLMProvider {
		base := cfg.APIBase
		if base == "" {
			base = "https://api.minimax.chat"
		}
		return New(cfg.APIKey, WithAPIBase(base))
	})

	providers.GlobalRegistry.Register("dashscope", func(cfg providers.ProviderConfig) providers.LLMProvider {
		base := cfg.APIBase
		if base == "" {
			base = "https://dashscope.aliyuncs.com/compatible-mode"
		}
		return New(cfg.APIKey, WithAPIBase(base))
	})

	providers.GlobalRegistry.Register("mimo", func(cfg providers.ProviderConfig) providers.LLMProvider {
		base := cfg.APIBase
		if base == "" {
			base = "https://api.xiaomimimo.com/v1"
		}
		return New(cfg.APIKey, WithAPIBase(base))
	})
}
