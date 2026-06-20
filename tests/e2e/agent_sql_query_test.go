package e2e

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/tests/e2e/helpers"
)

func TestAgent_SqlQueryHappyPath(t *testing.T) {
	helpers.SkipIfUnavailable(t, "http://127.0.0.1:11434", "qwen3:0.5b")

	binPath := helpers.BuildGolem(t)
	dbPath := helpers.SeedDemoDB(t)

	// Create a temporary config with Ollama model
	configPath := writeOllamaConfig(t)

	// Capture transcript
	transcript, err := helpers.New(filepath.Join(t.TempDir(), "transcripts"), "agent_sql_query")
	if err != nil {
		t.Fatalf("create transcript: %v", err)
	}
	defer transcript.Close()

	cmd := exec.Command(binPath, "agent",
		"-c", configPath,
		"-M", "ollama/qwen3:0.5b",
		"--db", dbPath,
		"--json-events",
		"-m", "How many rows are in the users table?",
	)
	cmd.Dir = filepath.Join(helpers.RepoRoot(t))

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("golem agent failed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Write stderr (json events) to transcript
	transcript.Write([]byte(stderr.String()))

	// Parse JSON events from stderr
	events := parseJSONEvents(t, stderr.String())

	// Assert: at least one tool_call event with sql_query
	hasSQLQueryCall := false
	hasSQLQueryResult := false
	for _, evt := range events {
		if evt.Type == "tool_call" && evt.ToolName == "sql_query" {
			hasSQLQueryCall = true
		}
		if evt.Type == "tool_result" && evt.ToolName == "sql_query" {
			hasSQLQueryResult = true
			// Assert result contains row data
			if !strings.Contains(evt.ToolOutput, "5") && !strings.Contains(evt.ToolOutput, "users") {
				t.Errorf("expected tool_result to mention rows or table, got: %s", evt.ToolOutput)
			}
		}
	}

	if !hasSQLQueryCall {
		t.Error("expected at least one tool_call event with tool_name=sql_query")
	}
	if !hasSQLQueryResult {
		t.Error("expected at least one tool_result event with tool_name=sql_query")
	}
}
