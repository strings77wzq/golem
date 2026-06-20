package agent

import (
	"context"
	"testing"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
)

func TestInitSession_CreatesNewSession(t *testing.T) {
	a, b, _, _ := setupTestAgent(t)
	defer b.Close()

	a.systemPrompt = "You are a test agent."

	msg := bus.InboundMessage{
		SessionID: "new-session",
		Content:   "Hello",
		Role:      bus.RoleUser,
	}

	sess, err := a.initSession(context.Background(), msg)
	if err != nil {
		t.Fatalf("initSession failed: %v", err)
	}

	// Session should have 2 messages: system prompt + user message
	msgs := sess.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != providers.RoleSystem {
		t.Errorf("expected first message role system, got %v", msgs[0].Role)
	}
	if msgs[0].Content != "You are a test agent." {
		t.Errorf("expected system prompt, got %q", msgs[0].Content)
	}
	if msgs[1].Role != providers.RoleUser {
		t.Errorf("expected second message role user, got %v", msgs[1].Role)
	}
	if msgs[1].Content != "Hello" {
		t.Errorf("expected user message 'Hello', got %q", msgs[1].Content)
	}
}

func TestInitSession_ReusesExistingSession(t *testing.T) {
	a, b, _, _ := setupTestAgent(t)
	defer b.Close()

	a.systemPrompt = "You are a test agent."

	// Pre-create session with existing messages
	store := a.sessionStore.(*session.MemoryStore)
	existing := session.NewSession("existing-session")
	existing.AddMessage(providers.Message{
		Role:    providers.RoleUser,
		Content: "previous message",
	})
	store.Save(existing)

	msg := bus.InboundMessage{
		SessionID: "existing-session",
		Content:   "new message",
		Role:      bus.RoleUser,
	}

	sess, err := a.initSession(context.Background(), msg)
	if err != nil {
		t.Fatalf("initSession failed: %v", err)
	}

	// Session should have 2 messages: existing + new user message
	// System prompt is NOT injected because session is not empty
	msgs := sess.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "previous message" {
		t.Errorf("expected existing message preserved, got %q", msgs[0].Content)
	}
	if msgs[1].Content != "new message" {
		t.Errorf("expected new user message, got %q", msgs[1].Content)
	}
}

func TestInitSession_EmptyPromptSkipsSystemMessage(t *testing.T) {
	a, b, _, _ := setupTestAgent(t)
	defer b.Close()

	a.systemPrompt = "" // No system prompt

	msg := bus.InboundMessage{
		SessionID: "no-prompt-session",
		Content:   "Hello",
		Role:      bus.RoleUser,
	}

	sess, err := a.initSession(context.Background(), msg)
	if err != nil {
		t.Fatalf("initSession failed: %v", err)
	}

	// Session should have only 1 message: user message (no system prompt)
	msgs := sess.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != providers.RoleUser {
		t.Errorf("expected user message, got %v", msgs[0].Role)
	}
}
