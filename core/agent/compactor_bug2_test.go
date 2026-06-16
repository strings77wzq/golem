package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
)

// Bug: When LLM fails, simpleSummary ignores tool calls entirely.
// The conversation has user -> assistant(tool_calls) -> tool_result -> user,
// but simpleSummary only shows user and assistant content, losing tool context.
func TestCompactor_SimpleSummaryIncludesToolCalls(t *testing.T) {
	mock := providers.NewMockProvider("test")
	// Force LLM failure by not adding any response
	compactor := NewCompactor(mock, "test-model")

	sess := session.NewSession("test")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "System"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Query the database"})
	sess.AddMessage(providers.Message{
		Role:      providers.RoleAssistant,
		Content:   "",
		ToolCalls: []providers.ToolCall{{ID: "call_1", Name: "sql_query", Arguments: map[string]interface{}{"query": "SELECT * FROM users"}}},
	})
	sess.AddMessage(providers.Message{Role: providers.RoleTool, Content: "id,name,email\n1,alice,alice@example.com"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "How many users?"})

	// Force compaction with very low budget
	result, err := compactor.Compact(context.Background(), sess, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "compacted") {
		t.Errorf("expected 'compacted', got: %s", result)
	}

	// Check that the summary preserves tool information
	messages := sess.GetMessages()
	foundSummary := false
	for _, msg := range messages {
		if strings.Contains(msg.Content, "sql_query") || strings.Contains(msg.Content, "Tool:") {
			foundSummary = true
			break
		}
	}
	if !foundSummary {
		t.Error("summary should mention tool calls, but didn't")
	}
}
