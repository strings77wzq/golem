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
}

// Middleware wraps the message processing pipeline.
// It receives the current message and a next function; calling next
// continues the pipeline, not calling next short-circuits it.
type Middleware func(ctx context.Context, msg *providers.Message, next func(context.Context, *providers.Message) (*providers.LLMResponse, error)) (*providers.LLMResponse, error)
