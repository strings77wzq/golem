package e2e

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// jsonEvent matches the structure emitted by --json-events.
type jsonEvent struct {
	Type       string `json:"type"`
	ToolName   string `json:"tool_name,omitempty"`
	ToolInput  string `json:"tool_input,omitempty"`
	ToolOutput string `json:"tool_output,omitempty"`
}

// writeOllamaConfig creates a temporary config file with Ollama model entry.
func writeOllamaConfig(t *testing.T) string {
	t.Helper()
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.json")

	config := map[string]interface{}{
		"agents": map[string]interface{}{
			"defaults": map[string]interface{}{
				"model_name":    "ollama/qwen3:0.5b",
				"max_tokens":    4096,
				"system_prompt": "You are Golem, a helpful AI assistant that can query databases. Always use the sql_query tool to interact with databases.",
			},
		},
		"model_list": []map[string]interface{}{
			{
				"model_name": "ollama/qwen3:0.5b",
				"model":      "ollama/qwen3:0.5b",
				"api_key":    "",
				"api_base":   "http://127.0.0.1:11434",
			},
		},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

// parseJSONEvents parses JSON lines from stderr into jsonEvent structs.
func parseJSONEvents(t *testing.T, content string) []jsonEvent {
	t.Helper()
	var events []jsonEvent
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var evt jsonEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}
		if evt.Type != "" {
			events = append(events, evt)
		}
	}
	return events
}
