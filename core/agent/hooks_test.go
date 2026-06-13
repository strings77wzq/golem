package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/foundation/logger"
)

func TestHooksBeforeMessage(t *testing.T) {
	called := false
	mock := providers.NewMockProvider("mock")
	mock.AddResponse(&providers.LLMResponse{Content: "ok"})

	b := bus.New()
	registry := newDefaultToolRegistry(t.TempDir())
	factory := providers.NewFactory()
	factory.Register("mock", mock)
	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test"
	cfg.ModelList = []config.ModelEntry{{ModelName: "mock/test", Model: "test", APIKey: "test"}}

	hooks := &Hooks{
		BeforeMessage: func(ctx context.Context, sessionID string, message string) error {
			called = true
			if message != "hello" {
				t.Errorf("expected 'hello', got %q", message)
			}
			return nil
		},
	}

	ag := New(b, registry, factory, store, history, log, cfg, WithHooks(hooks))
	_, err := ag.Chat(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if !called {
		t.Error("BeforeMessage hook was not called")
	}
}

func TestHooksAfterLLM(t *testing.T) {
	called := false
	mock := providers.NewMockProvider("mock")
	mock.AddResponse(&providers.LLMResponse{Content: "response"})

	b := bus.New()
	registry := newDefaultToolRegistry(t.TempDir())
	factory := providers.NewFactory()
	factory.Register("mock", mock)
	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test"
	cfg.ModelList = []config.ModelEntry{{ModelName: "mock/test", Model: "test", APIKey: "test"}}

	hooks := &Hooks{
		AfterLLM: func(ctx context.Context, resp *providers.LLMResponse) error {
			called = true
			if resp.Content != "response" {
				t.Errorf("expected 'response', got %q", resp.Content)
			}
			return nil
		},
	}

	ag := New(b, registry, factory, store, history, log, cfg, WithHooks(hooks))
	_, err := ag.Chat(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if !called {
		t.Error("AfterLLM hook was not called")
	}
}

func TestHooksOnError(t *testing.T) {
	called := false
	mock := providers.NewMockProvider("mock")
	// No responses queued — will fail

	b := bus.New()
	registry := newDefaultToolRegistry(t.TempDir())
	factory := providers.NewFactory()
	factory.Register("mock", mock)
	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test"
	cfg.ModelList = []config.ModelEntry{{ModelName: "mock/test", Model: "test", APIKey: "test"}}

	hooks := &Hooks{
		OnError: func(ctx context.Context, err error) error {
			called = true
			return nil
		},
	}

	ag := New(b, registry, factory, store, history, log, cfg, WithHooks(hooks))
	_, err := ag.Chat(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	if !called {
		t.Error("OnError hook was not called")
	}
}

func TestHooksErrorPropagation(t *testing.T) {
	hookErr := errors.New("hook failed")
	mock := providers.NewMockProvider("mock")
	mock.AddResponse(&providers.LLMResponse{Content: "ok"})

	b := bus.New()
	registry := newDefaultToolRegistry(t.TempDir())
	factory := providers.NewFactory()
	factory.Register("mock", mock)
	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test"
	cfg.ModelList = []config.ModelEntry{{ModelName: "mock/test", Model: "test", APIKey: "test"}}

	hooks := &Hooks{
		BeforeMessage: func(ctx context.Context, sessionID string, message string) error {
			return hookErr
		},
	}

	ag := New(b, registry, factory, store, history, log, cfg, WithHooks(hooks))
	// Agent should still work even if hook returns error
	_, err := ag.Chat(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
}

func TestNoHooks(t *testing.T) {
	mock := providers.NewMockProvider("mock")
	mock.AddResponse(&providers.LLMResponse{Content: "ok"})

	b := bus.New()
	registry := newDefaultToolRegistry(t.TempDir())
	factory := providers.NewFactory()
	factory.Register("mock", mock)
	store := session.NewMemoryStore()
	history := session.NewHistoryManager(4096)
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test"
	cfg.ModelList = []config.ModelEntry{{ModelName: "mock/test", Model: "test", APIKey: "test"}}

	ag := New(b, registry, factory, store, history, log, cfg)
	_, err := ag.Chat(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
}
