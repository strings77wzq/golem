package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

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
	AgentMessagesTotal.Inc()
	startTime := time.Now()
	sess, found := a.sessionStore.Get(msg.SessionID)
	if !found {
		sess = session.NewSession(msg.SessionID)
		if err := a.sessionStore.Save(sess); err != nil {
			a.logger.Error("failed to save new session", err)
			a.emitError(msg.SessionID, "failed to create session", emit)
			return "", nil, fmt.Errorf("failed to create session: %w", err)
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

	model := a.config.Agents.Defaults.ModelName
	toolDefs := a.toolRegistry.ListDefinitions()

	// Hook: before message processing
	if a.hooks != nil && a.hooks.BeforeMessage != nil {
		if err := a.hooks.BeforeMessage(ctx, msg.SessionID, msg.Content); err != nil {
			a.logger.Error("before_message hook failed", err)
		}
	}

	// Planning mode: decompose complex tasks
	if a.planEnabled && a.planner != nil && a.isComplexTask(msg.Content) {
		return a.processWithPlan(ctx, msg, sess, model, toolDefs, streamFinal, onToken, emit)
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

	// Guard against infinite loops if the LLM repeatedly calls tools without converging.
	// Default maxToolIterations=25 prevents runaway agents while allowing complex tasks.
	for i := 0; i < a.maxToolIterations; i++ {
		contextMsgs := a.contextManager.BuildContext(sess, toolDefs, "")

		// Observability: track context token usage
		totalCtxTokens := 0
		for _, msg := range contextMsgs {
			totalCtxTokens += len(msg.Content)
		}
		AgentContextTokens.Set(float64(totalCtxTokens))

		// Use router if set, otherwise fall back to factory
		var resp *providers.LLMResponse
		var err error
		var provider providers.LLMProvider
		var modelName string

		if a.router != nil {
			resp, err = a.router.Chat(ctx, model, contextMsgs, toolDefs, &providers.ChatOptions{})
			if err != nil {
				a.logger.Error("router chat failed", err)
				a.emitError(msg.SessionID, fmt.Sprintf("router error: %v", err), emit)
				return "", nil, err
			}
			if resp == nil {
				return "", nil, fmt.Errorf("router returned nil response for model %q", model)
			}
			provider = nil // not needed for router path
			modelName = model
		} else {
			var usedModel string
			provider, modelName, usedModel, err = a.providerFactory.GetProviderForModelWithFallback(
				model, a.config.Agents.Defaults.FallbackModels,
			)
			if err != nil {
				a.logger.Error("failed to get provider for model", err)
				a.emitError(msg.SessionID, fmt.Sprintf("failed to get provider: %v", err), emit)
				return "", nil, err
			}
			_ = usedModel // available for logging if needed
		}

		// Hook: before LLM call
		if a.hooks != nil && a.hooks.BeforeLLM != nil {
			if err := a.hooks.BeforeLLM(ctx, contextMsgs); err != nil {
				a.logger.Error("before_llm hook failed", err)
			}
		}

		// Observability: record LLM call
		AgentLLMCalls.Inc()
		llmStart := time.Now()

		var streamed bool
		if a.router == nil {
			// Standard path: resolve provider from factory and invoke
			resp, streamed, err = a.invokeProvider(ctx, provider, contextMsgs, toolDefs, modelName, streamFinal, onToken, msg.SessionID, emit)
		}
		// When router is set, resp is already populated from router.Chat() above
		llmDuration := time.Since(llmStart)
		AgentLLMLatency.Observe(llmDuration.Seconds())

		if err != nil {
			a.logger.Error("LLM chat failed", err)
			AgentLLMErrors.Inc()
			if a.hooks != nil && a.hooks.OnError != nil {
				a.hooks.OnError(ctx, err)
			}
			a.emitError(msg.SessionID, fmt.Sprintf("LLM error: %v", err), emit)
			return "", nil, err
		}

		// Observability: record token usage
		AgentLLMTokens.Add(int64(resp.Usage.TotalTokens))

		// Hook: after LLM call
		if a.hooks != nil && a.hooks.AfterLLM != nil {
			if err := a.hooks.AfterLLM(ctx, resp); err != nil {
				a.logger.Error("after_llm hook failed", err)
			}
		}

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

			// Record usage for cost tracking
			if a.tracker != nil && tokenUsage != nil {
				a.tracker.Record(msg.SessionID, modelName, usage.TokenUsage{
					PromptTokens:     tokenUsage.PromptTokens,
					CompletionTokens: tokenUsage.CompletionTokens,
					TotalTokens:      tokenUsage.TotalTokens,
				})
				// Track cost in metric (scaled by 10000 for integer storage)
				pricing := usage.GetPricing(modelName)
				costCents := float64(tokenUsage.PromptTokens)*pricing.InputPerToken +
					float64(tokenUsage.CompletionTokens)*pricing.OutputPerToken
				AgentLLMCostUSD.Add(int64(costCents * 10000))
			}

			if emit != nil {
				finalContent := resp.Content
				// When streaming, tokens already emitted via onToken callback.
				// Final emit only carries Done flag + Usage for completion signal.
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

			// Hook: after message processing (success)
			if a.hooks != nil && a.hooks.AfterMessage != nil {
				if err := a.hooks.AfterMessage(ctx, msg.SessionID, resp.Content); err != nil {
					a.logger.Error("after_message hook failed", err)
				}
			}

			// Observability: record plan duration
			AgentPlanDuration.Observe(time.Since(startTime).Seconds())

			return resp.Content, tokenUsage, nil
		}

		sess.AddMessage(providers.Message{
			Role:      providers.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute tool calls in parallel for better performance
		var wg errgroup.Group
		results := make([]*tools.ToolResult, len(resp.ToolCalls))
		errors := make([]error, len(resp.ToolCalls))

		// Pre-tool shell hooks: run sequentially before parallel execution
		blockedTools := make(map[string]bool)
		if a.hooks != nil && a.hooks.PreToolShell != nil {
			for _, tc := range resp.ToolCalls {
				output, hookErr := a.hooks.PreToolShell.Execute(&HookInput{
					SessionID: msg.SessionID,
					ToolName:  tc.Name,
					ToolInput: tc.Arguments,
				})
				if hookErr != nil || !output.Allowed {
					reason := "hook error"
					if output != nil {
						reason = output.Reason
					}
					blockedTools[tc.ID] = true
					sess.AddMessage(providers.Message{
						Role:       providers.RoleTool,
						Content:    fmt.Sprintf("Tool blocked by policy: %s", reason),
						ToolCallID: tc.ID,
					})
					if emit != nil {
						emit(bus.OutboundMessage{
							SessionID: msg.SessionID,
							Content:   fmt.Sprintf("Tool %s blocked: %s", tc.Name, reason),
							Role:      bus.RoleTool,
							Done:      false,
						})
					}
				}
			}
		}

		for i, tc := range resp.ToolCalls {
			i, tc := i, tc // capture loop variables
			wg.Go(func() error {
				// Skip blocked tools
				if blockedTools[tc.ID] {
					return nil
				}
				// Observability: record tool call
				AgentToolCalls.Inc()
				toolStart := time.Now()

				tool, found := a.toolRegistry.Get(tc.Name)
				if !found {
					errors[i] = fmt.Errorf("tool %q not found", tc.Name)
					AgentToolErrors.Inc()
					return nil
				}

				result, err := tool.Execute(ctx, tc.Arguments)
				toolDuration := time.Since(toolStart)
				AgentToolLatency.Observe(toolDuration.Seconds())

				results[i] = result
				errors[i] = err
				if err != nil {
					AgentToolErrors.Inc()
				}
				return nil
			})
		}

		// Wait for all tool executions to complete
		if err := wg.Wait(); err != nil {
			a.logger.Error("tool execution failed", err)
		}

		// Process results in order (but execution was parallel)
		for i, tc := range resp.ToolCalls {
			if errors[i] != nil {
				// Build informative error message for LLM feedback loop
				errMsg := a.buildToolErrorMessage(tc, errors[i])

				sess.AddMessage(providers.Message{
					Role:       providers.RoleTool,
					Content:    errMsg,
					ToolCallID: tc.ID,
				})
				continue
			}

			result := results[i]
			if result == nil {
				continue
			}

			sess.AddMessage(providers.Message{
				Role:       providers.RoleTool,
				Content:    result.ForLLM,
				ToolCallID: tc.ID,
			})

			// Post-tool shell hook
			if a.hooks != nil && a.hooks.PostToolShell != nil && !blockedTools[tc.ID] {
				a.hooks.PostToolShell.Execute(&HookInput{
					SessionID:  msg.SessionID,
					ToolName:   tc.Name,
					ToolInput:  tc.Arguments,
					ToolOutput: result.ForLLM,
				})
			}

			if result.ForUser != "" && !result.Silent {
				// ForUser: user-visible feedback (e.g., "Searched for X", "Downloaded file Y")
				// Silent: suppress display for noisy tools that produce too much output
				if emit != nil {
					emit(bus.OutboundMessage{
						SessionID: msg.SessionID,
						Content:   result.ForUser,
						Role:      bus.RoleTool,
						Done:      false,
					})
				}
			}
		}

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

	// Hook: after message processing
	if a.hooks != nil && a.hooks.AfterMessage != nil {
		if err := a.hooks.AfterMessage(ctx, msg.SessionID, "max tool iterations reached"); err != nil {
			a.logger.Error("after_message hook failed", err)
		}
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
		return a.processMessageFallback(ctx, msg, sess, model, toolDefs, streamFinal, onToken, emit)
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

// processMessageFallback is the regular ReAct loop (used when planning fails).
// It delegates to shared helpers for tool execution and result processing.
func (a *Agent) processMessageFallback(
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

		provider, modelName, _, err := a.providerFactory.GetProviderForModelWithFallback(
			model, a.config.Agents.Defaults.FallbackModels,
		)
		if err != nil {
			a.emitError(msg.SessionID, fmt.Sprintf("failed to get provider: %v", err), emit)
			return "", nil, err
		}

		AgentLLMCalls.Inc()
		llmStart := time.Now()
		resp, streamed, err := a.invokeProvider(ctx, provider, contextMsgs, toolDefs, modelName, streamFinal, onToken, msg.SessionID, emit)
		AgentLLMLatency.Observe(time.Since(llmStart).Seconds())

		if err != nil {
			AgentLLMErrors.Inc()
			a.emitError(msg.SessionID, fmt.Sprintf("LLM error: %v", err), emit)
			return "", nil, err
		}

		AgentLLMTokens.Add(int64(resp.Usage.TotalTokens))

		// No tool calls — final response
		if len(resp.ToolCalls) == 0 {
			tokenUsage := a.saveAndEmitFinal(sess, resp, streamed, modelName, msg, emit)
			return resp.Content, tokenUsage, nil
		}

		// Tool calls — execute and continue loop
		sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: resp.Content, ToolCalls: resp.ToolCalls})
		results, errors := a.executeTools(ctx, resp)
		a.processToolResults(sess, resp, results, errors, msg.SessionID, emit)

		if err := a.sessionStore.Save(sess); err != nil {
			a.logger.Error("failed to save session", err)
		}
	}

	return "max tool iterations reached", nil, nil
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
