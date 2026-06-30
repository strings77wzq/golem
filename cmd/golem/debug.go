package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/internal/wiring"
	"gopkg.in/yaml.v3"
)

func newDebugCommand(registry *tools.Registry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug and inspection commands",
		Long:  "Commands for inspecting golem's internal state, tools, and configuration",
	}

	cmd.AddCommand(newDebugToolsCommand(registry))
	cmd.AddCommand(newDebugConfigCommand(""))

	return cmd
}

func newDebugToolsCommand(registry *tools.Registry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "List all registered tools with descriptions and parameters",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Lazy init: create default registry if none provided
			r := registry
			if r == nil {
				workspace, _ := os.Getwd()
				r = wiring.BuildToolRegistry(workspace)
			}
			return runDebugTools(cmd, r)
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON instead of human-readable format")
	return cmd
}

func newDebugConfigCommand(configPath string) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show parsed configuration (API keys masked)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := configPath
			if path == "" {
				var err error
				path, err = getConfigPath(cmd)
				if err != nil {
					return err
				}
			}
			return runDebugConfig(cmd, path)
		},
	}
}

func runDebugTools(cmd *cobra.Command, registry *tools.Registry) error {
	allTools := registry.ListTools()

	if len(allTools) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "0 tools registered")
		return nil
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		return outputToolsJSON(cmd, allTools)
	}

	return outputToolsHuman(cmd, allTools)
}

func outputToolsHuman(cmd *cobra.Command, allTools []tools.Tool) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%d tools registered (alphabetical order)\n\n", len(allTools))

	for _, t := range allTools {
		def := tools.ToDefinition(t)
		fmt.Fprintf(out, "  %s\n", def.Name)
		fmt.Fprintf(out, "    %s\n", def.Description)
		if len(def.Parameters) > 0 {
			fmt.Fprintf(out, "    parameters:\n")
			for _, p := range def.Parameters {
				req := ""
				if p.Required {
					req = " (required)"
				}
				fmt.Fprintf(out, "      %s: %s%s\n", p.Name, p.Type, req)
				if p.Description != "" {
					fmt.Fprintf(out, "        %s\n", p.Description)
				}
			}
		}
		fmt.Fprintln(out)
	}

	return nil
}

func outputToolsJSON(cmd *cobra.Command, allTools []tools.Tool) error {
	defs := make([]tools.ToolDefinition, len(allTools))
	for i, t := range allTools {
		defs[i] = tools.ToDefinition(t)
	}

	data, err := json.MarshalIndent(defs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling tools to JSON: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func runDebugConfig(cmd *cobra.Command, configPath string) error {
	data, err := os.ReadFile(configPath) // #nosec G304 -- CLI tool, file path from user input
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	// Mask sensitive keys
	masked := maskSensitiveKeys(raw)

	// Output as YAML for readability
	yamlData, err := yaml.Marshal(masked)
	if err != nil {
		return fmt.Errorf("converting to YAML: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(yamlData))
	return nil
}

func maskSensitiveKeys(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			if isSensitiveKey(k) {
				result[k] = "***"
			} else {
				result[k] = maskSensitiveKeys(v)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = maskSensitiveKeys(v)
		}
		return result
	default:
		return v
	}
}

func isSensitiveKey(key string) bool {
	sensitive := []string{
		"api_key", "apikey", "api-key",
		"secret", "token", "password",
		"auth_token", "auth-token",
		"access_key", "access-key",
		"private_key", "private-key",
	}
	for _, s := range sensitive {
		if key == s {
			return true
		}
	}
	return false
}
