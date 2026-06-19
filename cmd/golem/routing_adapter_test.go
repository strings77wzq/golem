package main

import (
	"testing"

	"github.com/strings77wzq/golem/core/providers"
)

func TestParseRoutingConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		routes  int
	}{
		{
			name:    "valid single route",
			input:   `{"routes":{"gpt-4o":["openai/gpt-4o","anthropic/claude-sonnet-4-20250514"]}}`,
			wantErr: false,
			routes:  1,
		},
		{
			name:    "valid multiple routes",
			input:   `{"routes":{"gpt-4o":["openai/gpt-4o"],"deepseek":["deepseek/deepseek-chat","openai/gpt-4o-mini"]}}`,
			wantErr: false,
			routes:  2,
		},
		{
			name:    "empty routes",
			input:   `{"routes":{}}`,
			wantErr: false,
			routes:  0,
		},
		{
			name:    "invalid JSON",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseRoutingConfig(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRoutingConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(cfg.Routes) != tt.routes {
				t.Errorf("ParseRoutingConfig() routes = %d, want %d", len(cfg.Routes), tt.routes)
			}
		})
	}
}

func TestLoadRoutingFactory(t *testing.T) {
	factory := providers.NewFactory()
	_, err := LoadRoutingFactory(`{"routes":{"gpt-4o":["openai/gpt-4o"]}}`, factory)
	if err != nil {
		t.Fatalf("LoadRoutingFactory() unexpected error: %v", err)
	}
}

func TestLoadRoutingFactoryEmpty(t *testing.T) {
	factory := providers.NewFactory()
	router, err := LoadRoutingFactory("", factory)
	if err != nil {
		t.Fatalf("LoadRoutingFactory() unexpected error: %v", err)
	}
	if router != nil {
		t.Error("LoadRoutingFactory() should return nil router for empty config")
	}
}

func TestLoadRoutingFactoryInvalid(t *testing.T) {
	factory := providers.NewFactory()
	_, err := LoadRoutingFactory("not json", factory)
	if err == nil {
		t.Error("LoadRoutingFactory() expected error for invalid JSON")
	}
}
