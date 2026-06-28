package main

import (
	"strings"
	"testing"

	"github.com/strings77wzq/golem/foundation/logger"
)

// TestLoadSkillsBehavior verifies LoadSkills behavior after moving from internal/wiring.
// This test ensures the function works correctly in its new location (cmd/golem).
func TestLoadSkillsBehavior(t *testing.T) {
	t.Run("loads_builtin_skills", func(t *testing.T) {
		log := logger.New(logger.DefaultOptions())
		registry := LoadSkills(log, "", "")

		if registry.Count() == 0 {
			t.Error("expected built-in skills to be registered, got 0")
		}

		// Verify specific built-in skills exist
		skills := registry.List()
		found := map[string]bool{}
		for _, s := range skills {
			found[s.Name] = true
		}

		// At least one built-in should exist
		if len(found) == 0 {
			t.Error("no built-in skills found in registry")
		}
	})

	t.Run("filters_by_name", func(t *testing.T) {
		log := logger.New(logger.DefaultOptions())

		// Get all skills first
		allRegistry := LoadSkills(log, "", "")
		allCount := allRegistry.Count()

		// Filter to a specific skill
		filteredRegistry := LoadSkills(log, "", "summarize")
		filteredCount := filteredRegistry.Count()

		if filteredCount >= allCount {
			t.Errorf("filter should reduce count: all=%d, filtered=%d", allCount, filteredCount)
		}

		if filteredCount != 1 {
			t.Errorf("expected 1 filtered skill, got %d", filteredCount)
		}

		// Verify the filtered skill is "summarize"
		skills := filteredRegistry.List()
		if len(skills) > 0 && skills[0].Name != "summarize" {
			t.Errorf("expected skill name 'summarize', got %q", skills[0].Name)
		}
	})

	t.Run("handles_multiple_filters", func(t *testing.T) {
		log := logger.New(logger.DefaultOptions())
		registry := LoadSkills(log, "", "summarize,analyze")

		// Should have at most 2 skills
		if registry.Count() > 2 {
			t.Errorf("expected at most 2 skills, got %d", registry.Count())
		}
	})

	t.Run("handles_empty_filter", func(t *testing.T) {
		log := logger.New(logger.DefaultOptions())
		registry := LoadSkills(log, "", "")

		if registry.Count() == 0 {
			t.Error("empty filter should load all built-in skills")
		}
	})

	t.Run("handles_nonexistent_filter", func(t *testing.T) {
		log := logger.New(logger.DefaultOptions())
		registry := LoadSkills(log, "", "nonexistent_skill_xyz")

		if registry.Count() != 0 {
			t.Errorf("nonexistent filter should result in 0 skills, got %d", registry.Count())
		}
	})
}

// TestBuildSystemPromptBehavior verifies BuildSystemPrompt behavior after moving from internal/wiring.
func TestBuildSystemPromptBehavior(t *testing.T) {
	t.Run("returns_base_prompt_when_empty", func(t *testing.T) {
		log := logger.New(logger.DefaultOptions())
		registry := LoadSkills(log, "", "nonexistent_xyz")
		basePrompt := "You are a helpful assistant."

		result := BuildSystemPrompt(basePrompt, registry)

		if result != basePrompt {
			t.Errorf("expected base prompt unchanged, got %q", result)
		}
	})

	t.Run("injects_skill_prompts", func(t *testing.T) {
		log := logger.New(logger.DefaultOptions())
		registry := LoadSkills(log, "", "")
		basePrompt := "You are a helpful assistant."

		result := BuildSystemPrompt(basePrompt, registry)

		// Should contain base prompt
		if !strings.Contains(result, basePrompt) {
			t.Error("result should contain base prompt")
		}

		// Should contain skill section header
		if !strings.Contains(result, "Available skills:") {
			t.Error("result should contain 'Available skills:' header")
		}

		// Should be longer than base prompt
		if len(result) <= len(basePrompt) {
			t.Error("result should be longer than base prompt when skills exist")
		}
	})

	t.Run("includes_skill_names_and_descriptions", func(t *testing.T) {
		log := logger.New(logger.DefaultOptions())
		registry := LoadSkills(log, "", "summarize")
		basePrompt := "Base"

		result := BuildSystemPrompt(basePrompt, registry)

		// Should contain skill name
		if !strings.Contains(result, "summarize") {
			t.Error("result should contain skill name 'summarize'")
		}

		// Should contain skill section markers
		if !strings.Contains(result, "## Skill:") {
			t.Error("result should contain '## Skill:' markers")
		}
	})

	t.Run("preserves_base_prompt_at_end", func(t *testing.T) {
		log := logger.New(logger.DefaultOptions())
		registry := LoadSkills(log, "", "")
		basePrompt := "You are a database assistant."

		result := BuildSystemPrompt(basePrompt, registry)

		// Base prompt should appear after skills section
		skillsIdx := strings.Index(result, "Available skills:")
		baseIdx := strings.Index(result, basePrompt)

		if skillsIdx == -1 {
			t.Fatal("skills section not found")
		}
		if baseIdx == -1 {
			t.Fatal("base prompt not found")
		}
		if baseIdx < skillsIdx {
			t.Error("base prompt should appear after skills section")
		}
	})
}

// TestLoadSkillsConcurrency verifies thread safety of LoadSkills.
func TestLoadSkillsConcurrency(t *testing.T) {
	log := logger.New(logger.DefaultOptions())
	const goroutines = 10

	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			registry := LoadSkills(log, "", "")
			if registry.Count() == 0 {
				t.Errorf("goroutine %d: expected skills", id)
			}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
}
