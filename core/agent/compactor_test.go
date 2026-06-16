package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
)

func TestCompactor_CompactsOldMessages(t *testing.T) {
	mock := providers.NewMockProvider("test")
	mock.AddResponse(&providers.LLMResponse{
		Content: "Summary: User asked about database schema, assistant explained tables.",
	})

	compactor := NewCompactor(mock, "test-model")

	store := session.NewMemoryStore()
	sess := session.NewSession("test")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "You are a database expert."})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "What tables are in the database?"})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "The database has users, orders, and products tables."})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Tell me about the users table."})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "The users table has columns: id, name, email, created_at."})
	store.Save(sess)

	// Use very low token budget to force compaction
	result, err := compactor.Compact(context.Background(), sess, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "compacted") {
		t.Errorf("result should mention 'compacted', got: %s", result)
	}

	// Verify session was modified
	messages := sess.GetMessages()
	if len(messages) < 3 {
		t.Errorf("expected at least 3 messages after compaction, got %d", len(messages))
	}

	// First message should still be system
	if messages[0].Role != providers.RoleSystem {
		t.Errorf("first message should be system, got %v", messages[0].Role)
	}
}

func TestCompactor_SkipsWhenUnderBudget(t *testing.T) {
	mock := providers.NewMockProvider("test")
	compactor := NewCompactor(mock, "test-model")

	sess := session.NewSession("test")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "You are helpful."})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hi"})

	result, err := compactor.Compact(context.Background(), sess, 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "already minimal") {
		t.Errorf("result should mention 'already minimal', got: %s", result)
	}

	// Verify session unchanged
	if sess.MessageCount() != 2 {
		t.Errorf("session should have 2 messages, got %d", sess.MessageCount())
	}

	// Verify no LLM call made
	if mock.CallCount != 0 {
		t.Errorf("LLM should not be called when under budget, got %d calls", mock.CallCount)
	}
}

func TestCompactor_PreservesRecentMessages(t *testing.T) {
	mock := providers.NewMockProvider("test")
	mock.AddResponse(&providers.LLMResponse{
		Content: "Summary of early conversation.",
	})

	compactor := NewCompactor(mock, "test-model")

	sess := session.NewSession("test")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "System prompt"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Message 1"})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "Response 1"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Message 2"})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "Response 2"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Message 3"})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "Response 3"})
	store := session.NewMemoryStore()
	store.Save(sess)

	// Use very low token budget to force compaction
	_, err := compactor.Compact(context.Background(), sess, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := sess.GetMessages()
	// Should have: system, summary, recent messages
	if len(messages) < 3 {
		t.Errorf("expected at least 3 messages, got %d", len(messages))
	}

	// Check that summary contains expected content
	foundSummary := false
	for _, msg := range messages {
		if strings.Contains(msg.Content, "Summary of early conversation") {
			foundSummary = true
			break
		}
	}
	if !foundSummary {
		t.Error("expected summary message in session")
	}
}

func TestCompactor_BuildsSummaryPrompt(t *testing.T) {
	mock := providers.NewMockProvider("test")
	mock.AddResponse(&providers.LLMResponse{Content: "summary"})

	compactor := NewCompactor(mock, "test-model")

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "What is SQL?"},
		{Role: providers.RoleAssistant, Content: "SQL is a query language."},
	}

	prompt := compactor.buildSummaryPrompt(messages)
	if !strings.Contains(prompt, "What is SQL?") {
		t.Errorf("prompt should contain original message, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Summarize") {
		t.Errorf("prompt should ask for summary, got: %s", prompt)
	}
}
