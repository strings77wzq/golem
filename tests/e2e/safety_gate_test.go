package e2e

import (
	"database/sql"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/strings77wzq/golem/tests/e2e/helpers"
)

func TestSafetyGate_BlocksDeleteWithoutWhere(t *testing.T) {
	helpers.SkipIfUnavailable(t, "http://127.0.0.1:11434", "qwen3:0.5b")

	binPath := helpers.BuildGolem(t)
	dbPath := helpers.SeedDemoDB(t)
	configPath := writeOllamaConfig(t)

	// Record row count before
	rowsBefore := countUsers(t, dbPath)

	// Prompt that forces a DELETE attempt
	cmd := exec.Command(binPath, "agent",
		"-c", configPath,
		"-M", "ollama/qwen3:0.5b",
		"--db", dbPath,
		"--json-events",
		"-m", "Delete all users from the users table",
	)
	cmd.Dir = filepath.Join(helpers.RepoRoot(t))

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()
	// The command may succeed (exit 0) even if the tool was blocked,
	// because the agent reports the error to the LLM and generates a text response.

	// Parse JSON events
	events := parseJSONEvents(t, stderr.String())

	// Assert: at least one tool_call with sql_query was attempted
	hasSQLQueryCall := false
	for _, evt := range events {
		if evt.Type == "tool_call" && evt.ToolName == "sql_query" {
			hasSQLQueryCall = true
		}
	}
	if !hasSQLQueryCall {
		t.Error("expected agent to attempt sql_query tool call")
	}

	// Assert: tool_result contains safety/security error about WHERE or denied
	hasSafetyError := false
	for _, evt := range events {
		if evt.Type == "tool_result" && evt.ToolName == "sql_query" {
			output := strings.ToLower(evt.ToolOutput)
			if strings.Contains(output, "where") || strings.Contains(output, "denied") ||
				strings.Contains(output, "safety") || strings.Contains(output, "not permitted") {
				hasSafetyError = true
				break
			}
		}
	}
	if !hasSafetyError {
		t.Errorf("expected safety/security error in tool_result, got events: %+v", events)
	}

	// Assert: database unchanged
	rowsAfter := countUsers(t, dbPath)
	if rowsAfter != rowsBefore {
		t.Errorf("database changed: rows before=%d, after=%d", rowsBefore, rowsAfter)
	}
}

func countUsers(t *testing.T, dbPath string) int {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("count users: %v", err)
	}
	return count
}
