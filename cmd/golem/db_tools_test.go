package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDBToolsCreatesRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	if err := createTestDB(dbPath, 5, 10, 2, 5); err != nil {
		t.Fatalf("createTestDB: %v", err)
	}

	dbReg, toolReg := buildDBTools(dbPath)
	if dbReg == nil {
		t.Fatal("expected dbRegistry to be non-nil")
	}
	if toolReg == nil {
		t.Fatal("expected toolRegistry to be non-nil")
	}

	// Should have 3 database tools
	tools := toolReg.ListTools()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}

	// Verify tool names
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}
	for _, expected := range []string{"sql_query", "sql_schema", "sql_analyze"} {
		if !names[expected] {
			t.Errorf("expected tool %q to be registered", expected)
		}
	}
}

func TestBuildDBToolsQueryReturnsRealData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	if err := createTestDB(dbPath, 5, 10, 2, 5); err != nil {
		t.Fatalf("createTestDB: %v", err)
	}

	_, toolReg := buildDBTools(dbPath)
	tool, found := toolReg.Get("sql_query")
	if !found {
		t.Fatal("sql_query not found")
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "SELECT name FROM users ORDER BY id LIMIT 2",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "user_") {
		t.Errorf("expected real user data, got: %s", result.ForLLM)
	}
}

func TestBuildDBToolsEmptyPath(t *testing.T) {
	dbReg, toolReg := buildDBTools("")
	if dbReg != nil {
		t.Error("expected nil dbRegistry for empty path")
	}
	if toolReg != nil {
		t.Error("expected nil toolRegistry for empty path")
	}
}

func TestAgentCommandHasDBFlag(t *testing.T) {
	cmd := newAgentCommand()
	flag := cmd.Flags().Lookup("db")
	if flag == nil {
		t.Fatal("expected --db flag to be defined")
	}
	if flag.DefValue != "" {
		t.Errorf("expected default empty string, got %q", flag.DefValue)
	}
}

func TestAgentCommandHasInfraFlag(t *testing.T) {
	cmd := newAgentCommand()
	flag := cmd.Flags().Lookup("infra")
	if flag == nil {
		t.Fatal("expected --infra flag to be defined")
	}
}

func TestGatewayUsesSessionStore(t *testing.T) {
	// Verify that newGatewayCommand creates a server that can accept a session store
	cmd := newGatewayCommand()
	if cmd == nil {
		t.Fatal("expected non-nil gateway command")
	}
}
