package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/planner"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/core/usage"
	"golang.org/x/sync/errgroup"
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
	resp, _, err := a.processMessage(ctx, bus.InboundMessage{
		SessionID: sessionID,
		Content:   message,
		Role:      bus.RoleUser,
	}, false, nil, nil)
	return resp, err
}

func (a *Agent) HandleMessageStream(ctx context.Context, sessionID string, message string, tokens chan<- string) error {
	defer close(tokens)
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

	// Guard against infinite loops if the LLM repeatedly calls tools without converging.
	// Default maxToolIterations=25 prevents runaway agents while allowing complex tasks.
	for i := 0; i < a.maxToolIterations; i++ {
		contextMsgs := a.contextManager.BuildContext(sess, toolDefs, "")

		provider, modelName, err := a.providerFactory.GetProviderForModel(model)
		if err != nil {
			a.logger.Error("failed to get provider for model", err)
			a.emitError(msg.SessionID, fmt.Sprintf("failed to get provider: %v", err), emit)
			return "", nil, err
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

		resp, streamed, err := a.invokeProvider(ctx, provider, contextMsgs, toolDefs, modelName, streamFinal, onToken, msg.SessionID, emit)
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

		for i, tc := range resp.ToolCalls {
			i, tc := i, tc // capture loop variables
			wg.Go(func() error {
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
				// Build informative error message for LLM
				errMsg := fmt.Sprintf("Tool %q failed: %v", tc.Name, errors[i])
				
				// Add available tools hint if tool not found
				if strings.Contains(errors[i].Error(), "not found") {
					var available []string
					for _, t := range a.toolRegistry.ListTools() {
						available = append(available, t.Name())
					}
					if len(available) > 0 {
						errMsg += fmt.Sprintf("\nAvailable tools: %s", strings.Join(available, ", "))
					}
					errMsg += "\nPlease try a different tool."
				}
				
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
	// Streaming is disabled when tools are present because mid-stream tool call arguments
	// require buffering the entire stream to parse JSON, eliminating any latency benefit.
	// The streaming contract only applies to final text responses without tool calls.
	canStream := ok && streamFinal && len(toolDefs) == 0
	if !canStream {
		resp, err := provider.Chat(ctx, messages, toolDefs, modelName, nil)
		return resp, false, err
	}
	resp, err := sp.ChatStream(ctx, messages, toolDefs, modelName, nil, a.wrapTokenEmitter(sessionID, emit, onToken))
	return resp, err == nil, err
}

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

// isComplexTask determines if a message should use planning mode.
func (a *Agent) isComplexTask(message string) bool {
	complexKeywords := []string{"plan", "deploy", "migrate", "setup", "configure",
		"build and", "first", "then", "finally", "step by step",
		"create a", "set up", "install and"}
	lower := strings.ToLower(message)
	for _, kw := range complexKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return len(message) > 150
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
		provider, modelName, err := a.providerFactory.GetProviderForModel(model)
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
	// This is the original ReAct loop logic
	for i := 0; i < a.maxToolIterations; i++ {
		contextMsgs := a.contextManager.BuildContext(sess, toolDefs, "")

		provider, modelName, err := a.providerFactory.GetProviderForModel(model)
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

		if len(resp.ToolCalls) == 0 {
			sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: resp.Content})
			if err := a.sessionStore.Save(sess); err != nil {
				a.logger.Error("failed to save session", err)
			}

			tokenUsage := &bus.TokenUsage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
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

		sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: resp.Content, ToolCalls: resp.ToolCalls})

		var wg errgroup.Group
		results := make([]*tools.ToolResult, len(resp.ToolCalls))
		errors := make([]error, len(resp.ToolCalls))

		for i, tc := range resp.ToolCalls {
			i, tc := i, tc
			wg.Go(func() error {
				AgentToolCalls.Inc()
				toolStart := time.Now()
				tool, found := a.toolRegistry.Get(tc.Name)
				if !found {
					errors[i] = fmt.Errorf("tool %q not found", tc.Name)
					AgentToolErrors.Inc()
					return nil
				}
				result, err := tool.Execute(ctx, tc.Arguments)
				AgentToolLatency.Observe(time.Since(toolStart).Seconds())
				results[i] = result
				errors[i] = err
				if err != nil {
					AgentToolErrors.Inc()
				}
				return nil
			})
		}

		if err := wg.Wait(); err != nil {
			a.logger.Error("tool execution failed", err)
		}

		for i, tc := range resp.ToolCalls {
			if errors[i] != nil {
				sess.AddMessage(providers.Message{Role: providers.RoleTool, Content: fmt.Sprintf("tool error: %v", errors[i]), ToolCallID: tc.ID})
				continue
			}
			result := results[i]
			if result == nil {
				continue
			}
			sess.AddMessage(providers.Message{Role: providers.RoleTool, Content: result.ForLLM, ToolCallID: tc.ID})
			if result.ForUser != "" && !result.Silent && emit != nil {
				emit(bus.OutboundMessage{SessionID: msg.SessionID, Content: result.ForUser, Role: bus.RoleTool, Done: false})
			}
		}

		if err := a.sessionStore.Save(sess); err != nil {
			a.logger.Error("failed to save session", err)
		}
	}

	return "max tool iterations reached", nil, nil
}
