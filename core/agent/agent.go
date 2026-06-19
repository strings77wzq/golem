// Package agent implements the ReAct (Reason + Act) loop that drives the AI
// assistant. It receives user messages from the message bus, calls the LLM,
// dispatches tool calls, and publishes responses back onto the bus.
// The Agent type is the central coordinator; use [New] to construct one and
// [Agent.Start] to run its event loop.
package agent

import (
	"context"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	golemctx "github.com/strings77wzq/golem/core/context"
	"github.com/strings77wzq/golem/core/planner"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/core/usage"
	"github.com/strings77wzq/golem/foundation/logger"
)

// Router abstracts the routing.Router for dependency injection.
type Router interface {
	Chat(ctx context.Context, modelName string, messages []providers.Message, toolDefs []tools.ToolDefinition, opts *providers.ChatOptions) (*providers.LLMResponse, error)
}

const (
	DefaultMaxToolIterations = 25
	TopicInbound             = "inbound"
	TopicOutbound            = "outbound"
)

// Runner abstracts the agent lifecycle for dependency injection and testing.
type Runner interface {
	Start(ctx context.Context)
}

// MessageHandler provides direct request/response handling for gateways and TUIs.
type MessageHandler interface {
	HandleMessage(ctx context.Context, sessionID string, message string) (string, error)
	HandleMessageStream(ctx context.Context, sessionID string, message string, tokens chan<- string) error
	HandleMessageStreamWithProgress(ctx context.Context, sessionID string, message string, tokens chan<- string, progress chan<- bus.OutboundMessage) error
	HandleCompact(ctx context.Context, sessionID string) (string, error)
	HandleFork(ctx context.Context, originalSessionID string, upToIndex int, newMessage string) (string, error)
}

// Agent is the core orchestrator that runs the ReAct loop
type Agent struct {
	bus               bus.Bus
	toolRegistry      *tools.Registry
	providerFactory   *providers.Factory
	router            Router
	sessionStore      session.SessionStore
	contextManager    *golemctx.Manager
	planner           *planner.Planner
	toolSelector      *ToolSelector
	reflector         *Reflector
	logger            logger.Logger
	config            *config.Config
	systemPrompt      string
	maxToolIterations int
	tracker           *usage.Tracker
	hooks             *Hooks
	compactor         *Compactor
	planEnabled       bool
}

// Option is a functional option for configuring Agent
type Option func(*Agent)

// WithMaxToolIterations sets the maximum number of ReAct loop iterations
func WithMaxToolIterations(n int) Option {
	return func(a *Agent) {
		a.maxToolIterations = n
	}
}

// WithSystemPrompt sets the default system prompt
func WithSystemPrompt(prompt string) Option {
	return func(a *Agent) {
		a.systemPrompt = prompt
	}
}

// WithTracker sets the usage tracker for cost accounting
func WithTracker(tracker *usage.Tracker) Option {
	return func(a *Agent) {
		a.tracker = tracker
	}
}

// WithHooks sets lifecycle hooks for the agent loop.
func WithHooks(hooks *Hooks) Option {
	return func(a *Agent) {
		a.hooks = hooks
	}
}

// WithPlanner enables planning mode for complex tasks.
func WithPlanner(llm providers.LLMProvider, model string) Option {
	return func(a *Agent) {
		a.planner = planner.NewPlanner(llm, model)
		a.toolSelector = NewToolSelector(a.toolRegistry)
		a.reflector = NewReflector()
		a.planEnabled = true
	}
}

// WithCompactor sets the LLM-driven session compactor.
func WithCompactor(compactor *Compactor) Option {
	return func(a *Agent) {
		a.compactor = compactor
	}
}

// WithRouter sets a routing-aware provider resolver.
// When set, the agent uses router.Chat() instead of direct factory resolution.
func WithRouter(r Router) Option {
	return func(a *Agent) {
		a.router = r
	}
}

// New creates a new Agent with the given dependencies
func New(
	b bus.Bus,
	registry *tools.Registry,
	factory *providers.Factory,
	store session.SessionStore,
	log logger.Logger,
	cfg *config.Config,
	opts ...Option,
) *Agent {
	totalTokens := cfg.Agents.Defaults.MaxTokens
	if totalTokens <= 0 {
		totalTokens = 8192
	}

	a := &Agent{
		bus:               b,
		toolRegistry:      registry,
		providerFactory:   factory,
		sessionStore:      store,
		contextManager:    golemctx.NewManager(totalTokens, cfg.Agents.Defaults.SystemPrompt),
		logger:            log,
		config:            cfg,
		systemPrompt:      cfg.Agents.Defaults.SystemPrompt,
		maxToolIterations: DefaultMaxToolIterations,
		tracker:           usage.NewTracker(),
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Start begins listening for inbound messages and processing them
func (a *Agent) Start(ctx context.Context) {
	a.logger.Info("agent started")
	defer a.logger.Info("agent stopped")

	// Subscribe to inbound messages
	ch := a.bus.Subscribe(TopicInbound)
	defer a.bus.Unsubscribe(TopicInbound, ch)

	// Process messages until context is cancelled
	for {
		select {
		case <-ctx.Done():
			return
		case raw := <-ch:
			msg, ok := raw.(bus.InboundMessage)
			if !ok {
				a.logger.Error("invalid inbound message type", nil)
				continue
			}
			a.handleMessage(ctx, msg)
		}
	}
}

// Compile-time interface compliance check.
var _ Runner = (*Agent)(nil)
var _ MessageHandler = (*Agent)(nil)
