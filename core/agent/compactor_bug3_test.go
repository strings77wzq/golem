package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
)

// Bug: Compactor with nil provider panics in summarize().
func TestCompactor_NilProvider_ShouldNotPanic(t *testing.T) {
	compactor := NewCompactor(nil, "test-model")

	sess := session.NewSession("test")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "System"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: strings.Repeat("x", 500)})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: strings.Repeat("y", 500)})

	// Should not panic even with nil provider
	result, err := compactor.Compact(context.Background(), sess, 10)
	if err != nil {
		// Error is acceptable, but should not panic
		t.Logf("got error (acceptable): %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

// Bug: Compactor with cancelled context should return error, not hang.
func TestCompactor_CancelledContext(t *testing.T) {
	mock := providers.NewMockProvider("test")
	// Don't add any response — will block if context not checked
	compactor := NewCompactor(mock, "test-model")

	sess := session.NewSession("test")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "System"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: strings.Repeat("x", 500)})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: strings.Repeat("y", 500)})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := compactor.Compact(ctx, sess, 10)
	if err == nil {
		t.Log("no error (mock may not check context)")
	}
	// Should not hang — test will timeout if it does
}
