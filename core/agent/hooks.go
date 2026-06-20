package agent

import (
	"context"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/tools"
)

// Hooks provides lifecycle callbacks for the agent loop.
// Each hook is optional — nil hooks are silently skipped.
type Hooks struct {
	BeforeMessage func(ctx context.Context, sessionID string, message string) error
	AfterMessage  func(ctx context.Context, sessionID string, response string) error
	BeforeLLM     func(ctx context.Context, messages []providers.Message) error
	AfterLLM      func(ctx context.Context, response *providers.LLMResponse) error
	BeforeTool    func(ctx context.Context, call providers.ToolCall) error
	AfterTool     func(ctx context.Context, call providers.ToolCall, result *tools.ToolResult) error
	OnError       func(ctx context.Context, err error) error
	PreToolShell  *ShellHook // shell command hook before tool execution
	PostToolShell *ShellHook // shell command hook after tool execution
}

// HookChain wraps lifecycle hooks with nil-safe dispatch.
// It uses a getter function to read hooks at call time,
// so hooks set after construction are automatically picked up.
type HookChain struct {
	getHooks func() *Hooks
}

// NewHookChain creates a HookChain that reads hooks via the given getter.
func NewHookChain(getHooks func() *Hooks) *HookChain {
	return &HookChain{getHooks: getHooks}
}

// RunBeforeMessage fires the BeforeMessage hook if registered.
func (hc *HookChain) RunBeforeMessage(ctx context.Context, sessionID string, message string) {
	if h := hc.getHooks(); h != nil && h.BeforeMessage != nil {
		if err := h.BeforeMessage(ctx, sessionID, message); err != nil {
			_ = err
		}
	}
}

// RunAfterMessage fires the AfterMessage hook if registered.
func (hc *HookChain) RunAfterMessage(ctx context.Context, sessionID string, response string) {
	if h := hc.getHooks(); h != nil && h.AfterMessage != nil {
		if err := h.AfterMessage(ctx, sessionID, response); err != nil {
			_ = err
		}
	}
}

// RunBeforeLLM fires the BeforeLLM hook if registered.
func (hc *HookChain) RunBeforeLLM(ctx context.Context, messages []providers.Message) {
	if h := hc.getHooks(); h != nil && h.BeforeLLM != nil {
		if err := h.BeforeLLM(ctx, messages); err != nil {
			_ = err
		}
	}
}

// RunAfterLLM fires the AfterLLM hook if registered.
func (hc *HookChain) RunAfterLLM(ctx context.Context, response *providers.LLMResponse) {
	if h := hc.getHooks(); h != nil && h.AfterLLM != nil {
		if err := h.AfterLLM(ctx, response); err != nil {
			_ = err
		}
	}
}

// RunOnError fires the OnError hook if registered.
func (hc *HookChain) RunOnError(ctx context.Context, err error) {
	if h := hc.getHooks(); h != nil && h.OnError != nil {
		if hookErr := h.OnError(ctx, err); hookErr != nil {
			_ = hookErr
		}
	}
}

// Middleware wraps the message processing pipeline.
// It receives the current message and a next function; calling next
// continues the pipeline, not calling next short-circuits it.
type Middleware func(ctx context.Context, msg *providers.Message, next func(context.Context, *providers.Message) (*providers.LLMResponse, error)) (*providers.LLMResponse, error)
