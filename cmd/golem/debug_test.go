package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/tools"
)

func setupTestRegistry() *tools.Registry {
	registry := tools.NewRegistry()
	registry.Register(&tools.MockTool{
		ToolName:        "test_tool",
		ToolDescription: "A test tool for debugging",
		ToolParameters: []tools.ToolParameter{
			{Name: "query", Type: "string", Description: "The query to execute", Required: true},
		},
	})
	registry.Register(&tools.MockTool{
		ToolName:        "another_tool",
		ToolDescription: "Another test tool",
		ToolParameters: []tools.ToolParameter{
			{Name: "path", Type: "string", Description: "File path", Required: false},
		},
	})
	return registry
}

func TestDebugTools_ListsAllTools(t *testing.T) {
	registry := setupTestRegistry()
	cmd := newDebugToolsCommand(registry)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "another_tool") {
		t.Errorf("output should contain 'another_tool', got: %s", output)
	}
	if !strings.Contains(output, "test_tool") {
		t.Errorf("output should contain 'test_tool', got: %s", output)
	}
	if !strings.Contains(output, "A test tool for debugging") {
		t.Errorf("output should contain tool description, got: %s", output)
	}
}

func TestDebugTools_ShowsParameters(t *testing.T) {
	registry := setupTestRegistry()
	cmd := newDebugToolsCommand(registry)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "query") {
		t.Errorf("output should contain parameter 'query', got: %s", output)
	}
	if !strings.Contains(output, "required") {
		t.Errorf("output should indicate required parameters, got: %s", output)
	}
}

func TestDebugTools_EmptyRegistry(t *testing.T) {
	registry := tools.NewRegistry()
	cmd := newDebugToolsCommand(registry)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "0 tools registered") {
		t.Errorf("output should show 0 tools, got: %s", output)
	}
}

func TestDebugTools_SortedAlphabetically(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&tools.MockTool{ToolName: "zebra", ToolDescription: "Z"})
	registry.Register(&tools.MockTool{ToolName: "alpha", ToolDescription: "A"})
	registry.Register(&tools.MockTool{ToolName: "middle", ToolDescription: "M"})

	cmd := newDebugToolsCommand(registry)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	alphaIdx := strings.Index(output, "alpha")
	middleIdx := strings.Index(output, "middle")
	zebraIdx := strings.Index(output, "zebra")

	if alphaIdx >= middleIdx || middleIdx >= zebraIdx {
		t.Errorf("tools should be sorted alphabetically, got order in output: %s", output)
	}
}

func TestDebugConfig_ShowsYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{
		"agents": {
			"defaults": {
				"model_name": "deepseek-chat",
				"system_prompt": "You are a database expert."
			}
		},
		"model_list": [
			{
				"model_name": "deepseek-chat",
				"model": "deepseek-chat",
				"api_base": "https://api.deepseek.com"
			}
		]
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cmd := newDebugConfigCommand(configPath)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "deepseek-chat") {
		t.Errorf("output should contain model name, got: %s", output)
	}
	if !strings.Contains(output, "database expert") {
		t.Errorf("output should contain system prompt, got: %s", output)
	}
}

func TestDebugConfig_MasksAPIKeys(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{
		"api_key": "sk-secret-api-key-12345",
		"agents": {
			"defaults": {
				"model_name": "gpt-4"
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cmd := newDebugConfigCommand(configPath)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "sk-secret-api-key-12345") {
		t.Errorf("API key should be masked, but found in output: %s", output)
	}
	if !strings.Contains(output, "***") {
		t.Errorf("output should contain masked value '***', got: %s", output)
	}
}

func TestDebugConfig_MissingFile(t *testing.T) {
	cmd := newDebugConfigCommand("/nonexistent/path/config.json")

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestDebugParentCommand(t *testing.T) {
	registry := setupTestRegistry()
	cmd := newDebugCommand(registry)

	if cmd.Use != "debug" {
		t.Errorf("expected Use='debug', got %q", cmd.Use)
	}

	// Should have subcommands
	foundTools := false
	foundConfig := false
	for _, c := range cmd.Commands() {
		switch c.Use {
		case "tools":
			foundTools = true
		case "config":
			foundConfig = true
		}
	}
	if !foundTools {
		t.Error("debug command should have 'tools' subcommand")
	}
	if !foundConfig {
		t.Error("debug command should have 'config' subcommand")
	}
}

func TestDebugTools_FlagJSON(t *testing.T) {
	registry := setupTestRegistry()
	cmd := newDebugToolsCommand(registry)
	cmd.SetArgs([]string{"--json"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"name"`) {
		t.Errorf("JSON output should contain 'name' field, got: %s", output)
	}
	if !strings.Contains(output, `"description"`) {
		t.Errorf("JSON output should contain 'description' field, got: %s", output)
	}
}


