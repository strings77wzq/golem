package mcp

import (
	"context"
	"os"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestHelperMCPServer is a subprocess entrypoint that runs a real MCP server
// over stdio. It is launched by TestDelegateExecute and TestManagerStart by
// re-executing the test binary with GOLEM_MCP_HELPER=1.
func TestHelperMCPServer(t *testing.T) {
	if os.Getenv("GOLEM_MCP_HELPER") != "1" {
		return
	}

	srv := sdk.NewServer(&sdk.Implementation{Name: "helper", Version: "1.0.0"}, nil)
	srv.AddTool(&sdk.Tool{
		Name:        "ping",
		Description: "Returns pong",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return &sdk.CallToolResult{
			Content: []sdk.Content{&sdk.TextContent{Text: "pong"}},
		}, nil
	})

	if err := srv.Run(context.Background(), &sdk.StdioTransport{}); err != nil {
		t.Fatalf("helper server failed: %v", err)
	}
}

func helperArgs() []string {
	return []string{"-test.run=TestHelperMCPServer"}
}

func TestDelegateExecuteSuccess(t *testing.T) {
	if os.Getenv("GOLEM_MCP_HELPER") != "1" {
		t.Setenv("GOLEM_MCP_HELPER", "1")
	}

	tool := NewDelegateTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command":   os.Args[0],
		"args":      []interface{}{"-test.run=TestHelperMCPServer"},
		"tool":      "ping",
		"arguments": map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "pong") {
		t.Errorf("expected pong, got %q", result.ForLLM)
	}
}

func TestDelegateExecuteCommandFailure(t *testing.T) {
	tool := NewDelegateTool()
	// A command that exits immediately without speaking MCP.
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command":   "true",
		"tool":      "ping",
		"arguments": map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for non-MCP command")
	}
}

func TestManagerStartAndProxy(t *testing.T) {
	t.Setenv("GOLEM_MCP_HELPER", "1")

	manager := NewManager()
	if err := manager.AddServer(ServerConfig{
		Name:    "helper",
		Command: os.Args[0],
		Args:    helperArgs(),
	}); err != nil {
		t.Fatalf("add server: %v", err)
	}

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer manager.Close() //nolint:errcheck

	proxies, err := manager.DiscoverTools(context.Background())
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(proxies) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(proxies))
	}

	proxy := proxies[0]
	if proxy.Name() != "mcp_helper_ping" {
		t.Errorf("expected mcp_helper_ping, got %s", proxy.Name())
	}

	result, err := proxy.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("proxy execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "pong") {
		t.Errorf("expected pong, got %q", result.ForLLM)
	}
}

func TestManagerStartPartialFailure(t *testing.T) {
	t.Setenv("GOLEM_MCP_HELPER", "1")

	manager := NewManager()
	if err := manager.AddServer(ServerConfig{Name: "good", Command: os.Args[0], Args: helperArgs()}); err != nil {
		t.Fatalf("add good: %v", err)
	}
	if err := manager.AddServer(ServerConfig{Name: "bad", Command: "true"}); err != nil {
		t.Fatalf("add bad: %v", err)
	}

	err := manager.Start(context.Background())
	if err == nil {
		t.Fatal("expected partial failure error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to start 1 server(s)") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "failed to connect server bad") {
		t.Errorf("expected bad server failure, got: %v", err)
	}
}

func TestCommandTransportProcessCleanup(t *testing.T) {
	t.Setenv("GOLEM_MCP_HELPER", "1")

	manager := NewManager()
	if err := manager.AddServer(ServerConfig{Name: "helper", Command: os.Args[0], Args: helperArgs()}); err != nil {
		t.Fatalf("add server: %v", err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := manager.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Double close must be idempotent.
	if err := manager.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}
