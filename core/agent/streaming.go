package agent

import (
	"context"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/tools"
)

// invokeProvider calls the LLM provider, using streaming when possible.
// Streaming is disabled when tools are present because mid-stream tool call
// arguments require buffering the entire stream to parse JSON.
func (a *Agent) invokeProvider(
	ctx context.Context,
	provider providers.LLMProvider,
	messages []providers.Message,
	toolDefs []tools.ToolDefinition,
	modelName string,
	streamFinal bool,
	onToken func(string),
	sessionID string,
	emit func(bus.OutboundMessage),
) (*providers.LLMResponse, bool, error) {
	sp, ok := provider.(providers.StreamingProvider)
	canStream := ok && streamFinal && len(toolDefs) == 0
	if !canStream {
		resp, err := provider.Chat(ctx, messages, toolDefs, modelName, nil)
		return resp, false, err
	}
	resp, err := sp.ChatStream(ctx, messages, toolDefs, modelName, nil, a.wrapTokenEmitter(sessionID, emit, onToken))
	return resp, err == nil, err
}

// wrapTokenEmitter creates a token callback that sends to both onToken and emit.
func (a *Agent) wrapTokenEmitter(sessionID string, emit func(bus.OutboundMessage), onToken func(string)) func(string) {
	return func(token string) {
		if onToken != nil {
			onToken(token)
		}
		if emit != nil {
			emit(bus.OutboundMessage{
				SessionID:  sessionID,
				Content:    token,
				Role:       bus.RoleAssistant,
				Done:       false,
				TokenDelta: token,
			})
		}
	}
}

// emitError sends an error message to the output channel.
func (a *Agent) emitError(sessionID, errMsg string, emit func(bus.OutboundMessage)) {
	if emit == nil {
		return
	}
	emit(bus.OutboundMessage{
		SessionID: sessionID,
		Content:   errMsg,
		Role:      bus.RoleAssistant,
		Done:      true,
	})
}
