package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/strings77wzq/golem/core/config"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "Manage Golem configuration file",
	}

	cmd.AddCommand(
		newConfigSetCommand(),
		newConfigGetCommand(),
		newConfigListCommand(),
		newConfigValidateCommand(),
		newConfigModelCommand(),
	)

	return cmd
}

func newConfigModelCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "model",
		Short: "Configure LLM provider interactively",
		Long:  "Set up your LLM provider: base URL, model name, and API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}
			return runConfigModelWizard(configPath)
		},
	}
}

func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}

			if err := ensureConfigDir(configPath); err != nil {
				return err
			}

			cfg := make(map[string]interface{})
			if data, err := os.ReadFile(configPath); err == nil {
				if err := json.Unmarshal(data, &cfg); err != nil {
					return fmt.Errorf("parsing existing config: %w", err)
				}
			}

			cfg[key] = value

			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}

			if err := os.WriteFile(configPath, data, 0644); err != nil {
				return fmt.Errorf("writing config: %w", err)
			}

			fmt.Printf("Set %s = %s\n", key, value)
			return nil
		},
	}
}

func newConfigGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}

			data, err := os.ReadFile(configPath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("config file does not exist: %s", configPath)
				}
				return fmt.Errorf("reading config: %w", err)
			}

			cfg := make(map[string]interface{})
			if err := json.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("parsing config: %w", err)
			}

			value, ok := cfg[key]
			if !ok {
				return fmt.Errorf("key %q not found in config", key)
			}

			fmt.Println(value)
			return nil
		},
	}
}

func newConfigListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}

			data, err := os.ReadFile(configPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("{}")
					return nil
				}
				return fmt.Errorf("reading config: %w", err)
			}

			var cfg interface{}
			if err := json.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("parsing config: %w", err)
			}

			formatted, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return fmt.Errorf("formatting config: %w", err)
			}

			fmt.Println(string(formatted))
			return nil
		},
	}
}

func getConfigPath(cmd *cobra.Command) (string, error) {
	configPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home directory: %w", err)
		}
		configPath = filepath.Join(home, ".golem", "config.json")
	}
	return configPath, nil
}

func ensureConfigDir(configPath string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	return nil
}

// runConfigModelWizard is the 3-step interactive LLM configuration wizard.
// Step 1: Base URL (chat-compatible or Anthropic-compatible)
// Step 2: Model name
// Step 3: API key
func runConfigModelWizard(configPath string) error {
	r := bufio.NewReader(os.Stdin)

	fmt.Println("=== Configure LLM Provider ===")
	fmt.Println()

	// Step 1: Base URL
	fmt.Print("Base URL (e.g. https://api.deepseek.com): ")
	baseURL, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading base URL: %w", err)
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	// Step 2: Model name
	fmt.Print("Model name (e.g. deepseek-chat): ")
	model, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading model name: %w", err)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	// Step 3: API key
	fmt.Print("API key: ")
	apiKey, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading API key: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Derive vendor from base URL
	vendor := deriveVendor(baseURL)
	fullModel := vendor + "/" + model

	// Load existing config or create new
	cfg, loadErr := config.Load(configPath)
	if loadErr != nil || cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Update config
	cfg.Agents.Defaults.ModelName = fullModel
	cfg.ModelList = []config.ModelEntry{{
		ModelName: fullModel,
		Model:     fullModel,
		APIKey:    apiKey,
		APIBase:   baseURL,
	}}

	// Write config
	if err := ensureConfigDir(configPath); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Println()
	fmt.Printf("✅ Config saved to %s\n", configPath)
	fmt.Printf("   Provider: %s\n", vendor)
	fmt.Printf("   Model:    %s\n", fullModel)
	fmt.Printf("   Base URL: %s\n", baseURL)
	fmt.Println()
	fmt.Printf("Try it: golem agent -m \"Hello\"\n")

	return nil
}

// deriveVendor infers the vendor name from the base URL.
func deriveVendor(baseURL string) string {
	lower := strings.ToLower(baseURL)
	switch {
	case strings.Contains(lower, "deepseek"):
		return "deepseek"
	case strings.Contains(lower, "openai"):
		return "openai"
	case strings.Contains(lower, "anthropic"):
		return "anthropic"
	case strings.Contains(lower, "moonshot") || strings.Contains(lower, "kimi"):
		return "moonshot"
	case strings.Contains(lower, "bigmodel") || strings.Contains(lower, "zhipu"):
		return "zhipu"
	case strings.Contains(lower, "minimax"):
		return "minimax"
	case strings.Contains(lower, "dashscope") || strings.Contains(lower, "aliyuncs"):
		return "dashscope"
	case strings.Contains(lower, "localhost:11434"):
		return "ollama"
	case strings.Contains(lower, "xiaomimimo"):
		return "mimo"
	default:
		return "custom"
	}
}
