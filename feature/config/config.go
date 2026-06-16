// Package config provides YAML-based declarative agent configuration.
// It allows defining agents, tools, databases, and MCP servers in a YAML file
// instead of using CLI flags.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	corecfg "github.com/strings77wzq/golem/core/config"
)

// AgentConfig represents a YAML agent definition.
type AgentConfig struct {
	Version    int            `yaml:"version"`
	Agent      AgentSpec      `yaml:"agent"`
	Tools      []ToolSpec     `yaml:"tools,omitempty"`
	Database   *DBSpec        `yaml:"database,omitempty"`
	MCPServers []MCPDef       `yaml:"mcp,omitempty"`
	Hooks      *HooksSpec     `yaml:"hooks,omitempty"`
}

// AgentSpec defines agent behavior.
type AgentSpec struct {
	Model             string            `yaml:"model"`
	FallbackModels    []string          `yaml:"fallback_models,omitempty"`
	SystemPrompt      string            `yaml:"system_prompt"`
	MaxTokens         int               `yaml:"max_tokens,omitempty"`
	MaxToolIterations int               `yaml:"max_tool_iterations,omitempty"`
	Commands          map[string]string `yaml:"commands,omitempty"`
}

// ToolSpec enables or configures a tool. Supports two formats:
// v1: {name: "tool_name", enabled: true}
// v2: {type: "database", path: "./data.db"}
type ToolSpec struct {
	// v1 fields (backward compatible)
	Name    string `yaml:"name,omitempty"`
	Enabled bool   `yaml:"enabled,omitempty"`

	// v2 fields (type-based configuration)
	Type    string   `yaml:"type,omitempty"`
	Path    string   `yaml:"path,omitempty"`
	Command string   `yaml:"command,omitempty"`
	Args    []string `yaml:"args,omitempty"`
	Docs    []string `yaml:"docs,omitempty"`
	URL     string   `yaml:"url,omitempty"`
	Tools   []string `yaml:"tools,omitempty"`
}

// IsTypeBased returns true if this tool spec uses v2 type-based configuration.
func (t *ToolSpec) IsTypeBased() bool {
	return t.Type != ""
}

// DBSpec defines database connection.
type DBSpec struct {
	Path string `yaml:"path"`
}

// MCPDef defines an MCP server.
type MCPDef struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command,omitempty"`
	Args    []string `yaml:"args,omitempty"`
	URL     string   `yaml:"url,omitempty"`
}

// HooksSpec defines lifecycle hooks.
type HooksSpec struct {
	PreToolUse  *HookDef `yaml:"pre_tool_use,omitempty"`
	PostToolUse *HookDef `yaml:"post_tool_use,omitempty"`
}

// HookDef defines a single hook.
type HookDef struct {
	Command string `yaml:"command"`
}

// LoadYAML parses a YAML agent config file.
func LoadYAML(path string) (*AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading YAML config: %w", err)
	}

	var cfg AgentConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing YAML config: %w", err)
	}

	return &cfg, nil
}

// IsYAMLConfig returns true if the path looks like a YAML config file.
func IsYAMLConfig(path string) bool {
	return strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")
}

// ToCoreConfig converts AgentConfig to the core config.Config format.
func (c *AgentConfig) ToCoreConfig() *corecfg.Config {
	maxTokens := c.Agent.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}

	cfg := &corecfg.Config{
		Agents: corecfg.AgentConfig{
			Defaults: corecfg.AgentDefaults{
				ModelName:      c.Agent.Model,
				MaxTokens:      maxTokens,
				SystemPrompt:   c.Agent.SystemPrompt,
				FallbackModels: c.Agent.FallbackModels,
			},
		},
	}

	// Wire hooks from YAML config
	if c.Hooks != nil {
		if c.Hooks.PreToolUse != nil {
			cfg.Agents.Defaults.PreToolHook = c.Hooks.PreToolUse.Command
		}
		if c.Hooks.PostToolUse != nil {
			cfg.Agents.Defaults.PostToolHook = c.Hooks.PostToolUse.Command
		}
	}

	// Build model list from the agent model
	if c.Agent.Model != "" {
		parts := strings.SplitN(c.Agent.Model, "/", 2)
		vendor := ""
		modelID := c.Agent.Model
		if len(parts) == 2 {
			vendor = parts[0]
			modelID = parts[1]
		}
		cfg.ModelList = []corecfg.ModelEntry{
			{
				ModelName: modelID,
				Model:     c.Agent.Model,
				APIKey:    os.Getenv(strings.ToUpper(vendor) + "_API_KEY"),
			},
		}
	}

	return cfg
}
