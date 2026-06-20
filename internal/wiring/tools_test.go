package wiring

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/tools"
)

func TestBuildToolRegistry(t *testing.T) {
	registry := BuildToolRegistry(t.TempDir())
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	defs := registry.ListDefinitions()
	if len(defs) != 6 {
		t.Errorf("expected 6 tools, got %d", len(defs))
	}
	// Verify tool names
	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Name] = true
	}
	for _, expected := range []string{"exec", "file_read", "file_write", "file_list", "think", "web_search"} {
		if !names[expected] {
			t.Errorf("expected tool %q", expected)
		}
	}
}

func TestBuildDBTools(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a test database
	dbRegistry := database.NewRegistry()
	driver := database.NewSQLiteDriver("test", dbPath)
	if err := driver.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	dbRegistry.RegisterSQL("test", driver)
	dbRegistry.SetDefault("test")

	dbReg, toolReg := BuildDBTools(dbPath, nil, nil)
	if dbReg == nil {
		t.Fatal("expected non-nil dbRegistry")
	}
	if toolReg == nil {
		t.Fatal("expected non-nil toolRegistry")
	}

	// Verify 3 database tools
	tools := toolReg.ListTools()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}
	for _, expected := range []string{"sql_query", "sql_schema", "sql_analyze"} {
		if !names[expected] {
			t.Errorf("expected tool %q", expected)
		}
	}
}

func TestBuildDBToolsEmpty(t *testing.T) {
	dbReg, toolReg := BuildDBTools("", nil, nil)
	if dbReg != nil {
		t.Error("expected nil dbRegistry for empty path")
	}
	if toolReg != nil {
		t.Error("expected nil toolRegistry for empty path")
	}
}

func TestRegisterInfraTools(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterInfraTools(registry)

	defs := registry.ListDefinitions()
	if len(defs) != 3 {
		t.Errorf("expected 3 infra tools, got %d", len(defs))
	}

	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Name] = true
	}
	for _, expected := range []string{"kubectl", "docker", "helm"} {
		if !names[expected] {
			t.Errorf("expected tool %q", expected)
		}
	}
}
