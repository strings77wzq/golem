package wiring

import (
	"testing"

	"github.com/strings77wzq/golem/foundation/logger"
)

func TestLoadSkills(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	registry := LoadSkills(log, "", "")
	if registry.Count() == 0 {
		t.Error("expected built-in skills to be registered")
	}
}

func TestLoadSkillsFilter(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	registry := LoadSkills(log, "", "summarize")
	if registry.Count() != 1 {
		t.Errorf("expected 1 filtered skill, got %d", registry.Count())
	}
}

func TestBuildSystemPromptEmpty(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	registry := LoadSkills(log, "", "")
	prompt := BuildSystemPrompt("base prompt", registry)
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestBuildSystemPromptWithSkills(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	registry := LoadSkills(log, "", "summarize")
	prompt := BuildSystemPrompt("base prompt", registry)
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
}
