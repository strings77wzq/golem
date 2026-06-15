package agent

import (
	"context"
	"testing"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/foundation/logger"
)

func TestNewFromConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/echo"
	cfg.ModelList = []config.ModelEntry{
		{ModelName: "mock/echo", Model: "echo", APIKey: "test"},
	}

	ag, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig failed: %v", err)
	}
	if ag == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestChatWithMockProvider(t *testing.T) {
	mock := providers.NewMockProvider("mock")
	mock.AddResponse(&providers.LLMResponse{
		Content: "Hello! I am Golem, a Go-native AI agent.",
	})

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/echo"
	cfg.ModelList = []config.ModelEntry{
		{ModelName: "mock/echo", Model: "echo", APIKey: "test"},
	}

	b := bus.New()
	registry := newDefaultToolRegistry(t.TempDir())
	factory := providers.NewFactory()
	factory.Register("mock", mock)
	store := session.NewMemoryStore()
	log := logger.NopLogger()

	ag := New(b, registry, factory, store, log, cfg)

	response, err := ag.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if response != "Hello! I am Golem, a Go-native AI agent." {
		t.Errorf("unexpected response: %q", response)
	}
}

func TestChatWithSession(t *testing.T) {
	mock := providers.NewMockProvider("mock")
	mock.AddResponse(&providers.LLMResponse{Content: "Response 1"})
	mock.AddResponse(&providers.LLMResponse{Content: "Response 2"})

	b := bus.New()
	registry := newDefaultToolRegistry(t.TempDir())
	factory := providers.NewFactory()
	factory.Register("mock", mock)
	store := session.NewMemoryStore()
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/echo"
	cfg.ModelList = []config.ModelEntry{
		{ModelName: "mock/echo", Model: "echo", APIKey: "test"},
	}

	ag := New(b, registry, factory, store, log, cfg)

	sessionID := "test-session"
	resp1, err := ag.ChatWithSession(context.Background(), sessionID, "Hello")
	if err != nil {
		t.Fatalf("ChatWithSession failed: %v", err)
	}
	if resp1 != "Response 1" {
		t.Errorf("expected 'Response 1', got %q", resp1)
	}

	// Second message in same session
	resp2, err := ag.ChatWithSession(context.Background(), sessionID, "How are you?")
	if err != nil {
		t.Fatalf("ChatWithSession second call failed: %v", err)
	}
	if resp2 != "Response 2" {
		t.Errorf("expected 'Response 2', got %q", resp2)
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()
	if id1 == id2 {
		t.Error("expected different session IDs")
	}
	if len(id1) == 0 {
		t.Error("expected non-empty session ID")
	}
}
