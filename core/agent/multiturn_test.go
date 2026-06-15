package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/foundation/logger"
)

type echoTool struct{}

func (e *echoTool) Name() string        { return "echo" }
func (e *echoTool) Description() string { return "Echoes input" }
func (e *echoTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{{Name: "input", Type: "string", Description: "Input to echo", Required: true}}
}
func (e *echoTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	input, _ := args["input"].(string)
	return &tools.ToolResult{ForLLM: "echoed: " + input, ForUser: input}, nil
}

func setupMultiTurnAgent(t *testing.T) (*Agent, bus.Bus, *providers.MockProvider, *tools.Registry) {
	t.Helper()
	registry := tools.NewRegistry()
	registry.Register(&echoTool{})

	mockProvider := providers.NewMockProvider("mock")
	factory := providers.NewFactory()
	factory.Register("mock", mockProvider)

	cfg := &config.Config{
		Agents: config.AgentConfig{
			Defaults: config.AgentDefaults{
				ModelName:  "mock/echo",
				MaxTokens:  4096,
				SystemPrompt: "You are a helpful assistant.",
			},
		},
	}

	b := bus.New()
	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.New(logger.DefaultOptions())

	a := New(b, registry, factory, store, history, log, cfg)
	return a, b, mockProvider, registry
}

// TestMultiTurnContextContinuity tests that agent maintains context across turns
func TestMultiTurnContextContinuity(t *testing.T) {
	a, b, mockProvider, _ := setupMultiTurnAgent(t)
	defer b.Close()

	// Queue 3 responses for 3 turns
	mockProvider.AddResponse(&providers.LLMResponse{Content: "Response 1"})
	mockProvider.AddResponse(&providers.LLMResponse{Content: "Response 2"})
	mockProvider.AddResponse(&providers.LLMResponse{Content: "Response 3"})

	outCh := b.Subscribe(TopicOutbound)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	sessionID := "multi-turn-session"

	// Turn 1
	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: sessionID,
		Content:   "My name is Alice",
		Role:      bus.RoleUser,
	})
	<-outCh

	// Turn 2
	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: sessionID,
		Content:   "What is my name?",
		Role:      bus.RoleUser,
	})
	<-outCh

	// Turn 3
	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: sessionID,
		Content:   "Remember my name for next time",
		Role:      bus.RoleUser,
	})
	<-outCh

	// Verify 3 LLM calls were made
	if mockProvider.CallCount != 3 {
		t.Errorf("expected 3 LLM calls, got %d", mockProvider.CallCount)
	}

	// Verify history contains all 3 user messages
	lastMessages := mockProvider.LastMessages
	userMsgCount := 0
	for _, msg := range lastMessages {
		if msg.Role == providers.RoleUser {
			userMsgCount++
		}
	}
	if userMsgCount != 3 {
		t.Errorf("expected 3 user messages in history, got %d", userMsgCount)
	}

	// Verify session was saved
	sess, ok := a.sessionStore.Get(sessionID)
	if !ok {
		t.Fatal("expected session to be saved")
	}
	if len(sess.Messages) < 6 { // 3 user + 3 assistant
		t.Errorf("expected at least 6 messages in session, got %d", len(sess.Messages))
	}
}

// TestMultiTurnToolCallHistory tests that tool call results persist across turns
func TestMultiTurnToolCallHistory(t *testing.T) {
	a, b, mockProvider, _ := setupMultiTurnAgent(t)
	defer b.Close()

	// Turn 1: LLM calls a tool
	mockProvider.AddResponse(&providers.LLMResponse{
		Content:   "",
		ToolCalls: []providers.ToolCall{{ID: "call-1", Name: "echo", Arguments: map[string]interface{}{"input": "test"}}},
	})
	// Turn 2: LLM responds with text
	mockProvider.AddResponse(&providers.LLMResponse{Content: "Done"})

	outCh := b.Subscribe(TopicOutbound)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	sessionID := "tool-history-session"

	// Turn 1: triggers tool call
	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: sessionID,
		Content:   "Echo test",
		Role:      bus.RoleUser,
	})
	<-outCh

	// Turn 2: verify tool result is in context
	b.Publish(TopicInbound, bus.InboundMessage{
		SessionID: sessionID,
		Content:   "What was the result?",
		Role:      bus.RoleUser,
	})
	<-outCh

	// Verify LLM saw the tool result in history
	lastMessages := mockProvider.LastMessages
	hasToolResult := false
	for _, msg := range lastMessages {
		if msg.Role == providers.RoleTool && strings.Contains(msg.Content, "echoed") {
			hasToolResult = true
			break
		}
	}
	if !hasToolResult {
		t.Error("expected tool result to be in LLM history for turn 2")
	}
}

