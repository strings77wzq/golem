package main

import (
	"testing"
	"time"

	"github.com/strings77wzq/golem/foundation/logger"
)

func TestParseHealthConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		enabled  bool
		interval time.Duration
	}{
		{
			name:     "boolean true",
			input:    "true",
			enabled:  true,
			interval: 5 * time.Minute,
		},
		{
			name:     "boolean false",
			input:    "false",
			enabled:  false,
			interval: 0,
		},
		{
			name:     "empty string",
			input:    "",
			enabled:  false,
			interval: 0,
		},
		{
			name:     "json with interval",
			input:    `{"interval":"60s"}`,
			enabled:  true,
			interval: 60 * time.Second,
		},
		{
			name:     "json default interval",
			input:    `{}`,
			enabled:  true,
			interval: 5 * time.Minute,
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseHealthConfig(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHealthConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if cfg.Enabled != tt.enabled {
					t.Errorf("ParseHealthConfig() enabled = %v, want %v", cfg.Enabled, tt.enabled)
				}
				if cfg.Enabled && cfg.Interval != tt.interval {
					t.Errorf("ParseHealthConfig() interval = %v, want %v", cfg.Interval, tt.interval)
				}
			}
		})
	}
}

func TestLoadHealthManager(t *testing.T) {
	log := logger.NopLogger()
	mgr, err := LoadHealthManager(`{"interval":"30s"}`, log)
	if err != nil {
		t.Fatalf("LoadHealthManager() unexpected error: %v", err)
	}
	if mgr == nil {
		t.Fatal("LoadHealthManager() returned nil manager")
	}
}

func TestLoadHealthManagerDisabled(t *testing.T) {
	log := logger.NopLogger()
	mgr, err := LoadHealthManager("false", log)
	if err != nil {
		t.Fatalf("LoadHealthManager() unexpected error: %v", err)
	}
	if mgr != nil {
		t.Error("LoadHealthManager() should return nil for disabled config")
	}
}

func TestLoadHealthManagerEmpty(t *testing.T) {
	log := logger.NopLogger()
	mgr, err := LoadHealthManager("", log)
	if err != nil {
		t.Fatalf("LoadHealthManager() unexpected error: %v", err)
	}
	if mgr != nil {
		t.Error("LoadHealthManager() should return nil for empty config")
	}
}
