package agent

import (
	"context"
	"testing"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
)

// Bug: Compactor panics when session has exactly 2 messages (system + 1 user).
// The old code did: old = rest[:len(rest)-keepRecent] where keepRecent=4
// When len(rest)=1, this becomes rest[:1-4] = rest[:-3] which panics.
func TestCompactor_NoPanicOnMinimalSession(t *testing.T) {
	mock := providers.NewMockProvider("test")
	compactor := NewCompactor(mock, "test-model")

	sess := session.NewSession("test")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "System"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hello"})

	// Should not panic
	result, err := compactor.Compact(context.Background(), sess, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "session is already minimal, nothing to compact" {
		t.Errorf("expected 'already minimal', got: %s", result)
	}
}

// Bug: Compactor panics when session has only user messages (no system prompt).
func TestCompactor_NoPanicOnUserOnlySession(t *testing.T) {
	mock := providers.NewMockProvider("test")
	compactor := NewCompactor(mock, "test-model")

	sess := session.NewSession("test")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hello"})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "Hi"})

	result, err := compactor.Compact(context.Background(), sess, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "session is already minimal, nothing to compact" {
		t.Errorf("expected 'already minimal', got: %s", result)
	}
}
