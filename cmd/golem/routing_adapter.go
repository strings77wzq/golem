package main

import (
	"encoding/json"
	"fmt"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/feature/routing"
)

// RoutingConfig holds the JSON configuration for the --routing flag.
type RoutingConfig struct {
	Routes map[string][]string `json:"routes"`
}

// ParseRoutingConfig parses the --routing flag value into a RoutingConfig.
func ParseRoutingConfig(flagValue string) (*RoutingConfig, error) {
	if flagValue == "" {
		return nil, fmt.Errorf("empty routing config")
	}
	var cfg RoutingConfig
	if err := json.Unmarshal([]byte(flagValue), &cfg); err != nil {
		return nil, fmt.Errorf("parsing routing config: %w", err)
	}
	if cfg.Routes == nil {
		cfg.Routes = make(map[string][]string)
	}
	return &cfg, nil
}

// LoadRoutingFactory parses the routing config and creates a Router wrapping the factory.
// Returns (nil, nil) if flagValue is empty (no routing needed).
func LoadRoutingFactory(flagValue string, factory *providers.Factory) (*routing.Router, error) {
	if flagValue == "" {
		return nil, nil
	}
	cfg, err := ParseRoutingConfig(flagValue)
	if err != nil {
		return nil, err
	}
	router := routing.NewRouter(factory)
	for model, chain := range cfg.Routes {
		router.AddRoute(model, chain...)
	}
	return router, nil
}
