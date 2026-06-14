package context

import (
	"fmt"
	"strings"

	"github.com/strings77wzq/golem/core/providers"
)

// Compressor handles message compression and truncation to fit within
// a token budget. It uses a three-stage strategy with tool chain preservation:
//
//  1. Keep recent messages untouched (preserves conversation continuity)
//  2. Truncate oversized tool outputs in middle messages
//  3. Drop oldest messages if still over budget, preserving tool chains atomically
type Compressor struct {
	MaxToolOutput     int
	KeepRecent        int
	TruncateThreshold int
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
	return &Compressor{
		MaxToolOutput:     maxToolOutput,
		KeepRecent:        4,
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

	// Stage 3: If still over budget, drop oldest messages preserving tool chains
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
	maxChars := maxTokens * 4
	if len(content) <= maxChars {
		return content
	}

	truncated := content[:maxChars]
	if idx := strings.LastIndex(truncated, "\n"); idx > maxChars/2 {
		truncated = truncated[:idx]
	}

	return truncated + fmt.Sprintf("\n... [truncated, %d tokens remaining]", maxTokens)
}

// toolBatch represents an atomic unit: one assistant message with tool_calls
// and all its corresponding tool result messages.
type toolBatch struct {
	assistantIdx int
	toolIndices  []int
	totalTokens  int
}

// identifyToolBatches scans messages and groups them into atomic batches.
// A batch is: assistant(tool_calls=[A,B]) + tool(A) + tool(B)
func (c *Compressor) identifyToolBatches(messages []providers.Message, tokenFunc func(providers.Message) int) []toolBatch {
	var batches []toolBatch

	// Build a map from ToolCallID to its index for fast lookup
	toolCallIDs := make(map[string]int) // ToolCallID → message index
	for i, msg := range messages {
		if msg.Role == providers.RoleTool && msg.ToolCallID != "" {
			toolCallIDs[msg.ToolCallID] = i
		}
	}

	for i, msg := range messages {
		if msg.Role != providers.RoleAssistant || len(msg.ToolCalls) == 0 {
			continue
		}

		batch := toolBatch{assistantIdx: i, totalTokens: tokenFunc(msg)}
		for _, tc := range msg.ToolCalls {
			if idx, ok := toolCallIDs[tc.ID]; ok {
				batch.toolIndices = append(batch.toolIndices, idx)
				batch.totalTokens += tokenFunc(messages[idx])
			}
		}

		if len(batch.toolIndices) > 0 {
			batches = append(batches, batch)
		}
	}

	return batches
}

// isInBatch checks if a message index belongs to any identified batch.
func isInBatch(idx int, batches []toolBatch) (batchIdx int, ok bool) {
	for bi, batch := range batches {
		if batch.assistantIdx == idx {
			return bi, true
		}
		for _, ti := range batch.toolIndices {
			if ti == idx {
				return bi, true
			}
		}
	}
	return -1, false
}

// fitToBudget drops oldest messages until everything fits, preserving tool chains.
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

	// Find where recent messages start in the compressed array
	recentStart := len(compressed)
	if len(recent) > 0 && len(compressed) > 0 {
		// Use the last message's content to find the boundary
		lastRecent := recent[0]
		for i := len(compressed) - 1; i >= 0; i-- {
			if compressed[i].Content == lastRecent.Content &&
				compressed[i].Role == lastRecent.Role {
				recentStart = i
				break
			}
		}
	}

	// Identify tool batches in old messages
	batches := c.identifyToolBatches(compressed, tokenFunc)

	// Track which batches are already included in result
	includedBatches := make(map[int]bool)
	for bi, batch := range batches {
		for _, idx := range batch.toolIndices {
			if idx >= recentStart {
				includedBatches[bi] = true
				break
			}
		}
	}

	// Add compressed messages from newest to oldest
	for i := len(compressed) - 1; i >= 0; i-- {
		if i >= recentStart {
			continue
		}

		msg := compressed[i]

		// Check if this message is part of a batch
		batchIdx, inBatch := isInBatch(i, batches)
		if inBatch {
			batch := batches[batchIdx]

			// If batch is already included, skip
			if includedBatches[batchIdx] {
				continue
			}

			// Try to fit the entire batch
			if used+batch.totalTokens <= budget {
				// Fit the batch in order: assistant first, then tools
				result = append([]providers.Message{compressed[batch.assistantIdx]}, result...)
				used += tokenFunc(compressed[batch.assistantIdx])

				for _, ti := range batch.toolIndices {
					result = append([]providers.Message{compressed[ti]}, result...)
					used += tokenFunc(compressed[ti])
				}
				includedBatches[batchIdx] = true
			} else {
				// Can't fit batch - drop entire batch
				// Add a summary of the assistant message instead
				summary := c.makeSummary(compressed[batch.assistantIdx])
				summaryTokens := tokenFunc(providers.Message{Role: providers.RoleAssistant, Content: summary})
				if used+summaryTokens <= budget {
					result = append([]providers.Message{{Role: providers.RoleAssistant, Content: summary}}, result...)
					used += summaryTokens
				}
				includedBatches[batchIdx] = true
			}
			continue
		}

		// Not part of a batch - handle normally
		msgTokens := tokenFunc(msg)
		if used+msgTokens > budget {
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
