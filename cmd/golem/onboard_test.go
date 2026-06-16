package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitDefaultYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "agent.yaml")

	err := writeInitConfig(configPath, "yaml", "deepseek/deepseek-chat", "sk-test", "https://api.deepseek.com", 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "deepseek/deepseek-chat") {
		t.Errorf("YAML should contain model name, got: %s", content)
	}
	if !strings.Contains(content, "sk-test") {
		t.Errorf("YAML should contain API key, got: %s", content)
	}
	if !strings.Contains(content, "https://api.deepseek.com") {
		t.Errorf("YAML should contain API base, got: %s", content)
	}
}

func TestInitJSONBackwardCompat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	err := writeInitConfig(configPath, "json", "openai/gpt-4", "sk-key", "", 4096)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "openai/gpt-4") {
		t.Errorf("JSON should contain model name, got: %s", content)
	}
	if !strings.Contains(content, "sk-key") {
		t.Errorf("JSON should contain API key, got: %s", content)
	}
}

func TestValidateConnectivity_Ollama(t *testing.T) {
	// This test checks the validation function, not actual connectivity
	err := validateProviderConnectivity("ollama", "http://localhost:11434", "")
	// We can't guarantee Ollama is running, so we just check it doesn't panic
	_ = err // err may be non-nil if Ollama isn't running
}

func TestValidateConnectivity_APIKeyPresent(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	err := validateProviderConnectivity("openai", "", "sk-test")
	if err != nil {
		t.Errorf("should not error when API key is provided, got: %v", err)
	}
}

func TestValidateConnectivity_APIKeyMissing(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	err := validateProviderConnectivity("openai", "", "")
	if err == nil {
		t.Error("should error when API key is missing")
	}
}

func TestNextStepsMessage(t *testing.T) {
	msg := nextStepsMessage("deepseek/deepseek-chat")
	if !strings.Contains(msg, "golem agent") {
		t.Errorf("next steps should contain 'golem agent', got: %s", msg)
	}
	if !strings.Contains(msg, "deepseek/deepseek-chat") {
		t.Errorf("next steps should contain model name, got: %s", msg)
	}
}
