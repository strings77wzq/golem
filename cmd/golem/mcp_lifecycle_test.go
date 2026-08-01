package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/strings77wzq/golem/feature/mcp"
)

// TestHelperMCPServerCmd is a subprocess entrypoint running a real MCP server
// over stdio, used by TestLoadMCPToolsContextCleanup.
func TestHelperMCPServerCmd(t *testing.T) {
	if os.Getenv("GOLEM_MCP_HELPER_CMD") != "1" {
		return
	}

	srv := sdk.NewServer(&sdk.Implementation{Name: "helper", Version: "1.0.0"}, nil)
	srv.AddTool(&sdk.Tool{
		Name:        "ping",
		Description: "Returns pong",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: "pong"}}}, nil
	})

	if err := srv.Run(context.Background(), &sdk.StdioTransport{}); err != nil {
		t.Fatalf("helper server failed: %v", err)
	}
}

func TestLoadMCPToolsContextCleanup(t *testing.T) {
	t.Setenv("GOLEM_MCP_HELPER_CMD", "1")

	ctx, cancel := context.WithCancel(context.Background())

	manager, err := LoadMCPTools(ctx, MCPConfig{
		Servers: []mcp.ServerConfig{{
			Name:    "helper",
			Command: os.Args[0],
			Args:    []string{"-test.run=TestHelperMCPServerCmd"},
		}},
	})
	if err != nil {
		t.Fatalf("LoadMCPTools: %v", err)
	}
	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	proxies, err := manager.DiscoverTools(context.Background())
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(proxies) != 1 || proxies[0].Name() != "mcp_helper_ping" {
		t.Fatalf("expected mcp_helper_ping, got %+v", proxies)
	}

	// Sanity: the proxy works while the connection is alive.
	if result, err := proxies[0].Execute(context.Background(), map[string]interface{}{}); err != nil || result.IsError {
		t.Fatalf("proxy call failed while alive: %v %+v", err, result)
	}

	// Cancelling the LoadMCPTools context must tear down the connection.
	cancel()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := proxies[0].Execute(context.Background(), map[string]interface{}{}); err != nil {
			if strings.Contains(err.Error(), "not initialized") ||
				strings.Contains(err.Error(), "connection") ||
				strings.Contains(err.Error(), "closed") {
				return // connection torn down as expected
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("connection still usable after context cancellation")
}
