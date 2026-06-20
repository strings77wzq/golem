package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/planner"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/core/usage"
)

func (a *Agent) handleMessage(ctx context.Context, msg bus.InboundMessage) {
	_, _, err := a.processMessage(ctx, msg, false, nil, func(out bus.OutboundMessage) {
		a.bus.Publish(TopicOutbound, out)
	})
	if err != nil {
		a.logger.Error("failed to process message", err, "session_id", msg.SessionID)
	}
}

func (a *Agent) HandleMessage(ctx context.Context, sessionID string, message string) (string, error) {
	AgentSessionsActive.Inc()
	defer AgentSessionsActive.Dec()

	resp, _, err := a.processMessage(ctx, bus.InboundMessage{
		SessionID: sessionID,
		Content:   message,
		Role:      bus.RoleUser,
	}, false, nil, nil)
	return resp, err
}

// HandleMessageWithEvents is like HandleMessage but emits structured events
// for each tool call via the emit callback. Used by --json-events for E2E observability.
func (a *Agent) HandleMessageWithEvents(ctx context.Context, sessionID string, message string, emit func(bus.OutboundMessage)) (string, error) {
	AgentSessionsActive.Inc()
	defer AgentSessionsActive.Dec()

	resp, _, err := a.processMessage(ctx, bus.InboundMessage{
		SessionID: sessionID,
		Content:   message,
		Role:      bus.RoleUser,
	}, false, nil, emit)
	return resp, err
}

func (a *Agent) HandleMessageStream(ctx context.Context, sessionID string, message string, tokens chan<- string) error {
	defer close(tokens)
	AgentSessionsActive.Inc()
	defer AgentSessionsActive.Dec()
	streamed := false
	content, _, err := a.processMessage(ctx, bus.InboundMessage{
		SessionID: sessionID,
		Content:   message,
		Role:      bus.RoleUser,
	}, true, func(token string) {
		streamed = true
		tokens <- token
	}, nil)
	// If the response came through a Chat (non-streaming) fallback (e.g. after tool use),
	// deliver the final content as a single token to honour the streaming contract.
	if err == nil && !streamed && content != "" {
		tokens <- content
	}
	return err
}

// HandleCompact compresses the session history by summarizing old messages.
// Uses LLM-driven Compactor when available, falls back to string truncation.
func (a *Agent) HandleCompact(ctx context.Context, sessionID string) (string, error) {
	sess, found := a.sessionStore.Get(sessionID)
	if !found {
		return "no active session to compact", nil
	}

	// Use LLM-driven compactor if available
	if a.compactor != nil {
		// Token budget: use 80% of configured max tokens
		budget := a.config.Agents.Defaults.MaxTokens * 8 / 10
		if budget <= 0 {
			budget = 8192 * 8 / 10
		}
		result, err := a.compactor.Compact(ctx, sess, budget)
		if err != nil {
			a.logger.Error("compactor failed, falling back to truncation", err)
			// Fall through to old truncation
		} else {
			if err := a.sessionStore.Save(sess); err != nil {
				a.logger.Error("failed to save compacted session", err)
			}
			return result, nil
		}
	}

	// Fallback: old string truncation
	messages := sess.GetMessages()
	if len(messages) <= 2 {
		return "session is already minimal, nothing to compact", nil
	}

	beforeCount := len(messages)
	beforeTokens := 0
	for _, msg := range messages {
		for _, r := range msg.Content {
			if r >= 0x4E00 && r <= 0x9FFF {
				beforeTokens++
			} else {
				beforeTokens += 2
			}
		}
	}

	// Build a summary of old messages (keep system + last 4 messages)
	var systemMsg providers.Message
	var recent []providers.Message
	var old []providers.Message

	if messages[0].Role == providers.RoleSystem {
		systemMsg = messages[0]
		rest := messages[1:]
		keepRecent := 4
		if keepRecent > len(rest) {
			keepRecent = len(rest)
		}
		old = rest[:len(rest)-keepRecent]
		recent = rest[len(rest)-keepRecent:]
	} else {
		keepRecent := 4
		if keepRecent > len(messages) {
			keepRecent = len(messages)
		}
		old = messages[:len(messages)-keepRecent]
		recent = messages[len(messages)-keepRecent:]
	}

	if len(old) == 0 {
		return "session is already minimal, nothing to compact", nil
	}

	// Summarize old messages
	summary := a.summarizeMessages(old)

	// Rebuild session with summary + recent
	sess.Clear()
	if systemMsg.Content != "" {
		sess.AddMessage(systemMsg)
	}
	sess.AddMessage(providers.Message{
		Role:    providers.RoleUser,
		Content: fmt.Sprintf("[System: Previous conversation summary]\n%s\n[End summary]", summary),
	})
	sess.AddMessage(providers.Message{
		Role:    providers.RoleAssistant,
		Content: "Understood. I have the context from the previous conversation summary. How can I help?",
	})
	for _, msg := range recent {
		sess.AddMessage(msg)
	}

	if err := a.sessionStore.Save(sess); err != nil {
		a.logger.Error("failed to save compacted session", err)
	}

	afterCount := sess.MessageCount()
	return fmt.Sprintf("compacted: %d messages → %d messages (saved ~%d tokens)", beforeCount, afterCount, beforeTokens/2), nil
}

