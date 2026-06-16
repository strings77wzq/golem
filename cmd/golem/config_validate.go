package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// ConfigValidateResult holds the outcome of config validation.
type ConfigValidateResult struct {
	Valid  bool
	Errors []string
}

func (r ConfigValidateResult) String() string {
	if r.Valid {
		return "✓ valid"
	}
	return "✗ " + strings.Join(r.Errors, "\n  ✗ ")
}

func newConfigValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long:  "Load and validate the golem configuration, checking required fields and model formats",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}

			result := validateConfigFile(configPath)
			out := cmd.OutOrStdout()

			if result.Valid {
				fmt.Fprintln(out, "✓ Configuration is valid")
				fmt.Fprintf(out, "  Config: %s\n", configPath)
			} else {
				fmt.Fprintf(out, "✗ Configuration has %d error(s):\n", len(result.Errors))
				for _, e := range result.Errors {
					fmt.Fprintf(out, "  ✗ %s\n", e)
				}
				return fmt.Errorf("config validation failed")
			}
			return nil
		},
	}
}

func validateConfigFile(configPath string) ConfigValidateResult {
	var result ConfigValidateResult

	data, err := os.ReadFile(configPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("cannot read config: %v", err))
		return result
	}

	var cfg struct {
		Agents struct {
			Defaults struct {
				ModelName string `json:"model_name"`
				MaxTokens int    `json:"max_tokens"`
			} `json:"defaults"`
		} `json:"agents"`
		ModelList []struct {
			ModelName string `json:"model_name"`
			Model     string `json:"model"`
			APIKey    string `json:"api_key"`
		} `json:"model_list"`
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid JSON: %v", err))
		return result
	}

	// Check model_name is set
	if cfg.Agents.Defaults.ModelName == "" {
		result.Errors = append(result.Errors, "agents.defaults.model_name is required")
	}

	// Validate model name format (should contain /)
	if cfg.Agents.Defaults.ModelName != "" && !strings.Contains(cfg.Agents.Defaults.ModelName, "/") {
		result.Errors = append(result.Errors, fmt.Sprintf(
			"model name %q should be in vendor/model format (e.g., openai/gpt-4)", cfg.Agents.Defaults.ModelName))
	}

	// Validate model_list entries match default
	if len(cfg.ModelList) == 0 && cfg.Agents.Defaults.ModelName != "" {
		result.Errors = append(result.Errors, "model_list is empty but model_name is set — add at least one model entry")
	}

	// Check model_list entries have matching model names
	for _, m := range cfg.ModelList {
		if m.Model == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("model_list entry %q has empty model field", m.ModelName))
		}
	}

	result.Valid = len(result.Errors) == 0
	return result
}
