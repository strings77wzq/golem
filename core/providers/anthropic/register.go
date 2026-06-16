package anthropic

import (
	"github.com/strings77wzq/golem/core/providers"
)

func init() {
	providers.GlobalRegistry.Register("anthropic", func(cfg providers.ProviderConfig) providers.LLMProvider {
		var opts []Option
		if cfg.APIBase != "" {
			opts = append(opts, WithAPIBase(cfg.APIBase))
		}
		return New(cfg.APIKey, opts...)
	})
}
