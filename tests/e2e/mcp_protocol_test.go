package e2e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/tests/e2e/helpers"
)

// startMCPServer launches golem mcp-server and returns the stdin pipe and a
// stdout scanner. The caller owns stdin.Close() and cmd.Wait().
func startMCPServer(t *testing.T, dbPath string) (io.WriteCloser, *bufio.Scanner, *exec.Cmd) {
	t.Helper()

	binPath := helpers.BuildGolem(t)
	cmd := exec.Command(binPath, "mcp-server", "--db", dbPath)
	cmd.Dir = filepath.Join(helpers.RepoRoot(t))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start mcp-server: %v", err)
	}

	return stdin, bufio.NewScanner(stdout), cmd
}

// mcpRequest sends a JSON-RPC request and reads exactly one response line.
func mcpRequest(t *testing.T, stdin io.WriteCloser, scanner *bufio.Scanner, id int, method string, params string) map[string]interface{} {
	t.Helper()

	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":%q`, id, method)
	if params != "" {
		req += `,"params":` + params
	}
	req += "}"

	if _, err := stdin.Write([]byte(req + "\n")); err != nil {
		t.Fatalf("write %s: %v", method, err)
	}
	if !scanner.Scan() {
		t.Fatalf("no response for %s", method)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(scanner.Text()), &resp); err != nil {
		t.Fatalf("parse %s response: %v\nraw: %s", method, err, scanner.Text())
	}
	return resp
}

// mcpInitialize performs the initialize handshake required by the official
// go-sdk server (legacy 2025-11-25 protocol, still supported).
func mcpInitialize(t *testing.T, stdin io.WriteCloser, scanner *bufio.Scanner) {
	t.Helper()

	resp := mcpRequest(t, stdin, scanner, 1, "initialize",
		`{"protocolVersion":"2025-11-25","clientInfo":{"name":"e2e","version":"1.0"},"capabilities":{}}`)
	if errMsg, hasErr := resp["error"]; hasErr {
		t.Fatalf("initialize failed: %v", errMsg)
	}
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("initialize: missing result: %v", resp)
	}
	if proto, _ := result["protocolVersion"].(string); proto != "2025-11-25" {
		t.Fatalf("initialize: unexpected protocol version %v", result["protocolVersion"])
	}
}

func TestMcp_ToolsListIncludesSqlQuery(t *testing.T) {
	dbPath := helpers.SeedDemoDB(t)

	stdin, scanner, cmd := startMCPServer(t, dbPath)
	defer func() {
		stdin.Close()
		cmd.Wait()
	}()

	mcpInitialize(t, stdin, scanner)

	resp := mcpRequest(t, stdin, scanner, 2, "tools/list", "")
	if errMsg, hasErr := resp["error"]; hasErr {
		t.Fatalf("MCP error: %v", errMsg)
	}

	result, _ := resp["result"].(map[string]interface{})
	toolsRaw, _ := result["tools"].([]interface{})
	var toolNames []string
	for _, toolRaw := range toolsRaw {
		tool, _ := toolRaw.(map[string]interface{})
		if name, _ := tool["name"].(string); name != "" {
			toolNames = append(toolNames, name)
		}
	}

	found := false
	for _, name := range toolNames {
		if name == "sql_query" {
			found = true
		}
	}
	if !found {
		t.Errorf("sql_query not in tool list: %v", toolNames)
	}
}

func TestMcp_ToolsCallSqlQuery(t *testing.T) {
	dbPath := helpers.SeedDemoDB(t)

	stdin, scanner, cmd := startMCPServer(t, dbPath)
	defer func() {
		stdin.Close()
		cmd.Wait()
	}()

	mcpInitialize(t, stdin, scanner)

	resp := mcpRequest(t, stdin, scanner, 2, "tools/call",
		`{"name":"sql_query","arguments":{"sql":"SELECT COUNT(*) as cnt FROM users"}}`)
	if errMsg, hasErr := resp["error"]; hasErr {
		t.Fatalf("MCP error: %v", errMsg)
	}

	result, _ := resp["result"].(map[string]interface{})
	contentRaw, _ := result["content"].([]interface{})
	if len(contentRaw) == 0 {
		t.Fatal("expected non-empty content array")
	}

	block, _ := contentRaw[0].(map[string]interface{})
	text, _ := block["text"].(string)
	if !strings.Contains(text, "5") && !strings.Contains(text, "cnt") {
		t.Errorf("expected content to mention count, got: %s", text)
	}
}