// summarizeMessages creates a concise summary of old messages.
func (a *Agent) summarizeMessages(messages []providers.Message) string {
	var sb strings.Builder
	sb.WriteString("Key points from the conversation:\n")

	for _, msg := range messages {
		switch msg.Role {
		case providers.RoleUser:
			content := msg.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("- User asked: %s\n", content))
		case providers.RoleAssistant:
			if msg.Content != "" {
				content := msg.Content
				if len(content) > 200 {
					content = content[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("- Assistant responded: %s\n", content))
			} else if len(msg.ToolCalls) > 0 {
				toolNames := make([]string, len(msg.ToolCalls))
				for i, tc := range msg.ToolCalls {
					toolNames[i] = tc.Name
				}
				sb.WriteString(fmt.Sprintf("- Assistant used tools: %s\n", strings.Join(toolNames, ", ")))
			}
		case providers.RoleTool:
			content := msg.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf("- Tool result: %s\n", content))
		}
	}

	return sb.String()
}

// HandleFork creates a forked session from an existing one.
// It copies messages up to (excluding) upToIndex, appends the new message,
// and returns the new session ID.
func (a *Agent) HandleFork(ctx context.Context, originalSessionID string, upToIndex int, newMessage string) (string, error) {
	sess, found := a.sessionStore.Get(originalSessionID)
	if !found {
		return "", fmt.Errorf("session %q not found", originalSessionID)
	}

	forked := sess.Fork(upToIndex, providers.Message{
		Role:    providers.RoleUser,
		Content: newMessage,
	})

	if err := a.sessionStore.Save(forked); err != nil {
		return "", fmt.Errorf("saving forked session: %w", err)
	}

	return forked.ID, nil
}

func (a *Agent) HandleMessageStreamWithProgress(ctx context.Context, sessionID string, message string, tokens chan<- string, progress chan<- bus.OutboundMessage) error {
	defer close(tokens)
	streamed := false

	emitFunc := func(out bus.OutboundMessage) {
		if out.Role == bus.RoleProgress {
			// Send progress to progress channel
			select {
			case progress <- out:
			default:
			}
		} else {
			// Send tokens to token channel
			if out.TokenDelta != "" {
				streamed = true
				tokens <- out.TokenDelta
			}
		}
	}

	content, _, err := a.processMessage(ctx, bus.InboundMessage{
		SessionID: sessionID,
		Content:   message,
		Role:      bus.RoleUser,
	}, true, func(token string) {
		streamed = true
		tokens <- token
	}, emitFunc)

	if err == nil && !streamed && content != "" {
		tokens <- content
	}
	return err
}

