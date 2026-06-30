// Package routing provides a fallback-capable [Router] that wraps multiple
// [providers.LLMProvider] instances. If the primary provider returns an error,
// the router automatically retries with the next registered provider.
package routing

import (
	"context"
	"sync"
	"time"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/tools"
)

const defaultCooldown = 5 * time.Minute

// Router routes model requests to providers with fallback support.
type Router struct {
	mu      sync.RWMutex
	routes  map[string][]string
	factory *providers.Factory
	chains  map[string]*FallbackChain
}

// NewRouter creates a new router with the given provider factory.
func NewRouter(factory *providers.Factory) *Router {
	return &Router{
		routes:  make(map[string][]string),
		factory: factory,
		chains:  make(map[string]*FallbackChain),
	}
}

// AddRoute maps a model alias to one or more provider/model pairs (in fallback order).
func (r *Router) AddRoute(modelName string, providerModels ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[modelName] = providerModels
	r.chains[modelName] = NewFallbackChain(providerModels, defaultCooldown)
}

// Chat routes the request to the appropriate provider with cooldown-aware fallback.
func (r *Router) Chat(ctx context.Context, modelName string, messages []providers.Message, toolDefs []tools.ToolDefinition, opts *providers.ChatOptions) (*providers.LLMResponse, error) {
	r.mu.RLock()
	route, hasRoute := r.routes[modelName]
	chain := r.chains[modelName]
	r.mu.RUnlock()

	if hasRoute && chain != nil {
		var lastErr error
		// Try each provider in the route, respecting cooldowns
		for _, providerModel := range route {
			if chain.IsInCooldown(providerModel) {
				continue
			}
			provider, model, err := r.factory.GetProviderForModel(providerModel)
			if err != nil {
				lastErr = err
				continue
			}

			resp, err := provider.Chat(ctx, messages, toolDefs, model, opts)
			if err != nil {
				lastErr = err
				chain.MarkFailed(providerModel)
				if IsRetryable(err) {
					continue
				}
				return nil, err
			}
			chain.MarkSuccess(providerModel)
			return resp, nil
		}
		return nil, lastErr
	}

	// No route defined — fall back to direct factory lookup
	provider, model, err := r.factory.GetProviderForModel(modelName)
	if err != nil {
		return nil, err
	}

	return provider.Chat(ctx, messages, toolDefs, model, opts)
}
