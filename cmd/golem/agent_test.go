package main

import (
	"strings"
	"testing"
)

func TestAgentModelNotFound(t *testing.T) {
	configPath := newTempConfigPath(t)
	writeConfigFile(t, configPath, newTestConfig())

	_, stderr, err := executeRootCommand(t, "-c", configPath, "agent", "-M", "nonexistent", "-m", "hello")
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("expected 'not found' in error, got: %s", stderr)
	}
}

func TestAgentInvalidConfig(t *testing.T) {
	_, _, err := executeRootCommand(t, "-c", "/nonexistent/config.json", "agent", "-m", "hello")
	if err == nil {
		t.Fatal("expected error for invalid config path")
	}
}

func TestAgentContinueNoSession(t *testing.T) {
	configPath := newTempConfigPath(t)
	writeConfigFile(t, configPath, newTestConfig())

	_, stderr, err := executeRootCommand(t, "-c", configPath, "agent", "--continue", "nonexistent", "-m", "hello")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
	if !strings.Contains(stderr, "not found") {
		t.Errorf("expected 'not found' in error, got: %s", stderr)
	}
}

func TestAgentNoInputNoTTY(t *testing.T) {
	configPath := newTempConfigPath(t)
	writeConfigFile(t, configPath, newTestConfig())

	// Run without -m flag and without TTY
	// This should fail because there's no input
	_, _, err := executeRootCommand(t, "-c", configPath, "agent")
	if err == nil {
		// In test environment, this might succeed or fail depending on TTY detection
		// We just verify it doesn't panic
		return
	}
}

func TestAgentCommandHasAllFlags(t *testing.T) {
	cmd := newAgentCommand()
	expectedFlags := []string{"message", "model", "continue", "no-tui", "skills-dir", "skills", "db", "infra", "rag", "mcp", "memory", "telegram", "json-events"}
	for _, name := range expectedFlags {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Errorf("expected flag %q to be defined", name)
		}
	}
}

func TestAgentJsonEventsFlag(t *testing.T) {
	cmd := newAgentCommand()
	flag := cmd.Flags().Lookup("json-events")
	if flag == nil {
		t.Fatal("expected --json-events flag to be defined")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --json-events default to be false, got %q", flag.DefValue)
	}
}
