package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/foundation/logger"
)

type providerPreset struct {
	label   string
	vendor  string
	apiBase string
	model   string
}

var providerPresets = []providerPreset{
	{"Ollama (local, no key needed)", "ollama", "http://localhost:11434", "ollama/llama3"},
	{"OpenAI (GPT-4o)", "openai", "", "openai/gpt-4o"},
	{"Anthropic (Claude 3.5 Sonnet)", "anthropic", "", "anthropic/claude-3-5-sonnet-20241022"},
	{"DeepSeek (V4)", "deepseek", "https://api.deepseek.com", "deepseek/deepseek-v4-flash"},
	{"MiMo (Xiaomi)", "mimo", "https://api.xiaomimimo.com/v1", "mimo/mimo-v2.5-pro"},
	{"Moonshot (Kimi)", "moonshot", "https://api.moonshot.cn", "moonshot/moonshot-v1-8k"},
	{"Zhipu (GLM)", "zhipu", "https://open.bigmodel.cn/api/paas", "zhipu/glm-4"},
	{"MiniMax", "minimax", "https://api.minimax.chat", "minimax/abab6.5s-chat"},
	{"DashScope (Qwen)", "dashscope", "https://dashscope.aliyuncs.com/compatible-mode", "dashscope/qwen-turbo"},
}

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Configure Golem for first use",
		Long:  "Interactive setup wizard: choose a provider, set your API key, and write config",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := getConfigPath(cmd)
			if err != nil {
				return err
			}
			format, _ := cmd.Flags().GetString("format")
			return runOnboardWizard(configPath, format)
		},
	}
	cmd.Flags().String("format", "yaml", "Config format: yaml (default) or json")
	return cmd
}

func runOnboardWizard(configPath string, format string) error {
	log := logger.New(logger.DefaultOptions())
	r := bufio.NewReader(os.Stdin)

	if _, err := os.Stat(configPath); err == nil {
		log.Info("config already exists", "path", configPath)
		fmt.Print("Reconfigure? [y/N]: ")
		ans, _ := r.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(ans)) != "y" {
			log.Info("keeping existing config")
			return nil
		}
	}

	fmt.Println("=== Golem setup ===")
	fmt.Println("Choose a provider:")
	for i, p := range providerPresets {
		fmt.Printf("  %d. %s\n", i+1, p.label)
	}

	preset, err := choosePreset(r)
	if err != nil {
		return err
	}

	apiKey := ""
	if preset.vendor != "ollama" {
		fmt.Printf("API key for %s: ", preset.label)
		apiKey, err = readLine(r)
		if err != nil {
			return err
		}
		apiKey = strings.TrimSpace(apiKey)
		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}
	} else {
		fmt.Println("Ollama requires no API key — running locally.")
	}

	apiBase := preset.apiBase
	if apiBase != "" {
		fmt.Printf("API base URL [%s]: ", apiBase)
		custom, _ := readLine(r)
		custom = strings.TrimSpace(custom)
		if custom != "" {
			apiBase = custom
		}
	}

	modelName := preset.model
	fmt.Printf("Default model name [%s]: ", modelName)
	customModel, _ := readLine(r)
	customModel = strings.TrimSpace(customModel)
	if customModel != "" {
		if !strings.Contains(customModel, "/") {
			customModel = preset.vendor + "/" + customModel
		}
		modelName = customModel
	}

	if err := ensureConfigDir(configPath); err != nil {
		return err
	}

	if err := writeInitConfig(configPath, format, modelName, apiKey, apiBase, 4096); err != nil {
		return err
	}

	// Connectivity validation
	fmt.Println()
	if err := validateProviderConnectivity(preset.vendor, apiBase, apiKey); err != nil {
		log.Warn("connectivity check failed", "error", err)
	} else {
		log.Info("connectivity check passed")
	}

	// Next steps
	fmt.Println()
	fmt.Println(nextStepsMessage(modelName))
	return nil
}

func writeInitConfig(configPath, format, modelName, apiKey, apiBase string, maxTokens int) error {
	cfg := &config.Config{
		Agents: config.AgentConfig{
			Defaults: config.AgentDefaults{
				ModelName: modelName,
				MaxTokens: maxTokens,
			},
		},
		ModelList: []config.ModelEntry{
			{
				ModelName: modelName,
				Model:     modelName,
				APIKey:    apiKey,
				APIBase:   apiBase,
			},
		},
	}

	if format == "yaml" {
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("serializing config: %w", err)
		}
		return os.WriteFile(configPath, data, 0600)
	}

	// JSON fallback
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}
	return os.WriteFile(configPath, data, 0600)
}

func validateProviderConnectivity(vendor, apiBase, apiKey string) error {
	if vendor == "ollama" {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(apiBase + "/api/tags")
		if err != nil {
			return fmt.Errorf("ollama not reachable at %s — is it running?", apiBase)
		}
		defer resp.Body.Close() //nolint:errcheck
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("ollama returned status %d", resp.StatusCode)
		}
		return nil
	}

	// For cloud providers, check API key presence
	if apiKey == "" {
		envKey := vendorEnvKey(vendor)
		if envKey == "" {
			return nil // can't validate unknown vendor
		}
		if os.Getenv(envKey) == "" {
			return fmt.Errorf("no API key provided and %s not set in environment", envKey)
		}
	}
	return nil
}

func vendorEnvKey(vendor string) string {
	switch vendor {
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "deepseek":
		return "DEEPSEEK_API_KEY"
	case "mimo":
		return "MIMO_API_KEY"
	case "moonshot":
		return "MOONSHOT_API_KEY"
	case "zhipu":
		return "ZHIPU_API_KEY"
	case "minimax":
		return "MINIMAX_API_KEY"
	case "dashscope":
		return "DASHSCOPE_API_KEY"
	default:
		return ""
	}
}

func nextStepsMessage(modelName string) string {
	return fmt.Sprintf(`Next steps:
  1. Test your setup:  golem agent -m "hello"
  2. Start interactive: golem agent
  3. Check status:      golem status

  Model: %s`, modelName)
}

func choosePreset(r *bufio.Reader) (providerPreset, error) {
	for {
		fmt.Printf("Enter number (1-%d): ", len(providerPresets))
		line, err := readLine(r)
		if err != nil {
			return providerPreset{}, err
		}
		line = strings.TrimSpace(line)
		var n int
		if _, err := fmt.Sscanf(line, "%d", &n); err == nil {
			if n >= 1 && n <= len(providerPresets) {
				return providerPresets[n-1], nil
			}
		}
		fmt.Println("Invalid choice, try again.")
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}
