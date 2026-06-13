// Package main — core wiring functions extracted from main.go.
// These build the registries, factories, and providers that the agent needs.
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/providers/anthropic"
	"github.com/strings77wzq/golem/core/providers/ollama"
	"github.com/strings77wzq/golem/core/providers/openai"
	"github.com/strings77wzq/golem/core/tools"
	toolexec "github.com/strings77wzq/golem/core/tools/exec"
	"github.com/strings77wzq/golem/core/tools/fileops"
	"github.com/strings77wzq/golem/core/tools/websearch"
	"github.com/strings77wzq/golem/feature/mcp"
	"github.com/strings77wzq/golem/feature/skills"
	"github.com/strings77wzq/golem/feature/skills/builtins"
	"github.com/strings77wzq/golem/foundation/logger"
)

// buildToolRegistry creates the default tool registry with built-in tools.
func buildToolRegistry(workspace string) *tools.Registry {
	registry := tools.NewRegistry()
	registry.Register(toolexec.New(workspace))
	registry.Register(fileops.NewFileReadTool(workspace))
	registry.Register(fileops.NewFileWriteTool(workspace))
	registry.Register(fileops.NewFileListTool(workspace))
	registry.Register(websearch.New())
	return registry
}

// registerProviders creates a Factory and registers providers from config ModelList.
func registerProviders(cfg *config.Config) *providers.Factory {
	factory := providers.NewFactory()

	registered := make(map[string]bool)

	for _, entry := range cfg.ModelList {
		vendor := entry.Vendor()
		if registered[vendor] {
			continue
		}
		registered[vendor] = true

		switch vendor {
		case "openai":
			var opts []openai.Option
			if entry.APIBase != "" {
				opts = append(opts, openai.WithAPIBase(entry.APIBase))
			}
			factory.Register(vendor, openai.New(entry.APIKey, opts...))
		case "anthropic":
			var opts []anthropic.Option
			if entry.APIBase != "" {
				opts = append(opts, anthropic.WithAPIBase(entry.APIBase))
			}
			factory.Register(vendor, anthropic.New(entry.APIKey, opts...))
		case "deepseek":
			base := entry.APIBase
			if base == "" {
				base = "https://api.deepseek.com"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "moonshot":
			base := entry.APIBase
			if base == "" {
				base = "https://api.moonshot.cn"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "zhipu":
			base := entry.APIBase
			if base == "" {
				base = "https://open.bigmodel.cn/api/paas"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "minimax":
			base := entry.APIBase
			if base == "" {
				base = "https://api.minimax.chat"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "dashscope":
			base := entry.APIBase
			if base == "" {
				base = "https://dashscope.aliyuncs.com/compatible-mode"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "ollama":
			var opts []ollama.Option
			if entry.APIBase != "" {
				opts = append(opts, ollama.WithAPIBase(entry.APIBase))
			}
			factory.Register(vendor, ollama.New(opts...))
		case "mimo":
			base := entry.APIBase
			if base == "" {
				base = "https://api.xiaomimimo.com/v1"
			}
			factory.Register(vendor, openai.New(entry.APIKey, openai.WithAPIBase(base)))
		case "mock":
			factory.Register(vendor, providers.NewMockProvider("mock"))
		}
	}

	if !registered["mock"] {
		factory.Register("mock", providers.NewMockProvider("mock"))
	}

	return factory
}



// loadSkills creates a skill registry, registers builtins, and optionally
// loads from a directory and filters by name.
func loadSkills(log logger.Logger, skillsDir, skillsFilter string) *skills.Registry {
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
				registry.Register(s)
			}
			log.Info("filtered skills", "count", registry.Count(), "names", skillsFilter)
		}
	}

	return registry
}

// buildSystemPrompt injects skill prompts into the base system prompt.
func buildSystemPrompt(basePrompt string, registry *skills.Registry) string {
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



// loadMCPTools loads MCP tools into the registry from a flag value.
func loadMCPTools(ctx context.Context, flagValue string, registry *tools.Registry, log logger.Logger) error {
	if flagValue == "" {
		return nil
	}
	mcpCfg, err := ParseMCPConfig(flagValue)
	if err != nil {
		return fmt.Errorf("parsing MCP config: %w", err)
	}
	mcpManager, err := LoadMCPTools(ctx, mcpCfg)
	if err != nil {
		return fmt.Errorf("loading MCP tools: %w", err)
	}
	mcpProxies, err := MCPToolsToRegistry(mcpManager)
	if err != nil {
		return fmt.Errorf("converting MCP tools: %w", err)
	}
	for _, proxy := range mcpProxies {
		registry.Register(proxy)
	}
	log.Info("loaded MCP tools", "count", len(mcpProxies))
	return nil
}

// loadRAGTools loads RAG tools into the registry from a flag value.
func loadRAGTools(ctx context.Context, cfg *config.Config, flagValue string, registry *tools.Registry) error {
	if flagValue == "" {
		return nil
	}
	ragCfg, err := ParseRagConfig(flagValue)
	if err != nil {
		return fmt.Errorf("parsing RAG config: %w", err)
	}
	if ragCfg.APIKey == "" {
		if defaultModel, _ := cfg.FindModel(cfg.Agents.Defaults.ModelName); defaultModel != nil {
			ragCfg.APIKey = defaultModel.APIKey
		}
	}
	ragRegistry, err := LoadRAGTools(ctx, ragCfg)
	if err != nil {
		return fmt.Errorf("loading RAG tools: %w", err)
	}
	for _, t := range ragRegistry.ListTools() {
		registry.Register(t)
	}
	return nil
}

// loadMemoryTools loads memory tools into the registry from a flag value.
func loadMemoryTools(ctx context.Context, flagValue string, registry *tools.Registry, log logger.Logger) error {
	if flagValue == "" {
		return nil
	}
	memCfg, err := ParseMemoryConfig(flagValue)
	if err != nil {
		return fmt.Errorf("parsing memory config: %w", err)
	}
	memRegistry, _, err := LoadMemoryTools(ctx, memCfg)
	if err != nil {
		return fmt.Errorf("loading memory tools: %w", err)
	}
	for _, t := range memRegistry.ListTools() {
		registry.Register(t)
	}
	log.Info("loaded memory tools")
	return nil
}

// MCPManager is an alias for convenience.
type MCPManager = mcp.Manager
