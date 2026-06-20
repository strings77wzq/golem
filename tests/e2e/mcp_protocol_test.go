package e2e

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/tests/e2e/helpers"
)

func TestMcp_ToolsListIncludesSqlQuery(t *testing.T) {
	binPath := helpers.BuildGolem(t)
	dbPath := helpers.SeedDemoDB(t)

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
	defer func() {
		stdin.Close()
		cmd.Wait()
	}()

	// Send tools/list request
	req := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	if _, err := stdin.Write([]byte(req + "\n")); err != nil {
		t.Fatalf("write tools/list: %v", err)
	}

	// Read response
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		t.Fatal("no response from mcp-server")
	}
	line := scanner.Text()

	var resp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("parse response: %v\nraw: %s", err, line)
	}
	if resp.Error != nil {
		t.Fatalf("MCP error: %d %s", resp.Error.Code, resp.Error.Message)
	}

	// Assert sql_query is in tool list
	found := false
	var toolNames []string
	for _, tool := range resp.Result.Tools {
		toolNames = append(toolNames, tool.Name)
		if tool.Name == "sql_query" {
			found = true
		}
	}
	if !found {
		t.Errorf("sql_query not in tool list: %v", toolNames)
	}
}

func TestMcp_ToolsCallSqlQuery(t *testing.T) {
	binPath := helpers.BuildGolem(t)
	dbPath := helpers.SeedDemoDB(t)

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
	defer func() {
		stdin.Close()
		cmd.Wait()
	}()

	// Send tools/call for sql_query
	req := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"sql_query","arguments":{"sql":"SELECT COUNT(*) as cnt FROM users"}}}`
	if _, err := stdin.Write([]byte(req + "\n")); err != nil {
		t.Fatalf("write tools/call: %v", err)
	}

	// Read response
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		t.Fatal("no response from mcp-server")
	}
	line := scanner.Text()

	var resp struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("parse response: %v\nraw: %s", err, line)
	}
	if resp.Error != nil {
		t.Fatalf("MCP error: %d %s", resp.Error.Code, resp.Error.Message)
	}

	// Assert non-empty content array
	if len(resp.Result.Content) == 0 {
		t.Fatal("expected non-empty content array")
	}

	// Assert content contains row data (5 users)
	text := resp.Result.Content[0].Text
	if !strings.Contains(text, "5") && !strings.Contains(text, "cnt") {
		t.Errorf("expected content to mention count, got: %s", text)
	}
}
