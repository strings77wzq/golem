package agent

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/core/usage"
)

// executeTools runs tool calls in parallel and returns results and errors.
func (a *Agent) executeTools(ctx context.Context, resp *providers.LLMResponse) ([]*tools.ToolResult, []error) {
	results := make([]*tools.ToolResult, len(resp.ToolCalls))
	errors := make([]error, len(resp.ToolCalls))

	var wg errgroup.Group
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

	wg.Wait()
	return results, errors
}

// runToolExecution orchestrates the full tool execution pipeline:
// PreToolShell hooks → parallel execution → result processing → PostToolShell hooks.
func (a *Agent) runToolExecution(ctx context.Context, resp *providers.LLMResponse, sess *session.Session, msg bus.InboundMessage, emit func(bus.OutboundMessage)) {
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

	// Filter out blocked tools and execute remaining in parallel
	var filteredCalls []providers.ToolCall
	for _, tc := range resp.ToolCalls {
		if !blockedTools[tc.ID] {
			filteredCalls = append(filteredCalls, tc)
		}
	}

	var results []*tools.ToolResult
	if len(filteredCalls) > 0 {
		filteredResp := &providers.LLMResponse{ToolCalls: filteredCalls}
		var errors []error
		results, errors = a.executeTools(ctx, filteredResp)
		a.processToolResults(sess, filteredResp, results, errors, msg.SessionID, emit)
	}

	// Post-tool shell hooks
	if a.hooks != nil && a.hooks.PostToolShell != nil {
		for _, tc := range resp.ToolCalls {
			if blockedTools[tc.ID] {
				continue
			}
			for j, fc := range filteredCalls {
				if fc.ID == tc.ID && results != nil && j < len(results) {
					if results[j] != nil {
						a.hooks.PostToolShell.Execute(&HookInput{
							SessionID:  msg.SessionID,
							ToolName:   tc.Name,
							ToolInput:  tc.Arguments,
							ToolOutput: results[j].ForLLM,
						})
					}
					break
				}
			}
		}
	}
}

// processToolResults processes tool execution results and appends them to the session.
func (a *Agent) processToolResults(sess *session.Session, resp *providers.LLMResponse, results []*tools.ToolResult, errors []error, sessionID string, emit func(bus.OutboundMessage)) {
	for i, tc := range resp.ToolCalls {
		if errors[i] != nil {
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

		if result.ForUser != "" && !result.Silent && emit != nil {
			emit(bus.OutboundMessage{
				SessionID: sessionID,
				Content:   result.ForUser,
				Role:      bus.RoleTool,
				Done:      false,
			})
		}
	}
}

// saveAndEmitFinal saves the session and emits the final response.
func (a *Agent) saveAndEmitFinal(sess *session.Session, resp *providers.LLMResponse, streamed bool, modelName string, msg bus.InboundMessage, emit func(bus.OutboundMessage)) *bus.TokenUsage {
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

	if a.tracker != nil {
		a.tracker.Record(msg.SessionID, modelName, usage.TokenUsage{
			PromptTokens:     tokenUsage.PromptTokens,
			CompletionTokens: tokenUsage.CompletionTokens,
			TotalTokens:      tokenUsage.TotalTokens,
		})
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

	return tokenUsage
}
