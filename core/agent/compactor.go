package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
)

// Compactor uses an LLM to summarize old messages, replacing string truncation.
type Compactor struct {
	provider providers.LLMProvider
	model    string
}

// NewCompactor creates a new LLM-driven session compactor.
func NewCompactor(provider providers.LLMProvider, model string) *Compactor {
	return &Compactor{
		provider: provider,
		model:    model,
	}
}

// Compact summarizes old messages in the session, keeping recent messages intact.
// Returns a description of what was done.
func (c *Compactor) Compact(ctx context.Context, sess *session.Session, tokenBudget int) (string, error) {
	messages := sess.GetMessages()
	if len(messages) <= 2 {
		return "session is already minimal, nothing to compact", nil
	}

	// Estimate current token count (rough: 1 token ≈ 4 chars)
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content)
	}
	estimatedTokens := totalChars / 4

	// Don't compact if under budget
	if estimatedTokens < tokenBudget {
		return "session is already minimal, nothing to compact", nil
	}

	// Split into old (to summarize) and recent (to keep)
	var systemMsg providers.Message
	var old []providers.Message
	var recent []providers.Message

	if messages[0].Role == providers.RoleSystem {
		systemMsg = messages[0]
		rest := messages[1:]
		keepRecent := 4
		if keepRecent >= len(rest) {
			keepRecent = len(rest) / 2 // keep half as recent, summarize other half
			if keepRecent < 1 {
				keepRecent = 1
			}
		}
		old = rest[:len(rest)-keepRecent]
		recent = rest[len(rest)-keepRecent:]
	} else {
		keepRecent := 4
		if keepRecent >= len(messages) {
			keepRecent = len(messages) / 2
			if keepRecent < 1 {
				keepRecent = 1
			}
		}
		old = messages[:len(messages)-keepRecent]
		recent = messages[len(messages)-keepRecent:]
	}

	if len(old) == 0 {
		return "session is already minimal, nothing to compact", nil
	}

	// Use LLM to summarize old messages (if provider available)
	var summary string
	if c.provider != nil {
		var err error
		summary, err = c.summarize(ctx, old)
		if err != nil {
			// Fallback to simple truncation if LLM fails
			summary = c.simpleSummary(old)
		}
	} else {
		summary = c.simpleSummary(old)
	}

	// Rebuild session
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

	return fmt.Sprintf("compacted: summarized %d old messages, kept %d recent", len(old), len(recent)), nil
}

// summarize calls the LLM to create a concise summary.
func (c *Compactor) summarize(ctx context.Context, messages []providers.Message) (string, error) {
	prompt := c.buildSummaryPrompt(messages)

	resp, err := c.provider.Chat(ctx, []providers.Message{
		{Role: providers.RoleUser, Content: prompt},
	}, nil, c.model, nil)
	if err != nil {
		return "", fmt.Errorf("LLM summarization failed: %w", err)
	}

	return resp.Content, nil
}

// buildSummaryPrompt creates the summarization prompt.
func (c *Compactor) buildSummaryPrompt(messages []providers.Message) string {
	var sb strings.Builder
	sb.WriteString("Summarize the following conversation concisely, preserving key facts, decisions, and context:\n\n")

	for _, msg := range messages {
		switch msg.Role {
		case providers.RoleUser:
			sb.WriteString(fmt.Sprintf("User: %s\n", msg.Content))
		case providers.RoleAssistant:
			if msg.Content != "" {
				sb.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
			}
		case providers.RoleTool:
			content := msg.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("Tool: %s\n", content))
		}
	}

	sb.WriteString("\nProvide a concise summary that captures the essential information.")
	return sb.String()
}

// simpleSummary creates a simple truncation-based summary as fallback.
func (c *Compactor) simpleSummary(messages []providers.Message) string {
	var sb strings.Builder
	sb.WriteString("Key points from the conversation:\n")

	for _, msg := range messages {
		switch msg.Role {
		case providers.RoleUser:
			content := msg.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf("- User asked: %s\n", content))
		case providers.RoleAssistant:
			if msg.Content != "" {
				content := msg.Content
				if len(content) > 100 {
					content = content[:100] + "..."
				}
				sb.WriteString(fmt.Sprintf("- Assistant: %s\n", content))
			}
			if len(msg.ToolCalls) > 0 {
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