// initSession gets or creates a session, injects system prompt, and appends the user message.
func (a *Agent) initSession(ctx context.Context, msg bus.InboundMessage) (*session.Session, error) {
	AgentMessagesTotal.Inc()

	sess, found := a.sessionStore.Get(msg.SessionID)
	if !found {
		sess = session.NewSession(msg.SessionID)
		if err := a.sessionStore.Save(sess); err != nil {
			a.logger.Error("failed to save new session", err)
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	if len(sess.GetMessages()) == 0 && a.systemPrompt != "" {
		sess.AddMessage(providers.Message{
			Role:    providers.RoleSystem,
			Content: a.systemPrompt,
		})
	}

	sess.AddMessage(providers.Message{
		Role:    providers.RoleUser,
		Content: msg.Content,
	})

	return sess, nil
}

// preProcess runs pre-processing hooks, planning dispatch, and auto-compaction.
func (a *Agent) preProcess(ctx context.Context, msg bus.InboundMessage, sess *session.Session, model string, toolDefs []tools.ToolDefinition, streamFinal bool, onToken func(string), emit func(bus.OutboundMessage)) (string, *bus.TokenUsage, bool, error) {
	// Hook: before message processing
	if a.hooks != nil && a.hooks.BeforeMessage != nil {
		if err := a.hooks.BeforeMessage(ctx, msg.SessionID, msg.Content); err != nil {
			a.logger.Error("before_message hook failed", err)
		}
	}

	// Planning mode: decompose complex tasks
	if a.planEnabled && a.planner != nil && a.isComplexTask(msg.Content) {
		content, usage, err := a.processWithPlan(ctx, msg, sess, model, toolDefs, streamFinal, onToken, emit)
		return content, usage, true, err
	}

	// Auto-compact if context is getting too large (80% of token budget)
	if a.compactor != nil {
		budget := a.config.Agents.Defaults.MaxTokens * 8 / 10
		if budget <= 0 {
			budget = 8192 * 8 / 10
		}
		if result, err := a.compactor.Compact(ctx, sess, budget); err == nil && strings.HasPrefix(result, "compacted") {
			a.logger.Info("auto-compacted session", "result", result)
		}
	}

	return "", nil, false, nil
}

// processMessage is the main message processing entry point.
func (a *Agent) processMessage(
	ctx context.Context,
	msg bus.InboundMessage,
	streamFinal bool,
	onToken func(string),
	emit func(bus.OutboundMessage),
) (string, *bus.TokenUsage, error) {
	// Observability: generate trace ID and record metrics
	traceID := NewTraceID()
	ctx = WithTraceID(ctx, traceID)
	startTime := time.Now()

	sess, err := a.initSession(ctx, msg)
	if err != nil {
		a.emitError(msg.SessionID, "failed to create session", emit)
		return "", nil, err
	}

	model := a.config.Agents.Defaults.ModelName
	toolDefs := a.toolRegistry.ListDefinitions()

	// Pre-processing: hooks, planning dispatch, auto-compaction
	content, tokenUsage, handled, err := a.preProcess(ctx, msg, sess, model, toolDefs, streamFinal, onToken, emit)
	if handled {
		return content, tokenUsage, err
	}

	// Core ReAct loop
	content, tokenUsage, err = a.reactLoop(ctx, msg, sess, model, toolDefs, streamFinal, onToken, emit)

	// Hook: after message processing
	if a.hooks != nil && a.hooks.AfterMessage != nil {
		hookMsg := content
		if hookMsg == "" {
			hookMsg = "max tool iterations reached"
		}
		if err := a.hooks.AfterMessage(ctx, msg.SessionID, hookMsg); err != nil {
			a.logger.Error("after_message hook failed", err)
		}
	}

	AgentPlanDuration.Observe(time.Since(startTime).Seconds())

	return content, tokenUsage, err
}

// resolveProvider returns the LLM provider and model name for the given model.
// Uses router if set, otherwise falls back to factory with fallback models.
func (a *Agent) resolveProvider(ctx context.Context, model string, contextMsgs []providers.Message, toolDefs []tools.ToolDefinition) (*providers.LLMResponse, providers.LLMProvider, string, error) {
	if a.router != nil {
		resp, err := a.router.Chat(ctx, model, contextMsgs, toolDefs, &providers.ChatOptions{})
		if err != nil {
			return nil, nil, "", fmt.Errorf("router chat: %w", err)
		}
		if resp == nil {
			return nil, nil, "", fmt.Errorf("router returned nil response for model %q", model)
		}
		return resp, nil, model, nil
	}

	provider, modelName, _, err := a.providerFactory.GetProviderForModelWithFallback(
		model, a.config.Agents.Defaults.FallbackModels,
	)
	if err != nil {
		return nil, nil, "", fmt.Errorf("resolve provider: %w", err)
	}
	return nil, provider, modelName, nil
}

// reactLoop runs the core ReAct (Reason + Act) loop.
// It is shared by processMessage and processWithPlan for consistent behavior.
func (a *Agent) reactLoop(
	ctx context.Context,
	msg bus.InboundMessage,
	sess *session.Session,
	model string,
	toolDefs []tools.ToolDefinition,
	streamFinal bool,
	onToken func(string),
	emit func(bus.OutboundMessage),
) (string, *bus.TokenUsage, error) {
	for i := 0; i < a.maxToolIterations; i++ {
		contextMsgs := a.contextManager.BuildContext(sess, toolDefs, "")

		totalCtxTokens := 0
		for _, msg := range contextMsgs {
			totalCtxTokens += len(msg.Content)
		}
		AgentContextTokens.Set(float64(totalCtxTokens))

		resp, provider, modelName, err := a.resolveProvider(ctx, model, contextMsgs, toolDefs)
		if err != nil {
			a.emitError(msg.SessionID, fmt.Sprintf("provider error: %v", err), emit)
			return "", nil, err
		}

		// Hook: before LLM call
		if a.hooks != nil && a.hooks.BeforeLLM != nil {
			if err := a.hooks.BeforeLLM(ctx, contextMsgs); err != nil {
				a.logger.Error("before_llm hook failed", err)
			}
		}

		AgentLLMCalls.Inc()
		llmStart := time.Now()

		var streamed bool
		if a.router == nil {
			resp, streamed, err = a.invokeProvider(ctx, provider, contextMsgs, toolDefs, modelName, streamFinal, onToken, msg.SessionID, emit)
		}
		llmDuration := time.Since(llmStart)
		AgentLLMLatency.Observe(llmDuration.Seconds())

		if err != nil {
			AgentLLMErrors.Inc()
			if a.hooks != nil && a.hooks.OnError != nil {
				a.hooks.OnError(ctx, err)
			}
			a.emitError(msg.SessionID, fmt.Sprintf("LLM error: %v", err), emit)
			return "", nil, err
		}

		AgentLLMTokens.Add(int64(resp.Usage.TotalTokens))

		// Hook: after LLM call
		if a.hooks != nil && a.hooks.AfterLLM != nil {
			if err := a.hooks.AfterLLM(ctx, resp); err != nil {
				a.logger.Error("after_llm hook failed", err)
			}
		}

		// No tool calls — final response
		if len(resp.ToolCalls) == 0 {
			sess.AddMessage(providers.Message{
				Role:    providers.RoleAssistant,
				Content: resp.Content,
			})
			if err := a.sessionStore.Save(sess); err != nil {
				a.logger.Error("failed to save session", err)
			}

			tokenUsage := &bus.TokenUsage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			}

			if a.tracker != nil && tokenUsage != nil {
				a.tracker.Record(msg.SessionID, modelName, usage.TokenUsage{
					PromptTokens:     tokenUsage.PromptTokens,
					CompletionTokens: tokenUsage.CompletionTokens,
					TotalTokens:      tokenUsage.TotalTokens,
				})
				pricing := usage.GetPricing(modelName)
				costCents := float64(tokenUsage.PromptTokens)*pricing.InputPerToken +
					float64(tokenUsage.CompletionTokens)*pricing.OutputPerToken
				AgentLLMCostUSD.Add(int64(costCents * 10000))
			}

			if emit != nil {
				finalContent := resp.Content
				if streamed {
					finalContent = ""
				}
				emit(bus.OutboundMessage{
					SessionID: msg.SessionID,
					Content:   finalContent,
					Role:      bus.RoleAssistant,
					Done:      true,
					Usage:     tokenUsage,
				})
			}

			return resp.Content, tokenUsage, nil
		}

		// Tool calls — execute and continue loop
		sess.AddMessage(providers.Message{
			Role:      providers.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		a.runToolExecution(ctx, resp, sess, msg, emit)

		if err := a.sessionStore.Save(sess); err != nil {
			a.logger.Error("failed to save session", err)
		}
	}

	if emit != nil {
		emit(bus.OutboundMessage{
			SessionID: msg.SessionID,
			Content:   "max tool iterations reached",
			Role:      bus.RoleAssistant,
			Done:      true,
		})
	}

	return "max tool iterations reached", nil, nil
}

// isComplexTask determines if a message should use planning mode.
// Uses word count as a signal — longer messages with multiple instructions
// are more likely to benefit from decomposition.
func (a *Agent) isComplexTask(message string) bool {
	words := strings.Fields(message)
	return len(words) > 30
}

// processWithPlan executes a message using the planner.
func (a *Agent) processWithPlan(
	ctx context.Context,
	msg bus.InboundMessage,
	sess *session.Session,
	model string,
	toolDefs []tools.ToolDefinition,
	streamFinal bool,
	onToken func(string),
	emit func(bus.OutboundMessage),
) (string, *bus.TokenUsage, error) {
	startTime := time.Now()

	// Step 1: Decompose into plan
	if emit != nil {
		emit(bus.OutboundMessage{
			SessionID:    msg.SessionID,
			Content:      "分解任务中...",
			Role:         bus.RoleProgress,
			ProgressType: bus.ProgressPlanStart,
		})
	}

	plan, err := a.planner.Decompose(ctx, msg.Content, toolDefs)
	if err != nil {
		a.logger.Error("planner decompose failed", err)
		return a.processMessage(ctx, msg, streamFinal, onToken, emit)
	}

	AgentPlanSteps.Set(float64(len(plan.Steps)))

	// Emit plan steps
	if emit != nil {
		for i, step := range plan.Steps {
			emit(bus.OutboundMessage{
				SessionID:    msg.SessionID,
				Content:      step.Description,
				Role:         bus.RoleProgress,
				ProgressType: bus.ProgressPlanStep,
				StepCurrent:  i + 1,
				StepTotal:    len(plan.Steps),
			})
		}
	}

	// Step 2: Execute each step
	plan.MarkRunning()
	var lastResponse string

	for a.planner.ShouldContinue(plan) {
		step := plan.NextPendingStep()
		if step == nil {
			break
		}

		step.Status = planner.StepRunning

		if emit != nil {
			emit(bus.OutboundMessage{
				SessionID:    msg.SessionID,
				Content:      step.Description,
				Role:         bus.RoleProgress,
				ProgressType: bus.ProgressStepStart,
				StepCurrent:  a.getStepIndex(plan, step.ID) + 1,
				StepTotal:    len(plan.Steps),
			})
		}

		// Select tools for this step
		stepTools := a.toolSelector.Select(step.Description, step.ToolHints, 8)

		// Execute via mini ReAct loop (max 5 iterations per step)
		stepResult := a.executeStep(ctx, sess, step, stepTools, model, 5, msg.SessionID, emit)

		step.Result = stepResult
		step.Status = planner.StepDone

		// Reflect on result
		reflection := a.reflector.Evaluate(step, stepResult, nil)
		if !reflection.Success && reflection.ShouldRevise {
			step.Status = planner.StepFailed
			step.Error = reflection.Reason

			if emit != nil {
				emit(bus.OutboundMessage{
					SessionID:    msg.SessionID,
					Content:      reflection.Reason,
					Role:         bus.RoleProgress,
					ProgressType: bus.ProgressStepFailed,
					StepCurrent:  a.getStepIndex(plan, step.ID) + 1,
					StepTotal:    len(plan.Steps),
				})
			}
			plan, err = a.planner.Revise(ctx, plan, step.ID, stepResult)
			if err != nil {
				a.logger.Error("planner revise failed", err)
			}
			AgentPlanRevisions.Inc()
			continue
		}

		if emit != nil {
			emit(bus.OutboundMessage{
				SessionID:    msg.SessionID,
				Content:      step.Description,
				Role:         bus.RoleProgress,
				ProgressType: bus.ProgressStepDone,
				StepCurrent:  a.getStepIndex(plan, step.ID) + 1,
				StepTotal:    len(plan.Steps),
			})
		}

		lastResponse = stepResult
	}

	plan.MarkComplete()
	AgentPlanDuration.Observe(time.Since(startTime).Seconds())

	if emit != nil {
		emit(bus.OutboundMessage{
			SessionID:    msg.SessionID,
			Content:      plan.Progress(),
			Role:         bus.RoleProgress,
			ProgressType: bus.ProgressPlanDone,
		})
	}

	return lastResponse, nil, nil
}

// getStepIndex returns the 0-based index of a step by ID.
func (a *Agent) getStepIndex(plan *planner.Plan, stepID string) int {
	for i, step := range plan.Steps {
		if step.ID == stepID {
			return i
		}
	}
	return 0
}

// executeStep runs a mini ReAct loop for a single plan step.
func (a *Agent) executeStep(
	ctx context.Context,
	sess *session.Session,
	step *planner.Step,
	toolDefs []tools.ToolDefinition,
	model string,
	maxIter int,
	sessionID string,
	emit func(bus.OutboundMessage),
) string {
	stepPrompt := fmt.Sprintf("Execute this step: %s\nExpected outcome: %s", step.Description, step.ExpectedOut)
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: stepPrompt})

	var lastContent string
	for i := 0; i < maxIter; i++ {
		contextMsgs := a.contextManager.BuildContext(sess, toolDefs, "")
		totalCtxTokens := 0
		for _, msg := range contextMsgs {
			totalCtxTokens += len(msg.Content)
		}
		AgentContextTokens.Set(float64(totalCtxTokens))

		provider, modelName, _, err := a.providerFactory.GetProviderForModelWithFallback(
			model, a.config.Agents.Defaults.FallbackModels,
		)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}

		AgentLLMCalls.Inc()
		llmStart := time.Now()
		resp, err := provider.Chat(ctx, contextMsgs, toolDefs, modelName, nil)
		AgentLLMLatency.Observe(time.Since(llmStart).Seconds())

		if err != nil {
			return fmt.Sprintf("LLM error: %v", err)
		}

		AgentLLMTokens.Add(int64(resp.Usage.TotalTokens))

		if len(resp.ToolCalls) == 0 {
			lastContent = resp.Content
			sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: resp.Content})
			break
		}

		sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: resp.Content, ToolCalls: resp.ToolCalls})

		// Execute tools
		for _, tc := range resp.ToolCalls {
			AgentToolCalls.Inc()
			toolStart := time.Now()

			// Emit tool call progress
			if emit != nil {
				emit(bus.OutboundMessage{
					SessionID:    sessionID,
					Content:      tc.Name,
					Role:         bus.RoleProgress,
					ProgressType: bus.ProgressToolCall,
					ToolName:     tc.Name,
				})
			}

			tool, found := a.toolRegistry.Get(tc.Name)
			if !found {
				continue
			}
			result, err := tool.Execute(ctx, tc.Arguments)
			AgentToolLatency.Observe(time.Since(toolStart).Seconds())
			if err != nil {
				AgentToolErrors.Inc()
			}

			toolResult := ""
			if result != nil {
				toolResult = result.ForLLM
				// Emit tool result progress (truncated for display)
				if emit != nil && result.ForUser != "" {
					summary := result.ForUser
					if len(summary) > 100 {
						summary = summary[:97] + "..."
					}
					emit(bus.OutboundMessage{
						SessionID:    sessionID,
						Content:      summary,
						Role:         bus.RoleProgress,
						ProgressType: bus.ProgressToolResult,
						ToolName:     tc.Name,
					})
				}
			}
			sess.AddMessage(providers.Message{Role: providers.RoleTool, Content: toolResult, ToolCallID: tc.ID})
			lastContent = toolResult
		}
	}

	return lastContent
}

