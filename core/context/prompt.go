package context

import (
	"fmt"
	"strings"
	"time"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/tools"
)

// PromptBuilder assembles the system prompt dynamically from multiple sources.
type PromptBuilder struct {
	basePrompt   string
	toolSummary  string
	skillPrompts string
	contextHints *string // nil = use defaults, non-nil = use value (even if empty)
}

// NewPromptBuilder creates a prompt builder with a base system prompt.
func NewPromptBuilder(basePrompt string) *PromptBuilder {
	return &PromptBuilder{
		basePrompt: basePrompt,
	}
}

// Build assembles the final system prompt from all sources.
// Order: base prompt → tool summary → skill prompts → context hints.
func (pb *PromptBuilder) Build(
	systemMsg *providers.Message,
	toolDefs []tools.ToolDefinition,
	extraPrompt string,
) string {
	var sb strings.Builder

	// 1. Base system prompt (from config or message)
	base := pb.basePrompt
	if systemMsg != nil && systemMsg.Content != "" {
		base = systemMsg.Content
	}
	if base != "" {
		sb.WriteString(base)
		sb.WriteString("\n\n")
	}

	// 2. Tool summary (auto-generated from tool definitions)
	toolSummary := pb.toolSummary
	if toolSummary == "" && len(toolDefs) > 0 {
		toolSummary = FormatToolSummary(toolDefs)
	}
	if toolSummary != "" {
		sb.WriteString(toolSummary)
		sb.WriteString("\n")
	}

	// 3. Skill prompts
	if pb.skillPrompts != "" {
		sb.WriteString(pb.skillPrompts)
		sb.WriteString("\n")
	}

	// 4. Context hints (time, user prefs, etc.)
	hints := ""
	if pb.contextHints != nil {
		hints = *pb.contextHints
	} else {
		hints = pb.defaultHints()
	}
	if hints != "" {
		sb.WriteString(hints)
		sb.WriteString("\n")
	}

	// 5. Extra prompt (from caller)
	if extraPrompt != "" {
		sb.WriteString(extraPrompt)
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}

// SetToolSummary sets a custom tool summary (overrides auto-generation).
func (pb *PromptBuilder) SetToolSummary(summary string) {
	pb.toolSummary = summary
}

// SetSkillPrompts sets skill prompts to inject.
func (pb *PromptBuilder) SetSkillPrompts(prompts string) {
	pb.skillPrompts = prompts
}

// SetContextHints sets context hints to inject.
// Pass nil to use defaults, pass pointer to empty string to disable.
func (pb *PromptBuilder) SetContextHints(hints string) {
	pb.contextHints = &hints
}

// defaultHints generates default context hints.
func (pb *PromptBuilder) defaultHints() string {
	now := time.Now()
	return fmt.Sprintf("Current time: %s", now.Format("2006-01-02 15:04 MST"))
}
