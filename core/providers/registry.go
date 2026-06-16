package providers

import (
	"fmt"
	"sync"
)

// ProviderConfig holds configuration for creating a provider.
type ProviderConfig struct {
	Vendor  string
	APIKey  string
	APIBase string
}

// ProviderConstructor is a function that creates a provider from config.
type ProviderConstructor func(cfg ProviderConfig) LLMProvider

// Registry manages provider constructors by vendor name.
type Registry struct {
	mu           sync.RWMutex
	constructors map[string]ProviderConstructor
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		constructors: make(map[string]ProviderConstructor),
	}
}

// Register adds a provider constructor for a vendor name.
func (r *Registry) Register(vendor string, fn ProviderConstructor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.constructors[vendor] = fn
}

// Create creates a provider for the given config.
// Returns error if no constructor registered for that vendor.
func (r *Registry) Create(cfg ProviderConfig) (LLMProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, ok := r.constructors[cfg.Vendor]
	if !ok {
		return nil, fmt.Errorf("no provider registered for vendor: %s", cfg.Vendor)
	}
	return fn(cfg), nil
}

// HasVendor checks if a vendor is registered.
func (r *Registry) HasVendor(vendor string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.constructors[vendor]
	return ok
}

// Vendors returns all registered vendor names.
func (r *Registry) Vendors() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	vendors := make([]string, 0, len(r.constructors))
	for v := range r.constructors {
		vendors = append(vendors, v)
	}
	return vendors
}

// GlobalRegistry is the default provider registry.
var GlobalRegistry = NewRegistry()
