package tui

import (
	"testing"
)

// Bug: /new command clears messages but doesn't reset thinking state.
// If user sends /new while agent is thinking, the thinking animation persists.
func TestNewSessionCommand_ResetsThinkingState(t *testing.T) {
	m := &Model{
		thinking:   true,
		lastError:  "some error",
		messages:   []chatMsg{{role: roleAssistant, content: "old message"}},
		sessionID:  "old-session",
		input:      "old input",
	}

	cmd := &newSessionCmd{}
	cmd.Execute(m)

	if m.thinking {
		t.Error("thinking state should be reset after /new")
	}
	if m.lastError != "" {
		t.Error("lastError should be cleared after /new")
	}
	if len(m.messages) != 1 {
		t.Errorf("expected 1 message (welcome), got %d", len(m.messages))
	}
	if m.sessionID != "" {
		t.Error("sessionID should be cleared after /new")
	}
	if m.input != "" {
		t.Error("input should be cleared after /new")
	}
}
