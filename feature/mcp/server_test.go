package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/tools"
)

type mockTool struct {
	name   string
	result string
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return "mock tool: " + m.name }
func (m *mockTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{{Name: "input", Type: "string", Description: "input", Required: true}}
}
func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	return &tools.ToolResult{ForLLM: m.result, ForUser: m.result}, nil
}

func writeRequest(t *testing.T, input *bytes.Buffer, req interface{}) {
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	input.Write(data)
}

func readResponse(t *testing.T, output *bytes.Buffer) map[string]interface{} {
	line := make([]byte, 4096)
	n, _ := output.Read(line)
	if n == 0 {
		t.Fatal("no response")
	}
	var resp map[string]interface{}
	json.Unmarshal(line[:n], &resp)
	return resp
}

func TestServerInitialize(t *testing.T) {
	var output bytes.Buffer
	input := bytes.NewBuffer(nil)
	reg := tools.NewRegistry()
	server := NewServer(input, &output, reg)

	writeRequest(t, input, map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]interface{}{"protocolVersion": "2024-11-05", "clientInfo": map[string]string{"name": "test", "version": "1.0"}},
	})
	server.handleMessage(context.Background())
	resp := readResponse(t, &output)
	if resp["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v", resp["jsonrpc"])
	}
}

func TestServerToolsList(t *testing.T) {
	var output bytes.Buffer
	input := bytes.NewBuffer(nil)
	reg := tools.NewRegistry()
	reg.Register(&mockTool{name: "sql_query", result: "ok"})
	reg.Register(&mockTool{name: "sql_schema", result: "ok"})
	server := NewServer(input, &output, reg)

	writeRequest(t, input, map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]interface{}{"protocolVersion": "2024-11-05", "clientInfo": map[string]string{"name": "test", "version": "1.0"}},
	})
	server.handleMessage(context.Background())
	readResponse(t, &output)

	writeRequest(t, input, map[string]interface{}{"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
	server.handleMessage(context.Background())
	resp := readResponse(t, &output)
	result := resp["result"].(map[string]interface{})
	toolList := result["tools"].([]interface{})
	if len(toolList) != 2 {
		t.Errorf("expected 2 tools, got %d", len(toolList))
	}
}

func TestServerToolsCall(t *testing.T) {
	var output bytes.Buffer
	input := bytes.NewBuffer(nil)
	reg := tools.NewRegistry()
	reg.Register(&mockTool{name: "sql_query", result: "| id | name |\n| 1 | Alice |"})
	server := NewServer(input, &output, reg)

	writeRequest(t, input, map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]interface{}{"protocolVersion": "2024-11-05", "clientInfo": map[string]string{"name": "test", "version": "1.0"}},
	})
	server.handleMessage(context.Background())
	readResponse(t, &output)

	writeRequest(t, input, map[string]interface{}{
		"jsonrpc": "2.0", "id": 3, "method": "tools/call",
		"params": map[string]interface{}{"name": "sql_query", "arguments": map[string]interface{}{"sql": "SELECT * FROM users"}},
	})
	server.handleMessage(context.Background())
	resp := readResponse(t, &output)
	result := resp["result"].(map[string]interface{})
	text := result["content"].([]interface{})[0].(map[string]interface{})["text"].(string)
	if !strings.Contains(text, "Alice") {
		t.Errorf("expected Alice in result, got %q", text)
	}
}

func TestServerUnknownMethod(t *testing.T) {
	var output bytes.Buffer
	input := bytes.NewBuffer(nil)
	reg := tools.NewRegistry()
	server := NewServer(input, &output, reg)

	writeRequest(t, input, map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": "unknown/method"})
	server.handleMessage(context.Background())
	resp := readResponse(t, &output)
	errObj := resp["error"].(map[string]interface{})
	if errObj["code"] != float64(-32601) {
		t.Errorf("expected method not found error, got %v", errObj)
	}
}
