package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigValidate_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	content := `{
		"agents": {
			"defaults": {
				"model_name": "deepseek/deepseek-chat",
				"max_tokens": 4096
			}
		},
		"model_list": [
			{
				"model_name": "deepseek/deepseek-chat",
				"model": "deepseek/deepseek-chat",
				"api_key": "sk-test"
			}
		]
	}`
	os.WriteFile(configPath, []byte(content), 0644)

	result := validateConfigFile(configPath)
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

func TestConfigValidate_MissingModelName(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	content := `{
		"agents": {
			"defaults": {
				"max_tokens": 4096
			}
		},
		"model_list": []
	}`
	os.WriteFile(configPath, []byte(content), 0644)

	result := validateConfigFile(configPath)
	if result.Valid {
		t.Error("expected invalid due to missing model_name")
	}
}

func TestConfigValidate_InvalidModelFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	content := `{
		"agents": {
			"defaults": {
				"model_name": "invalid-model-no-slash"
			}
		},
		"model_list": [
			{
				"model_name": "invalid-model-no-slash",
				"model": "invalid-model-no-slash"
			}
		]
	}`
	os.WriteFile(configPath, []byte(content), 0644)

	result := validateConfigFile(configPath)
	if result.Valid {
		t.Error("expected invalid due to model format (missing vendor/)")
	}
}

func TestConfigValidate_MissingFile(t *testing.T) {
	result := validateConfigFile("/nonexistent/config.json")
	if result.Valid {
		t.Error("expected invalid for missing file")
	}
}

func TestConfigValidate_MaxTokensZero(t *testing.T) {
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
				"model_name": "openai/gpt-4",
				"model": "openai/gpt-4"
			}
		]
	}`
	os.WriteFile(configPath, []byte(content), 0644)

	result := validateConfigFile(configPath)
	// max_tokens 0 should still be valid (uses default)
	if !result.Valid {
		t.Errorf("expected valid with zero max_tokens (uses default), got: %v", result.Errors)
	}
}

func TestConfigValidateResult_String(t *testing.T) {
	result := ConfigValidateResult{
		Valid: false,
		Errors: []string{"missing model_name", "invalid model format"},
	}
	s := result.String()
	if len(s) == 0 {
		t.Error("String() should return non-empty string")
	}
}
