package mcp

import (
	"context"
	"errors"
	"fmt"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/strings77wzq/golem/core/tools"
)

// testGolemTool is a configurable tools.Tool used to exercise the SDK adapter.
type testGolemTool struct {
	name   string
	desc   string
	params []tools.ToolParameter
	exec   func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error)
}

func (t *testGolemTool) Name() string        { return t.name }
func (t *testGolemTool) Description() string { return t.desc }
func (t *testGolemTool) Parameters() []tools.ToolParameter {
	return t.params
}
func (t *testGolemTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	return t.exec(ctx, args)
}

func TestInputSchemaFromParameters(t *testing.T) {
	params := []tools.ToolParameter{
		{Name: "query", Type: "string", Description: "Search query", Required: true},
		{Name: "limit", Type: "number", Description: "Result limit", Required: false},
	}

	schema := inputSchemaFromParameters(params)
	if schema["type"] != "object" {
		t.Errorf("expected type object, got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map, got %T", schema["properties"])
	}
	if len(props) != 2 {
		t.Errorf("expected 2 properties, got %d", len(props))
	}

	q, ok := props["query"].(map[string]any)
	if !ok || q["type"] != "string" || q["description"] != "Search query" {
		t.Errorf("query property malformed: %v", props["query"])
	}

	required, ok := schema["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "query" {
		t.Errorf("expected required=[query], got %v", schema["required"])
	}
}

func TestInputSchemaFromParametersEmpty(t *testing.T) {
	schema := inputSchemaFromParameters(nil)
	props, ok := schema["properties"].(map[string]any)
	if !ok || len(props) != 0 {
		t.Errorf("expected empty properties, got %v", schema["properties"])
	}
	required := schema["required"].([]string)
	if len(required) != 0 {
		t.Errorf("expected empty required, got %v", required)
	}
}

// connectServerClient wires an in-memory server↔client pair.
func connectServerClient(t *testing.T, srv *Server) *sdk.ClientSession {
	t.Helper()

	clientTransport, serverTransport := sdk.NewInMemoryTransports()

	srvCtx, srvCancel := context.WithCancel(context.Background())
	t.Cleanup(srvCancel)
	if _, err := srv.Connect(srvCtx, serverTransport); err != nil {
		t.Fatalf("server connect failed: %v", err)
	}

	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	t.Cleanup(func() { session.Close() }) //nolint:errcheck

	return session
}

func TestServerExposesRegistryTools(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&testGolemTool{
		name: "echo",
		desc: "Echo back the input",
		params: []tools.ToolParameter{
			{Name: "message", Type: "string", Description: "Message to echo", Required: true},
		},
		exec: func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
			return &tools.ToolResult{ForLLM: args["message"].(string), ForUser: args["message"].(string)}, nil
		},
	})

	srv := NewServer(registry)
	session := connectServerClient(t, srv)

	result, err := session.ListTools(context.Background(), &sdk.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}
	if len(result.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result.Tools))
	}
	tool := result.Tools[0]
	if tool.Name != "echo" {
		t.Errorf("expected tool echo, got %s", tool.Name)
	}
	if tool.Description != "Echo back the input" {
		t.Errorf("unexpected description %q", tool.Description)
	}
	if tool.InputSchema == nil {
		t.Error("expected non-nil input schema")
	}
}

func TestServerRoutesToolCall(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&testGolemTool{
		name: "add",
		desc: "Add two numbers",
		params: []tools.ToolParameter{
			{Name: "a", Type: "number", Description: "First number", Required: true},
			{Name: "b", Type: "number", Description: "Second number", Required: true},
		},
		exec: func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
			a := int(args["a"].(float64))
			b := int(args["b"].(float64))
			text := fmt.Sprintf("%d", a+b)
			return &tools.ToolResult{ForLLM: text, ForUser: text}, nil
		},
	})

	srv := NewServer(registry)
	session := connectServerClient(t, srv)

	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "add",
		Arguments: map[string]any{"a": float64(2), "b": float64(3)},
	})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if result.IsError {
		t.Error("expected no error")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}
	tc, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if tc.Text != "5" {
		t.Errorf("expected text 5, got %q", tc.Text)
	}
}

func TestServerMapsToolErrorToIsError(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&testGolemTool{
		name: "failing",
		desc: "Always fails",
		exec: func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
			return &tools.ToolResult{ForLLM: "denied by policy", ForUser: "", IsError: true}, nil
		},
	})

	srv := NewServer(registry)
	session := connectServerClient(t, srv)

	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "failing"})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for failing tool")
	}
}

func TestServerMapsExecutionErrorToIsError(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&testGolemTool{
		name: "crashing",
		desc: "Returns a Go error",
		exec: func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
			return nil, errors.New("boom")
		},
	})

	srv := NewServer(registry)
	session := connectServerClient(t, srv)

	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "crashing"})
	if err != nil {
		t.Fatalf("call tool failed: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for execution error")
	}
	if len(result.Content) == 0 {
		t.Fatal("expected error content")
	}
}
