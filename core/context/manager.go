// Package context manages the LLM context window: token budget allocation,
// dynamic system prompt assembly, message compression, and tool output
// truncation. It replaces the simple truncation in HistoryManager with
// intelligent context management inspired by Claude Code's harness design.
package context

import (
	"fmt"
	"strings"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
)

// Manager builds the optimal message list for an LLM call.
type Manager struct {
	budget     *TokenBudget
	builder    *PromptBuilder
	compressor *Compressor
	tokenFunc  func(providers.Message) int
}

// NewManager creates a context manager with default settings.
func NewManager(totalTokens int, baseSystemPrompt string) *Manager {
	budget := &TokenBudget{
		TotalTokens:  totalTokens,
		SystemRatio:  0.2,
		ToolsRatio:   0.3,
		HistoryRatio: 0.5,
	}
	budget.Compute()

	return &Manager{
		budget:     budget,
		builder:    NewPromptBuilder(baseSystemPrompt),
		compressor: NewCompressor(),
		tokenFunc:  DefaultTokenEstimator,
	}
}

// NewManagerWithConfig creates a context manager with custom configuration.
func NewManagerWithConfig(cfg ManagerConfig) *Manager {
	budget := &TokenBudget{
		TotalTokens:  cfg.TotalTokens,
		SystemRatio:  cfg.SystemRatio,
		ToolsRatio:   cfg.ToolsRatio,
		HistoryRatio: cfg.HistoryRatio,
	}
	budget.Compute()

	return &Manager{
		budget:     budget,
		builder:    NewPromptBuilder(cfg.BaseSystemPrompt),
		compressor: NewCompressorWithConfig(cfg.MaxToolOutput, cfg.SummarizeThreshold),
		tokenFunc:  DefaultTokenEstimator,
	}
}

// ManagerConfig holds configuration for the context manager.
type ManagerConfig struct {
	TotalTokens       int
	SystemRatio       float64
	ToolsRatio        float64
	HistoryRatio      float64
	BaseSystemPrompt  string
	MaxToolOutput     int
	SummarizeThreshold int
}

// DefaultConfig returns sensible defaults for most LLM providers.
func DefaultConfig() ManagerConfig {
	return ManagerConfig{
		TotalTokens:       8192,
		SystemRatio:       0.2,
		ToolsRatio:        0.3,
		HistoryRatio:      0.5,
		BaseSystemPrompt:  "",
		MaxToolOutput:     2000,
		SummarizeThreshold: 10,
	}
}

// BuildContext creates the final message list for an LLM call.
func (m *Manager) BuildContext(
	sess *session.Session,
	toolDefs []tools.ToolDefinition,
	extraPrompt string,
) []providers.Message {
	messages := sess.GetMessages()
	if len(messages) == 0 {
		return nil
	}

	// 1. Extract system message
	var systemMsg *providers.Message
	var history []providers.Message
	if messages[0].Role == providers.RoleSystem {
		systemMsg = &messages[0]
		history = messages[1:]
	} else {
		history = messages
	}

	// 2. Build dynamic system prompt
	systemPrompt := m.builder.Build(systemMsg, toolDefs, extraPrompt)
	systemTokens := m.tokenFunc(providers.Message{Role: providers.RoleSystem, Content: systemPrompt})

	// 3. Compute available budget for history
	historyBudget := m.budget.HistoryTokens
	if systemTokens < m.budget.SystemTokens {
		// Unused system budget flows to history
		historyBudget += m.budget.SystemTokens - systemTokens
	}

	// 4. Compress history to fit budget
	compressed := m.compressor.Compress(history, historyBudget, m.tokenFunc)

	// 5. Assemble result
	result := make([]providers.Message, 0, 1+len(compressed))
	result = append(result, providers.Message{
		Role:    providers.RoleSystem,
		Content: systemPrompt,
	})
	result = append(result, compressed...)

	return result
}

// Budget returns the current token budget configuration.
func (m *Manager) Budget() *TokenBudget {
	return m.budget
}

// Builder returns the prompt builder for customization.
func (m *Manager) Builder() *PromptBuilder {
	return m.builder
}

// DefaultTokenEstimator gives a rough token estimate for a message.
// Uses chars/4 for ASCII and chars/2 for CJK characters.
func DefaultTokenEstimator(msg providers.Message) int {
	var cjkCount, asciiCount int
	for _, r := range msg.Content {
		if isCJKRune(r) {
			cjkCount++
		} else {
			asciiCount++
		}
	}
	// Account for tool calls
	toolCallTokens := len(msg.ToolCalls) * 50
	return (cjkCount+1)/2 + (asciiCount+3)/4 + toolCallTokens
}

func isCJKRune(r rune) bool {
	if r >= 0x4E00 && r <= 0x9FFF {
		return true
	}
	if r >= 0x3400 && r <= 0x4DBF {
		return true
	}
	if r >= 0xF900 && r <= 0xFAFF {
		return true
	}
	if r >= 0x3040 && r <= 0x309F {
		return true
	}
	if r >= 0x30A0 && r <= 0x30FF {
		return true
	}
	if r >= 0xAC00 && r <= 0xD7AF {
		return true
	}
	if r >= 0x1100 && r <= 0x11FF {
		return true
	}
	return false
}

// BudgetReport returns a human-readable summary of token usage.
func (m *Manager) BudgetReport(messages []providers.Message) string {
	total := 0
	for _, msg := range messages {
		total += m.tokenFunc(msg)
	}
	return fmt.Sprintf("tokens: %d/%d (%.0f%% used)", total, m.budget.TotalTokens, float64(total)/float64(m.budget.TotalTokens)*100)
}

// SetToolSummary sets a custom tool summary for the system prompt.
func (m *Manager) SetToolSummary(summary string) {
	m.builder.SetToolSummary(summary)
}

// SetSkillPrompts sets skill prompts to inject into system prompt.
func (m *Manager) SetSkillPrompts(prompts string) {
	m.builder.SetSkillPrompts(prompts)
}

// SetContextHints sets context hints (time, user prefs, etc.).
func (m *Manager) SetContextHints(hints string) {
	m.builder.SetContextHints(hints)
}

// FormatToolSummary creates a concise tool summary from tool definitions.
func FormatToolSummary(toolDefs []tools.ToolDefinition) string {
	if len(toolDefs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Available tools:\n")
	for _, td := range toolDefs {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", td.Name, td.Description))
	}
	return sb.String()
}