// TestSessionPersistenceAcrossRestarts tests session save/load cycle
func TestSessionPersistenceAcrossRestarts(t *testing.T) {
	// Create a shared store for both agents
	sharedStore := session.NewMemoryStore()
	
	// Create agent 1 with shared store
	registry1 := tools.NewRegistry()
	registry1.Register(&echoTool{})
	mockProvider1 := providers.NewMockProvider("mock")
	factory1 := providers.NewFactory()
	factory1.Register("mock", mockProvider1)
	cfg1 := &config.Config{
		Agents: config.AgentConfig{
			Defaults: config.AgentDefaults{
				ModelName:    "mock/echo",
				MaxTokens:    4096,
				SystemPrompt: "You are a helpful assistant.",
			},
		},
	}
	b1 := bus.New()
	history1 := session.NewHistoryManager(4096)
	log1 := logger.New(logger.DefaultOptions())
	a1 := New(b1, registry1, factory1, sharedStore, history1, log1, cfg1)

	mockProvider1.AddResponse(&providers.LLMResponse{Content: "Hello"})
	mockProvider1.AddResponse(&providers.LLMResponse{Content: "Welcome back"})

	outCh1 := b1.Subscribe(TopicOutbound)
	ctx1, cancel1 := context.WithCancel(context.Background())

	startAgent(ctx1, a1)

	b1.Publish(TopicInbound, bus.InboundMessage{SessionID: "persist-test", Content: "Hi", Role: bus.RoleUser})
	<-outCh1
	b1.Publish(TopicInbound, bus.InboundMessage{SessionID: "persist-test", Content: "Bye", Role: bus.RoleUser})
	<-outCh1

	// Verify session exists
	sess1, ok := sharedStore.Get("persist-test")
	if !ok {
		t.Fatal("expected session to exist")
	}
	msgCount1 := len(sess1.Messages)
	if msgCount1 < 4 {
		t.Errorf("expected at least 4 messages, got %d", msgCount1)
	}

	// Create agent 2 with same shared store (simulates restart)
	registry2 := tools.NewRegistry()
	registry2.Register(&echoTool{})
	mockProvider2 := providers.NewMockProvider("mock")
	factory2 := providers.NewFactory()
	factory2.Register("mock", mockProvider2)
	cfg2 := &config.Config{
		Agents: config.AgentConfig{
			Defaults: config.AgentDefaults{
				ModelName:    "mock/echo",
				MaxTokens:    4096,
				SystemPrompt: "You are a helpful assistant.",
			},
		},
	}
	b2 := bus.New()
	history2 := session.NewHistoryManager(4096)
	log2 := logger.New(logger.DefaultOptions())
	a2 := New(b2, registry2, factory2, sharedStore, history2, log2, cfg2)

	mockProvider2.AddResponse(&providers.LLMResponse{Content: "Resumed"})

	outCh2 := b2.Subscribe(TopicOutbound)
	ctx2, cancel2 := context.WithCancel(context.Background())

	startAgent(ctx2, a2)

	// Resume session
	b2.Publish(TopicInbound, bus.InboundMessage{SessionID: "persist-test", Content: "I'm back", Role: bus.RoleUser})
	<-outCh2

	// Verify session was loaded and new message added
	sess2, ok := sharedStore.Get("persist-test")
	if !ok {
		t.Fatal("expected session to exist after resume")
	}
	if len(sess2.Messages) <= msgCount1 {
		t.Errorf("expected more messages after resume, got %d (was %d)", len(sess2.Messages), msgCount1)
	}

	cancel1()
	cancel2()
	b1.Close()
	b2.Close()
}
