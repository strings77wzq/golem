package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
)

func TestAutoCompact_TriggersAtThreshold(t *testing.T) {
	mock := providers.NewMockProvider("test")
	mock.AddResponse(&providers.LLMResponse{
		Content: "Summary of conversation.",
	})

	compactor := NewCompactor(mock, "test-model")
	sess := session.NewSession("test")

	// Add enough messages to exceed 80% of 100 token budget
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: strings.Repeat("x", 300)})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: strings.Repeat("y", 300)})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: strings.Repeat("z", 300)})

	result, err := compactor.Compact(context.Background(), sess, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "compacted") {
		t.Errorf("expected compaction, got: %s", result)
	}
}

func TestAutoCompact_SkipsUnderThreshold(t *testing.T) {
	mock := providers.NewMockProvider("test")
	compactor := NewCompactor(mock, "test-model")
	sess := session.NewSession("test")

	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "System"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hello"})

	// 1000 token budget, only ~20 tokens used
	result, err := compactor.Compact(context.Background(), sess, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "already minimal") {
		t.Errorf("expected 'already minimal', got: %s", result)
	}
}

func TestAutoCompact_PreservesSystemPrompt(t *testing.T) {
	mock := providers.NewMockProvider("test")
	mock.AddResponse(&providers.LLMResponse{
		Content: "Summary.",
	})

	compactor := NewCompactor(mock, "test-model")
	sess := session.NewSession("test")

	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "You are a database expert."})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: strings.Repeat("query ", 100)})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: strings.Repeat("answer ", 100)})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: strings.Repeat("followup ", 100)})

	_, err := compactor.Compact(context.Background(), sess, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := sess.GetMessages()
	if len(messages) == 0 {
		t.Fatal("session should not be empty after compaction")
	}

	// First message should still be system prompt
	if messages[0].Role != providers.RoleSystem {
		t.Errorf("first message should be system, got %v", messages[0].Role)
	}
	if !strings.Contains(messages[0].Content, "database expert") {
		t.Errorf("system prompt should be preserved, got: %s", messages[0].Content)
	}
}
