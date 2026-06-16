package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Bug: Config validate rejects valid model names with multiple slashes
// like "bedrock/us.anthropic.claude-3-5-sonnet-20241022-v2:0" which is valid
// for AWS Bedrock but fails the single-slash format check.
func TestConfigValidate_MultiSlashModelName(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	content := `{
		"agents": {
			"defaults": {
				"model_name": "bedrock/us.anthropic.claude-3-5-sonnet-20241022-v2:0"
			}
		},
		"model_list": [
			{
				"model_name": "bedrock/us.anthropic.claude-3-5-sonnet-20241022-v2:0",
				"model": "bedrock/us.anthropic.claude-3-5-sonnet-20241022-v2:0"
			}
		]
	}`
	os.WriteFile(configPath, []byte(content), 0644)

	result := validateConfigFile(configPath)
	if !result.Valid {
		t.Errorf("multi-slash model name should be valid, got errors: %v", result.Errors)
	}
}

// Bug: Config validate doesn't check if model_list entry matches default model.
func TestConfigValidate_ModelMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	content := `{
		"agents": {
			"defaults": {
				"model_name": "openai/gpt-4"
			}
		},
		"model_list": [
			{
				"model_name": "anthropic/claude-3",
				"model": "anthropic/claude-3"
			}
		]
	}`
	os.WriteFile(configPath, []byte(content), 0644)

	result := validateConfigFile(configPath)
	// Should warn that default model not in model_list
	if result.Valid {
		t.Error("config with mismatched default model and model_list should be invalid")
	}
}
