package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadYAML_BasicConfig(t *testing.T) {
	yaml := `
version: 1
agent:
  model: openai/gpt-4o
  system_prompt: You are a database assistant.
  max_tokens: 8192
`
	path := writeTempYAML(t, yaml)
	cfg, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Agent.Model != "openai/gpt-4o" {
		t.Errorf("expected 'openai/gpt-4o', got %q", cfg.Agent.Model)
	}
	if cfg.Agent.SystemPrompt != "You are a database assistant." {
		t.Errorf("unexpected system_prompt: %q", cfg.Agent.SystemPrompt)
	}
	if cfg.Agent.MaxTokens != 8192 {
		t.Errorf("expected 8192, got %d", cfg.Agent.MaxTokens)
	}
}

func TestLoadYAML_WithFallback(t *testing.T) {
	yaml := `
version: 1
agent:
  model: openai/gpt-4o
  fallback_models:
    - anthropic/claude-3-haiku
    - ollama/qwen3
`
	path := writeTempYAML(t, yaml)
	cfg, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Agent.FallbackModels) != 2 {
		t.Fatalf("expected 2 fallbacks, got %d", len(cfg.Agent.FallbackModels))
	}
	if cfg.Agent.FallbackModels[0] != "anthropic/claude-3-haiku" {
		t.Errorf("unexpected fallback[0]: %q", cfg.Agent.FallbackModels[0])
	}
}

func TestLoadYAML_WithTools(t *testing.T) {
	yaml := `
version: 1
agent:
  model: openai/gpt-4o
tools:
  - name: sql_query
    enabled: true
  - name: exec
    enabled: false
`
	path := writeTempYAML(t, yaml)
	cfg, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(cfg.Tools))
	}
	if cfg.Tools[0].Name != "sql_query" || !cfg.Tools[0].Enabled {
		t.Error("tool[0] should be sql_query enabled")
	}
	if cfg.Tools[1].Name != "exec" || cfg.Tools[1].Enabled {
		t.Error("tool[1] should be exec disabled")
	}
}

func TestLoadYAML_WithDatabase(t *testing.T) {
	yaml := `
version: 1
agent:
  model: openai/gpt-4o
database:
  path: ./myapp.db
`
	path := writeTempYAML(t, yaml)
	cfg, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Database == nil {
		t.Fatal("expected database config")
	}
	if cfg.Database.Path != "./myapp.db" {
		t.Errorf("expected './myapp.db', got %q", cfg.Database.Path)
	}
}

func TestLoadYAML_InvalidYAML(t *testing.T) {
	yaml := `
version: 1
agent:
  model: [invalid
`
	path := writeTempYAML(t, yaml)
	_, err := LoadYAML(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadYAML_FileNotFound(t *testing.T) {
	_, err := LoadYAML("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadYAML_EmptyConfig(t *testing.T) {
	yaml := `version: 1`
	path := writeTempYAML(t, yaml)
	cfg, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Agent.Model != "" {
		t.Errorf("expected empty model, got %q", cfg.Agent.Model)
	}
}

func TestToCoreConfig(t *testing.T) {
	yaml := `
version: 1
agent:
  model: openai/gpt-4o
  fallback_models:
    - anthropic/claude-3-haiku
  system_prompt: You are helpful.
  max_tokens: 4096
`
	path := writeTempYAML(t, yaml)
	cfg, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	coreCfg := cfg.ToCoreConfig()
	if coreCfg.Agents.Defaults.ModelName != "openai/gpt-4o" {
		t.Errorf("expected model 'openai/gpt-4o', got %q", coreCfg.Agents.Defaults.ModelName)
	}
	if coreCfg.Agents.Defaults.SystemPrompt != "You are helpful." {
		t.Errorf("unexpected system prompt: %q", coreCfg.Agents.Defaults.SystemPrompt)
	}
	if coreCfg.Agents.Defaults.MaxTokens != 4096 {
		t.Errorf("expected 4096, got %d", coreCfg.Agents.Defaults.MaxTokens)
	}
	if len(coreCfg.Agents.Defaults.FallbackModels) != 1 {
		t.Errorf("expected 1 fallback, got %d", len(coreCfg.Agents.Defaults.FallbackModels))
	}
}

func TestToCoreConfig_Defaults(t *testing.T) {
	yaml := `
version: 1
agent:
  model: openai/gpt-4o
`
	path := writeTempYAML(t, yaml)
	cfg, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	coreCfg := cfg.ToCoreConfig()
	if coreCfg.Agents.Defaults.MaxTokens != 8192 {
		t.Errorf("expected default max_tokens 8192, got %d", coreCfg.Agents.Defaults.MaxTokens)
	}
}

func TestDetectYAML(t *testing.T) {
	tests := []struct {
		path   string
		expect bool
	}{
		{"agent.yaml", true},
		{"agent.yml", true},
		{"config.json", false},
		{"config.yaml.bak", false},
		{"/path/to/agent.yaml", true},
	}
	for _, tt := range tests {
		if got := IsYAMLConfig(tt.path); got != tt.expect {
			t.Errorf("IsYAMLConfig(%q) = %v, want %v", tt.path, got, tt.expect)
		}
	}
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp YAML: %v", err)
	}
	return path
}
