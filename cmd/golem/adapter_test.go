package main

import (
	"context"
	"testing"

	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/foundation/logger"
	"github.com/strings77wzq/golem/internal/wiring"
)

func TestLoadSkillsDefault(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	registry := wiring.LoadSkills(log, "", "")
	if registry.Count() == 0 {
		t.Error("expected built-in skills to be registered")
	}
}

func TestLoadSkillsFilter(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	registry := wiring.LoadSkills(log, "", "summarize")
	if registry.Count() != 1 {
		t.Errorf("expected 1 filtered skill, got %d", registry.Count())
	}
}

func TestBuildSystemPromptEmpty(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	registry := wiring.LoadSkills(log, "", "")
	prompt := wiring.BuildSystemPrompt("base prompt", registry)
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestBuildSystemPromptWithSkills(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	registry := wiring.LoadSkills(log, "", "summarize")
	prompt := wiring.BuildSystemPrompt("base prompt", registry)
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestLoadMCPToolsEmpty(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	registry := tools.NewRegistry()
	err := loadMCPTools(context.Background(), "", registry, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Count() != 0 {
		t.Errorf("expected 0 tools, got %d", registry.Count())
	}
}

func TestLoadMCPToolsInvalidJSON(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	registry := tools.NewRegistry()
	err := loadMCPTools(context.Background(), "not-json", registry, log)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadRAGToolsEmpty(t *testing.T) {
	cfg := newTestConfig()
	registry := tools.NewRegistry()
	err := loadRAGTools(context.Background(), cfg, "", registry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Count() != 0 {
		t.Errorf("expected 0 tools, got %d", registry.Count())
	}
}

func TestLoadMemoryToolsEmpty(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	registry := tools.NewRegistry()
	err := loadMemoryTools(context.Background(), "", registry, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Count() != 0 {
		t.Errorf("expected 0 tools, got %d", registry.Count())
	}
}

func TestParseMCPConfig(t *testing.T) {
	cfg, err := ParseMCPConfig(`{"servers":[{"name":"test","command":"echo"}]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(cfg.Servers))
	}
}

func TestParseRagConfig(t *testing.T) {
	cfg, err := ParseRagConfig(`{"api_key":"test","chunk_size":500}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "test" {
		t.Errorf("expected APIKey 'test', got %q", cfg.APIKey)
	}
}

func TestParseMemoryConfig(t *testing.T) {
	cfg, err := ParseMemoryConfig(`{"path":"/tmp/memory.json"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = cfg
}

func TestNewTestConfig(t *testing.T) {
	cfg := newTestConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Agents.Defaults.ModelName != "mock" {
		t.Errorf("expected model 'mock', got %q", cfg.Agents.Defaults.ModelName)
	}
}
