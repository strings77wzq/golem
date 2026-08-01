package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/strings77wzq/golem/feature/mcp"
)

func TestBuildMCPToolsWithRealDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create test database
	if err := createTestDB(dbPath, 10, 20, 5, 30); err != nil {
		t.Fatalf("createTestDB: %v", err)
	}

	// Build real tools
	registry := buildMCPTools(dbPath, false, "", nil)
	if registry.Count() == 0 {
		t.Fatal("expected tools to be registered")
	}

	// Verify sql_query is a real tool (not a stub)
	tool, found := registry.Get("sql_query")
	if !found {
		t.Fatal("sql_query tool not found")
	}

	// Execute real SQL query
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"sql": "SELECT COUNT(*) as cnt FROM users",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "10") {
		t.Errorf("expected count 10 in result, got: %s", result.ForLLM)
	}
}

func TestBuildMCPToolsReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	if err := createTestDB(dbPath, 5, 10, 2, 5); err != nil {
		t.Fatalf("createTestDB: %v", err)
	}

	registry := buildMCPTools(dbPath, true, "", nil)

	// exec should NOT be present in read-only mode
	_, found := registry.Get("exec")
	if found {
		t.Error("exec tool should not be registered in read-only mode")
	}

	// file_write should NOT be present in read-only mode
	_, found = registry.Get("file_write")
	if found {
		t.Error("file_write tool should not be registered in read-only mode")
	}

	// sql_query should be present
	_, found = registry.Get("sql_query")
	if !found {
		t.Error("sql_query tool should be registered")
	}
}

func TestBuildMCPToolsFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	if err := createTestDB(dbPath, 5, 10, 2, 5); err != nil {
		t.Fatalf("createTestDB: %v", err)
	}

	registry := buildMCPTools(dbPath, false, "sql_query,sql_schema", nil)

	// Only sql_query and sql_schema should be present
	if registry.Count() != 2 {
		t.Errorf("expected 2 tools, got %d", registry.Count())
	}

	_, found := registry.Get("sql_query")
	if !found {
		t.Error("sql_query should be registered")
	}
	_, found = registry.Get("sql_schema")
	if !found {
		t.Error("sql_schema should be registered")
	}
	_, found = registry.Get("sql_analyze")
	if found {
		t.Error("sql_analyze should not be registered (not in filter)")
	}
}

func TestMCPServerEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	if err := createTestDB(dbPath, 5, 10, 2, 5); err != nil {
		t.Fatalf("createTestDB: %v", err)
	}

	registry := buildMCPTools(dbPath, false, "sql_query", nil)
	server := mcp.NewServer(registry)

	// Wire the server to a client over in-memory transports.
	clientTransport, serverTransport := sdk.NewInMemoryTransports()

	srvCtx, srvCancel := context.WithCancel(context.Background())
	defer srvCancel()
	if _, err := server.Connect(srvCtx, serverTransport); err != nil {
		t.Fatalf("server connect: %v", err)
	}

	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1.0"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close() //nolint:errcheck

	// tools/list must expose the registered sql_query tool.
	listResult, err := session.ListTools(context.Background(), &sdk.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	found := false
	for _, tool := range listResult.Tools {
		if tool.Name == "sql_query" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("sql_query not exposed by MCP server")
	}

	// tools/call must execute against the real database.
	callResult, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "sql_query",
		Arguments: json.RawMessage(`{"sql": "SELECT name FROM users ORDER BY id LIMIT 2"}`),
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if callResult.IsError {
		t.Fatalf("unexpected error result: %+v", callResult)
	}
	text := ""
	for _, block := range callResult.Content {
		if tc, ok := block.(*sdk.TextContent); ok {
			text += tc.Text
		}
	}
	if strings.Contains(text, "executed with input") {
		t.Errorf("got stub result, not real data: %s", text)
	}
	if !strings.Contains(text, "user_") {
		t.Errorf("expected real user data, got: %s", text)
	}
}
