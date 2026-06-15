package session

import (
	"testing"

	"github.com/strings77wzq/golem/core/providers"
)

func TestSession_Fork_Basic(t *testing.T) {
	sess := NewSession("original")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "You are helpful."})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hello"})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "Hi there!"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "How are you?"})

	forked := sess.Fork(2, providers.Message{Role: providers.RoleUser, Content: "What's up?"})

	// Forked session should have: system + user1 + new message = 3
	if forked.MessageCount() != 3 {
		t.Fatalf("expected 3 messages in forked session, got %d", forked.MessageCount())
	}
	msgs := forked.GetMessages()
	if msgs[0].Content != "You are helpful." {
		t.Errorf("expected system prompt preserved, got %q", msgs[0].Content)
	}
	if msgs[1].Content != "Hello" {
		t.Errorf("expected first user message preserved, got %q", msgs[1].Content)
	}
	if msgs[2].Content != "What's up?" {
		t.Errorf("expected new message, got %q", msgs[2].Content)
	}
}

func TestSession_Fork_OriginalUnchanged(t *testing.T) {
	sess := NewSession("original")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "system"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "msg1"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "msg2"})

	_ = sess.Fork(1, providers.Message{Role: providers.RoleUser, Content: "forked"})

	// Original should still have 3 messages
	if sess.MessageCount() != 3 {
		t.Errorf("original session changed: expected 3, got %d", sess.MessageCount())
	}
}

func TestSession_Fork_NewID(t *testing.T) {
	sess := NewSession("original")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "test"})

	forked := sess.Fork(0)

	if forked.ID == sess.ID {
		t.Error("forked session should have different ID")
	}
}

func TestSession_Fork_Index0(t *testing.T) {
	sess := NewSession("original")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "system"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "hello"})

	forked := sess.Fork(0, providers.Message{Role: providers.RoleUser, Content: "new start"})

	// Only the new message (no system prompt since index 0 means nothing copied)
	if forked.MessageCount() != 1 {
		t.Fatalf("expected 1 message, got %d", forked.MessageCount())
	}
	if forked.GetMessages()[0].Content != "new start" {
		t.Errorf("expected 'new start', got %q", forked.GetMessages()[0].Content)
	}
}

func TestSession_Fork_IndexEqualLength(t *testing.T) {
	sess := NewSession("original")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "a"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "b"})

	forked := sess.Fork(2, providers.Message{Role: providers.RoleUser, Content: "c"})

	// All original + new message
	if forked.MessageCount() != 3 {
		t.Fatalf("expected 3 messages, got %d", forked.MessageCount())
	}
}

func TestSession_Fork_IndexBeyondLength(t *testing.T) {
	sess := NewSession("original")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "a"})

	forked := sess.Fork(100, providers.Message{Role: providers.RoleUser, Content: "b"})

	// Clamped to length + new message
	if forked.MessageCount() != 2 {
		t.Fatalf("expected 2 messages, got %d", forked.MessageCount())
	}
}

func TestSession_Fork_MultipleNewMessages(t *testing.T) {
	sess := NewSession("original")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "sys"})

	forked := sess.Fork(1,
		providers.Message{Role: providers.RoleUser, Content: "q1"},
		providers.Message{Role: providers.RoleAssistant, Content: "a1"},
	)

	// system + q1 + a1 = 3
	if forked.MessageCount() != 3 {
		t.Fatalf("expected 3 messages, got %d", forked.MessageCount())
	}
}

func TestSession_Fork_ToolCallsPreserved(t *testing.T) {
	sess := NewSession("original")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "sys"})
	sess.AddMessage(providers.Message{
		Role:    providers.RoleAssistant,
		Content: "",
		ToolCalls: []providers.ToolCall{
			{ID: "call-1", Name: "sql_query", Arguments: map[string]interface{}{"sql": "SELECT 1"}},
		},
	})
	sess.AddMessage(providers.Message{
		Role:       providers.RoleTool,
		Content:    "result",
		ToolCallID: "call-1",
	})

	forked := sess.Fork(2, providers.Message{Role: providers.RoleUser, Content: "new"})

	msgs := forked.GetMessages()
	if len(msgs) < 2 {
		t.Fatal("expected at least 2 messages")
	}
	// First message should be system, second should have tool calls
	if len(msgs[1].ToolCalls) == 0 {
		t.Error("expected tool calls preserved in forked session")
	}
}
