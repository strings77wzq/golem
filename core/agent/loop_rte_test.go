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

func TestReactLoop_ReturnsFinalResponse(t *testing.T) {
	a, b, mockProvider, _ := setupTestAgent(t)
	defer b.Close()

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Hello from reactLoop",
		Usage:   providers.TokenUsage{TotalTokens: 100},
	})

	sess := session.NewSession("test-react")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "hi"})

	toolDefs := a.toolRegistry.ListDefinitions()
	msg := bus.InboundMessage{SessionID: "test-react"}

	content, usage, err := a.reactLoop(context.Background(), msg, sess, "mock/test-model", toolDefs, false, nil, nil)
	if err != nil {
		t.Fatalf("reactLoop failed: %v", err)
	}
	if content != "Hello from reactLoop" {
		t.Errorf("expected 'Hello from reactLoop', got %q", content)
	}
	if usage == nil || usage.TotalTokens != 100 {
		t.Errorf("expected usage with 100 tokens, got %v", usage)
	}

	// Session should have user + assistant messages
	msgs := sess.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[1].Role != providers.RoleAssistant {
		t.Errorf("expected assistant message, got %v", msgs[1].Role)
	}
}

func TestReactLoop_ToolCallContinuesLoop(t *testing.T) {
	a, b, mockProvider, registry := setupTestAgent(t)
	defer b.Close()

	// Register echo tool
	echoTool := &tools.MockTool{ToolName: "echo", ToolDescription: "echoes"}
	echoTool.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		return &tools.ToolResult{ForLLM: "echoed"}, nil
	}
	registry.Register(echoTool)

	// First call: tool call. Second call: final response.
	mockProvider.AddResponse(&providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "echo", Arguments: map[string]interface{}{"text": "hello"}},
		},
	})
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Done after tool",
		Usage:   providers.TokenUsage{TotalTokens: 200},
	})

	sess := session.NewSession("test-react-loop")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "use echo"})

	toolDefs := a.toolRegistry.ListDefinitions()
	msg := bus.InboundMessage{SessionID: "test-react-loop"}

	content, _, err := a.reactLoop(context.Background(), msg, sess, "mock/test-model", toolDefs, false, nil, nil)
	if err != nil {
		t.Fatalf("reactLoop failed: %v", err)
	}
	if content != "Done after tool" {
		t.Errorf("expected 'Done after tool', got %q", content)
	}

	// Session should have: user, assistant+toolcall, toolresult, assistant
	msgs := sess.GetMessages()
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}
}

func TestReactLoop_MaxIterationsReached(t *testing.T) {
	a, b, mockProvider, registry := setupTestAgent(t)
	defer b.Close()

	// Register echo tool
	echoTool := &tools.MockTool{ToolName: "echo", ToolDescription: "echoes"}
	echoTool.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		return &tools.ToolResult{ForLLM: "echoed"}, nil
	}
	registry.Register(echoTool)

	// Always return tool calls — will hit max iterations (2 iterations needed)
	mockProvider.AddResponse(&providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{ID: "call_1", Name: "echo", Arguments: nil},
		},
	})
	mockProvider.AddResponse(&providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{ID: "call_2", Name: "echo", Arguments: nil},
		},
	})

	sess := session.NewSession("test-max-iter")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "loop forever"})

	a.maxToolIterations = 2 // Low limit for testing
	toolDefs := a.toolRegistry.ListDefinitions()
	msg := bus.InboundMessage{SessionID: "test-max-iter"}

	content, _, err := a.reactLoop(context.Background(), msg, sess, "mock/test-model", toolDefs, false, nil, nil)
	if err != nil {
		t.Fatalf("reactLoop failed: %v", err)
	}
	if content != "max tool iterations reached" {
		t.Errorf("expected 'max tool iterations reached', got %q", content)
	}
}
