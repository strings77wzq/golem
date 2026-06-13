package agent

import (
	"context"
	"testing"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/foundation/logger"
)

// mockToolForTest is a simple mock tool for testing.
type mockToolForTest struct {
	name    string
	execute func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error)
}

func (m *mockToolForTest) Name() string        { return m.name }
func (m *mockToolForTest) Description() string { return "mock tool: " + m.name }
func (m *mockToolForTest) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "input", Type: "string", Description: "input", Required: true},
	}
}
func (m *mockToolForTest) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	if m.execute != nil {
		return m.execute(ctx, args)
	}
	return &tools.ToolResult{ForLLM: "mock result", ForUser: "done"}, nil
}

func setupIntegrationTest(t *testing.T) (*Agent, bus.Bus, *providers.MockProvider) {
	t.Helper()

	busInst := bus.New()

	registry := tools.NewRegistry()
	registry.Register(&mockToolForTest{name: "test_tool"})

	factory := providers.NewFactory()
	mockProvider := providers.NewMockProvider("mock")
	factory.Register("mock", mockProvider)

	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test"
	cfg.ModelList = []config.ModelEntry{
		{ModelName: "mock/test", Model: "test", APIKey: "test"},
	}

	ag := New(busInst, registry, factory, store, history, log, cfg)
	return ag, busInst, mockProvider
}

func TestIntegrationSimpleChat(t *testing.T) {
	ag, _, mockProvider := setupIntegrationTest(t)
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Hello! I can help you with that.",
	})

	ctx := context.Background()
	response, err := ag.Chat(ctx, "hi")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if response == "" {
		t.Error("expected non-empty response")
	}
}

func TestIntegrationToolCall(t *testing.T) {
	b := bus.New()
	registry := tools.NewRegistry()

	// Register a mock tool that returns a specific result
	registry.Register(&mockToolForTest{
		name: "get_weather",
		execute: func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
			return &tools.ToolResult{ForLLM: "Sunny, 25°C", ForUser: "Weather: Sunny, 25°C"}, nil
		},
	})

	factory := providers.NewFactory()
	mockProvider := providers.NewMockProvider("mock")
	factory.Register("mock", mockProvider)

	// First LLM response: request tool call
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "",
		ToolCalls: []providers.ToolCall{
			{ID: "call-1", Name: "get_weather", Arguments: map[string]interface{}{"location": "Beijing"}},
		},
	})
	// Second LLM response: final answer after tool result
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "The weather in Beijing is sunny, 25°C.",
	})

	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test"

	ag := New(b, registry, factory, store, history, log, cfg)

	ctx := context.Background()
	response, err := ag.Chat(ctx, "What's the weather in Beijing?")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if response == "" {
		t.Error("expected non-empty response")
	}
}

func TestIntegrationMultiTurnSession(t *testing.T) {
	b := bus.New()
	registry := tools.NewRegistry()
	factory := providers.NewFactory()
	mockProvider := providers.NewMockProvider("mock")
	factory.Register("mock", mockProvider)

	mockProvider.AddResponse(&providers.LLMResponse{Content: "My name is Golem."})
	mockProvider.AddResponse(&providers.LLMResponse{Content: "You said your name is Golem."})

	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test"

	ag := New(b, registry, factory, store, history, log, cfg)
	sessionID := "multi-turn-test"

	ctx := context.Background()

	// Turn 1
	resp1, err := ag.ChatWithSession(ctx, sessionID, "My name is Golem")
	if err != nil {
		t.Fatalf("Turn 1 failed: %v", err)
	}
	if resp1 == "" {
		t.Error("expected non-empty response for turn 1")
	}

	// Turn 2
	resp2, err := ag.ChatWithSession(ctx, sessionID, "What's my name?")
	if err != nil {
		t.Fatalf("Turn 2 failed: %v", err)
	}
	if resp2 == "" {
		t.Error("expected non-empty response for turn 2")
	}
}

func TestIntegrationTraceID(t *testing.T) {
	b := bus.New()
	registry := tools.NewRegistry()
	factory := providers.NewFactory()
	mockProvider := providers.NewMockProvider("mock")
	factory.Register("mock", mockProvider)

	mockProvider.AddResponse(&providers.LLMResponse{Content: "ok"})

	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test"

	ag := New(b, registry, factory, store, history, log, cfg)

	ctx := context.Background()
	_, err := ag.Chat(ctx, "test")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	// Trace ID is generated internally, just verify no crash
}

func TestIntegrationMetricsIncrement(t *testing.T) {
	b := bus.New()
	registry := tools.NewRegistry()
	factory := providers.NewFactory()
	mockProvider := providers.NewMockProvider("mock")
	factory.Register("mock", mockProvider)

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "response",
		Usage:   providers.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	})

	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test"

	ag := New(b, registry, factory, store, history, log, cfg)

	// Record initial metric values
	initMessages := AgentMessagesTotal.Value()
	initLLMCalls := AgentLLMCalls.Value()
	initLLMTokens := AgentLLMTokens.Value()

	ctx := context.Background()
	_, err := ag.Chat(ctx, "test metrics")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// Verify metrics incremented
	if AgentMessagesTotal.Value() <= initMessages {
		t.Error("AgentMessagesTotal not incremented")
	}
	if AgentLLMCalls.Value() <= initLLMCalls {
		t.Error("AgentLLMCalls not incremented")
	}
	if AgentLLMTokens.Value() <= initLLMTokens {
		t.Error("AgentLLMTokens not incremented")
	}
}

func TestIntegrationContextManager(t *testing.T) {
	b := bus.New()
	registry := tools.NewRegistry()
	factory := providers.NewFactory()
	mockProvider := providers.NewMockProvider("mock")
	factory.Register("mock", mockProvider)

	mockProvider.AddResponse(&providers.LLMResponse{Content: "ok"})

	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test"

	ag := New(b, registry, factory, store, history, log, cfg)

	// Verify context manager is initialized
	if ag.contextManager == nil {
		t.Error("expected context manager to be initialized")
	}
}
