package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/strings77wzq/golem/core/tools"
)

// Regression tests for security-review H1/H2 and code-review findings.

func TestDelegateExecuteNonexistentCommand(t *testing.T) {
	// Regression for H2: cmd.Process.Kill() on a command whose Start never
	// succeeded must not panic.
	tool := NewDelegateTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command":   "/nonexistent/binary-xyz",
		"tool":      "ping",
		"arguments": map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for nonexistent command")
	}
}

func TestDelegateExecuteTimeoutParam(t *testing.T) {
	// The advertised timeout parameter must be honored (capped at default).
	start := time.Now()
	tool := NewDelegateTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command":   "sleep",
		"args":      []interface{}{"10"},
		"tool":      "ping",
		"arguments": map[string]interface{}{},
		"timeout":   float64(1),
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !result.IsError {
		t.Error("expected timeout error")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("timeout param ignored: took %v", elapsed)
	}
}

func TestSanitizeToolName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"ping", "ping"},
		{"my-tool_2", "my-tool_2"},
		{"bad name!@#", "badname"},
		{"中文工具", "unnamed"},
		{"", "unnamed"},
		{strings.Repeat("a", 100), strings.Repeat("a", maxToolNameLen)},
	}
	for _, tt := range tests {
		if got := sanitizeToolName(tt.in); got != tt.want {
			t.Errorf("sanitizeToolName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestServerRejectsOversizedArguments(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&testGolemTool{
		name: "echo",
		exec: func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
			return &tools.ToolResult{ForLLM: "ok"}, nil
		},
	})

	srv := NewServer(registry)
	session := connectServerClient(t, srv)

	big := strings.Repeat("x", maxCallArgumentsBytes+1)
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "echo",
		Arguments: json.RawMessage(`{"data":"` + big + `"}`),
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for oversized arguments")
	}
}

func TestServerTruncatesOversizedOutput(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&testGolemTool{
		name: "big",
		exec: func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
			return &tools.ToolResult{ForLLM: strings.Repeat("y", maxServerOutputBytes+1000)}, nil
		},
	})

	srv := NewServer(registry)
	session := connectServerClient(t, srv)

	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "big"})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	tc := result.Content[0].(*sdk.TextContent)
	// Truncation marker is "\n...[truncated]" (15 bytes).
	if len(tc.Text) > maxServerOutputBytes+len("\n...[truncated]") {
		t.Errorf("output not truncated: %d bytes", len(tc.Text))
	}
	if !strings.HasSuffix(tc.Text, "...[truncated]") {
		t.Errorf("missing truncation marker: %q", tc.Text[len(tc.Text)-20:])
	}
}

func TestServerHandlesNilToolResult(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&testGolemTool{
		name: "nilresult",
		exec: func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
			return nil, nil
		},
	})

	srv := NewServer(registry)
	session := connectServerClient(t, srv)

	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "nilresult"})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for nil result")
	}
}

func TestProxyExecuteReturnsErrorOnly(t *testing.T) {
	// Regression for review L2: on remote failure the proxy returns (nil, err)
	// so the agent loop formats the message instead of dropping it.
	manager := NewManager()
	proxy := MCPToolProxy{serverName: "ghost", mcpTool: MCPTool{Name: "tool"}, manager: manager}

	result, err := proxy.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}
}
