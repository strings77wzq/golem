package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/strings77wzq/golem/feature/health"
	"github.com/strings77wzq/golem/foundation/logger"
)

// HealthConfig holds the JSON configuration for the --health flag.
type HealthConfig struct {
	Enabled  bool          `json:"-"`
	Interval time.Duration `json:"-"`
}

// ParseHealthConfig parses the --health flag value.
// Accepts "true"/"false" (boolean shorthand) or JSON with "interval" field.
func ParseHealthConfig(flagValue string) (*HealthConfig, error) {
	if flagValue == "" {
		return &HealthConfig{Enabled: false}, nil
	}

	// Boolean shorthand
	if flagValue == "true" {
		return &HealthConfig{Enabled: true, Interval: 5 * time.Minute}, nil
	}
	if flagValue == "false" {
		return &HealthConfig{Enabled: false}, nil
	}

	// JSON format
	var raw struct {
		Interval string `json:"interval"`
	}
	if err := json.Unmarshal([]byte(flagValue), &raw); err != nil {
		return nil, fmt.Errorf("parsing health config: %w", err)
	}

	interval := 5 * time.Minute
	if raw.Interval != "" {
		d, err := time.ParseDuration(raw.Interval)
		if err != nil {
			return nil, fmt.Errorf("parsing health interval: %w", err)
		}
		interval = d
	}

	return &HealthConfig{Enabled: true, Interval: interval}, nil
}

// LoadHealthManager parses the health config and creates a Manager.
// Returns (nil, nil) if disabled or empty.
func LoadHealthManager(flagValue string, log logger.Logger) (*health.Manager, error) {
	cfg, err := ParseHealthConfig(flagValue)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, nil
	}
	mgr := health.New(log, health.WithInterval(cfg.Interval))
	return mgr, nil
}
