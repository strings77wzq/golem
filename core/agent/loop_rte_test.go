package agent

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
)

func TestRunToolExecution_ParallelAndResults(t *testing.T) {
	a, b, _, registry := setupTestAgent(t)
	defer b.Close()

	var callCount int32
	tool := &tools.MockTool{ToolName: "counter", ToolDescription: "counts"}
	tool.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		atomic.AddInt32(&callCount, 1)
		time.Sleep(50 * time.Millisecond)
		return &tools.ToolResult{ForLLM: "counted", ForUser: "user sees this"}, nil
	}
	registry.Register(tool)

	sess := session.NewSession("test-rte")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "test"})

	resp := &providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{ID: "c1", Name: "counter", Arguments: nil},
			{ID: "c2", Name: "counter", Arguments: nil},
			{ID: "c3", Name: "counter", Arguments: nil},
		},
	}

	msg := bus.InboundMessage{SessionID: "test-rte"}
	a.runToolExecution(context.Background(), resp, sess, msg, nil)

	if atomic.LoadInt32(&callCount) != 3 {
		t.Errorf("expected 3 tool calls, got %d", callCount)
	}

	// Session should have 3 tool result messages
	msgs := sess.GetMessages()
	toolMsgs := 0
	for _, m := range msgs {
		if m.Role == providers.RoleTool {
			toolMsgs++
		}
	}
	if toolMsgs != 3 {
		t.Errorf("expected 3 tool messages, got %d", toolMsgs)
	}
}

func TestRunToolExecution_EmitsResults(t *testing.T) {
	a, b, _, registry := setupTestAgent(t)
	defer b.Close()

	tool := &tools.MockTool{ToolName: "echo", ToolDescription: "echoes"}
	tool.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		return &tools.ToolResult{ForLLM: "llm", ForUser: "user"}, nil
	}
	registry.Register(tool)

	sess := session.NewSession("test-emit")
	resp := &providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{ID: "c1", Name: "echo", Arguments: nil},
		},
	}

	var emitted []bus.OutboundMessage
	emitFn := func(msg bus.OutboundMessage) {
		emitted = append(emitted, msg)
	}

	msg := bus.InboundMessage{SessionID: "test-emit"}
	a.runToolExecution(context.Background(), resp, sess, msg, emitFn)

	if len(emitted) != 1 {
		t.Fatalf("expected 1 emitted message, got %d", len(emitted))
	}
	if emitted[0].Content != "user" {
		t.Errorf("expected 'user', got %q", emitted[0].Content)
	}
}