// buildToolErrorMessage creates an informative error message for the LLM feedback loop.
// The message includes: tool name, error details, arguments, and suggested actions.
func (a *Agent) buildToolErrorMessage(tc providers.ToolCall, err error) string {
	var sb strings.Builder

	// Header with tool name
	sb.WriteString(fmt.Sprintf("[Tool Error] %s failed\n", tc.Name))

	// Error details
	errStr := err.Error()
	sb.WriteString(fmt.Sprintf("Error: %s\n", errStr))

	// Arguments that were passed (helps LLM understand what went wrong)
	if tc.Arguments != nil {
		argsJSON, marshalErr := json.Marshal(tc.Arguments)
		if marshalErr == nil {
			sb.WriteString(fmt.Sprintf("Arguments: %s\n", string(argsJSON)))
		}
	}

	// Suggested actions based on error type
	sb.WriteString("\nSuggested actions:\n")

	errLower := strings.ToLower(errStr)
	switch {
	case strings.Contains(errLower, "not found"):
		// Tool not found - list available tools
		var available []string
		for _, t := range a.toolRegistry.ListTools() {
			available = append(available, t.Name())
		}
		if len(available) > 0 {
			sb.WriteString(fmt.Sprintf("- Available tools: %s\n", strings.Join(available, ", ")))
		}
		sb.WriteString("- Try a different tool that can accomplish the same goal\n")

	case strings.Contains(errLower, "connection") || strings.Contains(errLower, "timeout"):
		// Connection/timeout error - suggest retry or different approach
		sb.WriteString("- This may be a temporary issue, try again\n")
		sb.WriteString("- Or try a different approach to accomplish the goal\n")

	case strings.Contains(errLower, "permission") || strings.Contains(errLower, "denied"):
		// Permission error - suggest different tool
		sb.WriteString("- You don't have permission for this operation\n")
		sb.WriteString("- Try a different tool or ask the user for help\n")

	case strings.Contains(errLower, "sql") || strings.Contains(errLower, "query"):
		// SQL error - suggest checking syntax
		sb.WriteString("- Check the SQL syntax\n")
		sb.WriteString("- Make sure the table/column names are correct\n")

	default:
		// Generic error - suggest retry or alternative
		sb.WriteString("- Try again with different arguments\n")
		sb.WriteString("- Or try a different approach\n")
	}

	return sb.String()
}
