package database

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/security"
)

func setupTestDB(t *testing.T) *database.Registry {
	t.Helper()
	reg := database.NewRegistry()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	driver := database.NewSQLiteDriver("test", dbPath)
	if err := driver.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Create test table
	_, err := driver.Execute(context.Background(), "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = driver.Execute(context.Background(), "INSERT INTO users (name, email) VALUES (?, ?)", "alice", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	_, err = driver.Execute(context.Background(), "INSERT INTO users (name, email) VALUES (?, ?)", "bob", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}

	reg.RegisterSQL("test", driver)
	reg.SetDefault("test")
	return reg
}

func TestSQLQueryTool(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLQueryTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "SELECT * FROM users ORDER BY id",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
	if result.ForLLM == "" {
		t.Error("expected non-empty result")
	}
}

func TestSQLQueryToolWithArgs(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLQueryTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql":  "SELECT * FROM users WHERE name = ?",
		"args": []interface{}{"alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
}

func TestSQLQueryToolWriteBlocked(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLQueryTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "DELETE FROM users WHERE id = 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for write operation")
	}
}

func TestSQLQueryToolNoSQL(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLQueryTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing sql")
	}
}

func TestSQLSchemaTool(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLSchemaTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
}

func TestSQLSchemaToolSpecificTable(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLSchemaTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"table": "users",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
}

func TestSQLAnalyzeTool(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLAnalyzeTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"table": "users",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
}

func TestSQLAnalyzeToolNoTable(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLAnalyzeTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing table")
	}
}

func TestToolNames(t *testing.T) {
	reg := database.NewRegistry()

	tools := []struct {
		name string
		tool interface{ Name() string }
	}{
		{"sql_query", NewSQLQueryTool(reg)},
		{"sql_schema", NewSQLSchemaTool(reg)},
		{"sql_analyze", NewSQLAnalyzeTool(reg)},
		{"redis_get", NewRedisGetTool(reg)},
		{"redis_set", NewRedisSetTool(reg)},
		{"redis_keys", NewRedisKeysTool(reg)},
		{"vector_search", NewVectorSearchTool(reg)},
		{"vector_collections", NewVectorCollectionsTool(reg)},
	}

	for _, tt := range tools {
		if tt.tool.Name() != tt.name {
			t.Errorf("tool Name() = %q, want %q", tt.tool.Name(), tt.name)
		}
	}
}

func TestSQLQueryToolTruncation(t *testing.T) {
	reg := setupTestDB(t)
	// Insert 60 rows to trigger truncation
	driver, _ := reg.GetSQL("test")
	for i := 0; i < 60; i++ {
		_, err := driver.Execute(context.Background(),
			"INSERT INTO users (name, email) VALUES (?, ?)",
			fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@example.com", i))
		if err != nil {
			t.Fatal(err)
		}
	}

	tool := NewSQLQueryToolWithMaxRows(reg, 50)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "SELECT * FROM users ORDER BY id",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
	// Should mention truncation
	if !strings.Contains(result.ForLLM, "showing") {
		t.Error("expected truncation notice in result")
	}
}

func TestSQLQueryToolSummary(t *testing.T) {
	reg := setupTestDB(t)
	// Insert 150 rows to trigger summary
	driver, _ := reg.GetSQL("test")
	for i := 0; i < 150; i++ {
		_, err := driver.Execute(context.Background(),
			"INSERT INTO users (name, email) VALUES (?, ?)",
			fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@example.com", i))
		if err != nil {
			t.Fatal(err)
		}
	}

	tool := NewSQLQueryTool(reg)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "SELECT * FROM users",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Should show summary, not raw rows
	if !strings.Contains(result.ForLLM, "Large result set") {
		t.Error("expected summary for large result set")
	}
	if !strings.Contains(result.ForLLM, "152 rows") {
		t.Errorf("expected total row count, got: %s", result.ForLLM[:100])
	}
}

func TestSQLQueryToolSecurityGateWriteBlocked(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLQueryTool(reg)

	// DELETE without permission should be blocked
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "DELETE FROM users WHERE id = 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for DELETE operation")
	}
	if !strings.Contains(result.ForLLM, "Security") && !strings.Contains(result.ForLLM, "permissions") {
		t.Errorf("expected security error message, got: %s", result.ForLLM)
	}
}

func TestSQLQueryToolSecurityGateUpdateWithoutWhere(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLQueryTool(reg)

	// UPDATE without WHERE should be blocked
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "UPDATE users SET name = 'hacked'",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for UPDATE without WHERE")
	}
}

func TestSQLQueryToolSecurityGateSelectAllowed(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLQueryTool(reg)

	// SELECT should always be allowed
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "SELECT * FROM users WHERE id = 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error for SELECT: %s", result.ForLLM)
	}
}

func TestSQLQueryToolSecurityGateDropBlocked(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLQueryTool(reg)

	// DROP TABLE should be blocked
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "DROP TABLE users",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for DROP TABLE")
	}
}

func TestSQLQueryToolWithWritePermission(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLQueryToolWithPermission(reg, security.PermWrite)

	// INSERT should be allowed with write permission
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "INSERT INTO users (name, email) VALUES ('test', 'test@example.com')",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error for INSERT with write permission: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "rows affected") {
		t.Errorf("expected rows affected message, got: %s", result.ForLLM)
	}
}

func TestSQLQueryToolWithWritePermissionUpdateWithoutWhere(t *testing.T) {
	reg := setupTestDB(t)
	tool := NewSQLQueryToolWithPermission(reg, security.PermWrite)

	// UPDATE without WHERE should be blocked even with write permission
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "UPDATE users SET name = 'hacked'",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for UPDATE without WHERE")
	}
	if !strings.Contains(result.ForLLM, "Safety") && !strings.Contains(result.ForLLM, "WHERE") {
		t.Errorf("expected safety error about WHERE clause, got: %s", result.ForLLM)
	}
}
