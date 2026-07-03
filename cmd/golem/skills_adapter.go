package main

import (
	"fmt"
	"strings"

	"github.com/strings77wzq/golem/feature/skills"
	"github.com/strings77wzq/golem/feature/skills/builtins"
	"github.com/strings77wzq/golem/foundation/logger"
)

// LoadSkills creates a skill registry, registers builtins, and optionally
// loads from a directory and filters by name.
func LoadSkills(log logger.Logger, skillsDir, skillsFilter string) *skills.Registry {
	registry := skills.NewRegistry()
	builtins.RegisterAll(registry)

	if skillsDir != "" {
		loader := skills.NewLoader()
		loaded, loadErr := loader.LoadFromDirectory(skillsDir)
		if loadErr != nil {
			log.Warn("failed to load skills from directory", "dir", skillsDir, "err", loadErr)
		} else {
			for _, s := range loaded {
				if regErr := registry.Register(s); regErr != nil {
					log.Warn("failed to register skill", "name", s.Name, "err", regErr)
				} else {
					log.Info("loaded skill", "name", s.Name)
				}
			}
		}
	}

	if skillsFilter != "" {
		requested := make(map[string]bool)
		for _, name := range strings.Split(skillsFilter, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				requested[name] = true
			}
		}
		if len(requested) > 0 {
			filtered := registry.List()[:0]
			for _, s := range registry.List() {
				if requested[s.Name] {
					filtered = append(filtered, s)
				}
			}
			registry = skills.NewRegistry()
			for _, s := range filtered {
				if err := registry.Register(s); err != nil {
					log.Warn("failed to register skill", "name", s.Name, "error", err)
				}
			}
			log.Info("filtered skills", "count", registry.Count(), "names", skillsFilter)
		}
	}

	return registry
}

// BuildSystemPrompt injects skill prompts into the base system prompt.
func BuildSystemPrompt(basePrompt string, registry *skills.Registry) string {
	if registry.Count() == 0 {
		return basePrompt
	}
	var sb strings.Builder
	sb.WriteString("Available skills:\n\n")
	for _, s := range registry.List() {
		sb.WriteString(fmt.Sprintf("## Skill: %s\n%s\n\n", s.Name, s.Description))
		for _, p := range s.Prompts {
			sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", p.Name, p.Content))
		}
	}
	sb.WriteString("---\n\n")
	sb.WriteString(basePrompt)
	return sb.String()
}
