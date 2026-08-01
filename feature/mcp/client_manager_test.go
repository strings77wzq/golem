package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/tools"
)

func TestManagerAddServer(t *testing.T) {
	manager := NewManager()

	cfg := ServerConfig{
		Name:    "test-server",
		Command: "echo",
		Args:    []string{"hello"},
	}

	if err := manager.AddServer(cfg); err != nil {
		t.Fatalf("failed to add server: %v", err)
	}

	t.Run("duplicate server name", func(t *testing.T) {
		err := manager.AddServer(cfg)
		if err == nil {
			t.Fatal("expected error for duplicate server, got nil")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error, got %v", err)
		}
	})

	t.Run("empty server name", func(t *testing.T) {
		err := manager.AddServer(ServerConfig{Command: "test"})
		if err == nil {
			t.Fatal("expected error for empty server name, got nil")
		}
		if !strings.Contains(err.Error(), "name is required") {
			t.Errorf("expected 'name is required' error, got %v", err)
		}
	})

	t.Run("empty command", func(t *testing.T) {
		err := manager.AddServer(ServerConfig{Name: "test2"})
		if err == nil {
			t.Fatal("expected error for empty command, got nil")
		}
		if !strings.Contains(err.Error(), "command is required") {
			t.Errorf("expected 'command is required' error, got %v", err)
		}
	})
}

func TestManagerCallToolUninitialized(t *testing.T) {
	manager := NewManager()
	if err := manager.AddServer(ServerConfig{Name: "srv", Command: "echo"}); err != nil {
		t.Fatalf("failed to add server: %v", err)
	}

	_, err := manager.callTool(context.Background(), "srv", "tool", nil)
	if err == nil {
		t.Fatal("expected error for uninitialized server, got nil")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("expected 'not initialized' error, got %v", err)
	}
}

func TestManagerCallToolUnknownServer(t *testing.T) {
	manager := NewManager()
	_, err := manager.callTool(context.Background(), "ghost", "tool", nil)
	if err == nil {
		t.Fatal("expected error for unknown server, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}

func TestManagerCloseWithoutStart(t *testing.T) {
	manager := NewManager()
	if err := manager.AddServer(ServerConfig{Name: "srv", Command: "echo"}); err != nil {
		t.Fatalf("failed to add server: %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("close without start should succeed: %v", err)
	}
}

func TestManagerStartReportsConnectionFailure(t *testing.T) {
	manager := NewManager()
	// A command that exits immediately without speaking MCP.
	if err := manager.AddServer(ServerConfig{Name: "broken", Command: "true"}); err != nil {
		t.Fatalf("failed to add server: %v", err)
	}

	err := manager.Start(context.Background())
	if err == nil {
		t.Fatal("expected error connecting to non-MCP command, got nil")
	}
	if !strings.Contains(err.Error(), "failed to connect server broken") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMCPToolProxy(t *testing.T) {
	inputSchemaJSON := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string", "description": "Search query"},
			"limit": {"type": "number", "description": "Result limit"}
		},
		"required": ["query"]
	}`)

	mcpTool := MCPTool{
		Name:        "search",
		Description: "Search for items",
		InputSchema: inputSchemaJSON,
	}

	manager := NewManager()
	proxy := MCPToolProxy{
		serverName: "test-server",
		mcpTool:    mcpTool,
		manager:    manager,
	}

	t.Run("name formatting", func(t *testing.T) {
		expected := "mcp_test-server_search"
		if name := proxy.Name(); name != expected {
			t.Errorf("expected name %s, got %s", expected, name)
		}
	})

	t.Run("description", func(t *testing.T) {
		if desc := proxy.Description(); desc != "Search for items" {
			t.Errorf("expected 'Search for items', got %s", desc)
		}
	})

	t.Run("parameters", func(t *testing.T) {
		params := proxy.Parameters()
		if len(params) != 2 {
			t.Fatalf("expected 2 parameters, got %d", len(params))
		}

		foundQuery := false
		foundLimit := false
		for _, param := range params {
			switch param.Name {
			case "query":
				foundQuery = true
				if param.Type != "string" || !param.Required {
					t.Errorf("query param mismatch: %+v", param)
				}
			case "limit":
				foundLimit = true
				if param.Type != "number" || param.Required {
					t.Errorf("limit param mismatch: %+v", param)
				}
			}
		}
		if !foundQuery || !foundLimit {
			t.Errorf("missing params: query=%v limit=%v", foundQuery, foundLimit)
		}
	})

	t.Run("invalid schema yields nil parameters", func(t *testing.T) {
		bad := MCPToolProxy{serverName: "s", mcpTool: MCPTool{Name: "x", InputSchema: json.RawMessage(`not-json`)}, manager: manager}
		if params := bad.Parameters(); params != nil {
			t.Errorf("expected nil parameters for invalid schema, got %v", params)
		}
	})

	t.Run("implements Tool interface", func(t *testing.T) {
		var _ tools.Tool = proxy
	})
}
