package agent

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	toolexec "github.com/strings77wzq/golem/core/tools/exec"
	"github.com/strings77wzq/golem/core/tools/fileops"
	"github.com/strings77wzq/golem/core/tools/websearch"
	"github.com/strings77wzq/golem/foundation/logger"
)

// QuickStart creates an Agent from a config file with sensible defaults.
// This is the simplest way to embed Golem into a Go project:
//
//	ag, err := agent.QuickStart("~/.golem/config.json")
//	if err != nil { log.Fatal(err) }
//	response, err := ag.Chat(ctx, "Hello, what can you do?")
func QuickStart(configPath string) (*Agent, error) {
	return QuickStartWithModel(configPath, "")
}

// QuickStartWithModel creates an Agent from a config file, overriding the model.
func QuickStartWithModel(configPath, modelName string) (*Agent, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = config.DefaultConfig()
		} else {
			return nil, fmt.Errorf("loading config: %w", err)
		}
	}

	if modelName != "" {
		cfg.Agents.Defaults.ModelName = modelName
	}

	return NewFromConfig(cfg)
}

// NewFromConfig creates an Agent from a config, building all dependencies.
func NewFromConfig(cfg *config.Config) (*Agent, error) {
	log := logger.New(logger.DefaultOptions())
	b := bus.New()
	b.SetOnDropped(func(topic string, count int) {
		log.Warn("bus message dropped", "topic", topic, "count", count)
	})
	workspace, _ := os.Getwd()
	registry := newDefaultToolRegistry(workspace)
	factory := newProviderFactory(cfg)
	store := session.NewMemoryStore()

	return New(b, registry, factory, store, log, cfg), nil
}

// Chat sends a single message and returns the response.
// It creates an isolated session for each call.
func (a *Agent) Chat(ctx context.Context, message string) (string, error) {
	return a.HandleMessage(ctx, generateSessionID(), message)
}

// ChatStream sends a message and streams tokens via the callback.
// It creates an isolated session for each call.
func (a *Agent) ChatStream(ctx context.Context, message string, onToken func(token string)) (string, error) {
	tokens := make(chan string, 64)
	// NOTE: Do NOT close tokens here — HandleMessageStream defers close(tokens).

	go func() {
		for token := range tokens {
			if onToken != nil {
				onToken(token)
			}
		}
	}()

	return "", a.HandleMessageStream(ctx, generateSessionID(), message, tokens)
}

// ChatWithSession sends a message within a persistent session.
// Use the same sessionID to maintain conversation context.
func (a *Agent) ChatWithSession(ctx context.Context, sessionID, message string) (string, error) {
	return a.HandleMessage(ctx, sessionID, message)
}

func newDefaultToolRegistry(workspace string) *tools.Registry {
	registry := tools.NewRegistry()
	registry.Register(toolexec.New(workspace))             //nolint:errcheck
	registry.Register(fileops.NewFileReadTool(workspace))  //nolint:errcheck
	registry.Register(fileops.NewFileWriteTool(workspace)) //nolint:errcheck
	registry.Register(fileops.NewFileListTool(workspace))  //nolint:errcheck
	registry.Register(websearch.New())                     //nolint:errcheck
	return registry
}

func newProviderFactory(cfg *config.Config) *providers.Factory {
	factory := providers.NewFactory()
	registered := make(map[string]bool)

	for _, entry := range cfg.ModelList {
		vendor := entry.Vendor()
		if registered[vendor] {
			continue
		}
		registered[vendor] = true

		// Use Provider Registry (vendors registered via init() in their packages)
		provider, err := providers.GlobalRegistry.Create(providers.ProviderConfig{
			Vendor:  vendor,
			APIKey:  entry.APIKey,
			APIBase: entry.APIBase,
		})
		if err != nil {
			// Unknown vendor — skip
			continue
		}
		factory.Register(vendor, provider)
	}

	if !registered["mock"] {
		factory.Register("mock", providers.NewMockProvider("mock"))
	}

	return factory
}

func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return filepath.Base(os.TempDir())
	}
	return fmt.Sprintf("%x", b)
}
