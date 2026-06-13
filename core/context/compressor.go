package context

import (
	"fmt"
	"strings"

	"github.com/strings77wzq/golem/core/providers"
)

// Compressor handles message compression and truncation to fit within
// a token budget. It uses a three-stage strategy:
//
//  1. Keep recent messages untouched (preserves conversation continuity)
//  2. Truncate oversized tool outputs in middle messages
//  3. Drop oldest messages if still over budget
type Compressor struct {
	MaxToolOutput     int  // max tokens per tool output (default 2000)
	KeepRecent        int  // number of recent messages to never compress (default 4)
	TruncateThreshold int  // truncate tool outputs larger than this (default 1000)
}

// NewCompressor creates a compressor with default settings.
func NewCompressor() *Compressor {
	return &Compressor{
		MaxToolOutput:     2000,
		KeepRecent:        4,
		TruncateThreshold: 1000,
	}
}

// NewCompressorWithConfig creates a compressor with custom settings.
func NewCompressorWithConfig(maxToolOutput, truncateThreshold int) *Compressor {
	keepRecent := 4
	if truncateThreshold > 0 {
		keepRecent = 4
	}
	return &Compressor{
		MaxToolOutput:     maxToolOutput,
		KeepRecent:        keepRecent,
		TruncateThreshold: truncateThreshold,
	}
}

// Compress fits messages within the token budget using progressive strategies.
func (c *Compressor) Compress(
	messages []providers.Message,
	budget int,
	tokenFunc func(providers.Message) int,
) []providers.Message {
	if len(messages) == 0 {
		return nil
	}

	// Stage 0: Check if everything fits
	total := 0
	for _, msg := range messages {
		total += tokenFunc(msg)
	}
	if total <= budget {
		return messages
	}

	// Stage 1: Keep recent messages, compress the rest
	keepCount := c.KeepRecent
	if keepCount > len(messages) {
		keepCount = len(messages)
	}

	recent := messages[len(messages)-keepCount:]
	old := messages[:len(messages)-keepCount]

	// Stage 2: Truncate tool outputs in old messages
	compressed := c.truncateToolOutputs(old, tokenFunc)

	// Stage 3: If still over budget, drop oldest messages
	result := c.fitToBudget(compressed, recent, budget, tokenFunc)

	return result
}

// truncateToolOutputs shortens oversized tool outputs in messages.
func (c *Compressor) truncateToolOutputs(
	messages []providers.Message,
	tokenFunc func(providers.Message) int,
) []providers.Message {
	result := make([]providers.Message, len(messages))
	for i, msg := range messages {
		if msg.Role == providers.RoleTool && tokenFunc(msg) > c.TruncateThreshold {
			msg.Content = c.truncateContent(msg.Content, c.MaxToolOutput)
		}
		result[i] = msg
	}
	return result
}

// truncateContent shortens content to fit within token limit.
func (c *Compressor) truncateContent(content string, maxTokens int) string {
	// Rough estimate: 4 chars per token for ASCII
	maxChars := maxTokens * 4
	if len(content) <= maxChars {
		return content
	}

	truncated := content[:maxChars]
	// Find last newline to avoid cutting mid-line
	if idx := strings.LastIndex(truncated, "\n"); idx > maxChars/2 {
		truncated = truncated[:idx]
	}

	return truncated + fmt.Sprintf("\n... [truncated, %d tokens remaining]", maxTokens)
}

// fitToBudget drops oldest messages until everything fits.
// Preserves tool message chains: if a tool message is kept, its preceding
// assistant message with tool_calls must also be kept.
func (c *Compressor) fitToBudget(
	compressed []providers.Message,
	recent []providers.Message,
	budget int,
	tokenFunc func(providers.Message) int,
) []providers.Message {
	// Start with recent messages
	result := make([]providers.Message, 0, len(recent))
	result = append(result, recent...)
	used := 0
	for _, msg := range recent {
		used += tokenFunc(msg)
	}

	// Build set of message indices already in result
	recentStart := len(compressed)
	if len(recent) > 0 {
		// Find where recent messages came from in compressed
		for i := len(compressed) - 1; i >= 0; i-- {
			if len(recent) > 0 && compressed[i].Content == recent[0].Content {
				recentStart = i
				break
			}
		}
	}

	// Add compressed messages from newest to oldest
	for i := len(compressed) - 1; i >= 0; i-- {
		if i >= recentStart {
			continue // already in result
		}

		msg := compressed[i]

		// If this is a tool message, check if its preceding assistant message is included
		if msg.Role == providers.RoleTool {
			// Find the preceding assistant message with tool_calls
			hasPredecessor := false
			for j := i - 1; j >= 0; j-- {
				if compressed[j].Role == providers.RoleAssistant && len(compressed[j].ToolCalls) > 0 {
					// Check if this assistant message is already in result
					for _, r := range result {
						if r.Content == compressed[j].Content && len(r.ToolCalls) > 0 {
							hasPredecessor = true
							break
						}
					}
					break
				}
			}
			if !hasPredecessor {
				// Skip this tool message to avoid breaking the chain
				continue
			}
		}

		msgTokens := tokenFunc(msg)
		if used+msgTokens > budget {
			// Can't fit this message, try to fit a summary
			summary := c.makeSummary(msg)
			summaryTokens := tokenFunc(providers.Message{Role: providers.RoleAssistant, Content: summary})
			if used+summaryTokens <= budget {
				result = append([]providers.Message{{Role: providers.RoleAssistant, Content: summary}}, result...)
				used += summaryTokens
			}
			continue
		}
		result = append([]providers.Message{msg}, result...)
		used += msgTokens
	}

	return result
}

// makeSummary creates a brief summary of a message for context preservation.
func (c *Compressor) makeSummary(msg providers.Message) string {
	content := msg.Content
	if len(content) > 200 {
		content = content[:200] + "..."
	}
	role := string(msg.Role)
	return fmt.Sprintf("[%s: %s]", role, content)
}
