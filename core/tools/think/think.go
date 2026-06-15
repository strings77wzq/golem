// Package think provides the "think" tool — a reasoning scratchpad that lets
// the agent think step-by-step before acting. The tool has no side effects;
// it simply records the agent's reasoning and returns it as context.
// Most useful for models without native reasoning/thinking capabilities.
package think

import (
	"context"
	"fmt"

	"github.com/strings77wzq/golem/core/tools"
)

// ThinkTool is a reasoning scratchpad with no side effects.
type ThinkTool struct{}

// New creates a new ThinkTool.
func New() *ThinkTool {
	return &ThinkTool{}
}

func (t *ThinkTool) Name() string { return "think" }

func (t *ThinkTool) Description() string {
	return "Reasoning scratchpad — think step-by-step before acting. " +
		"Use this to plan, analyze, or break down complex problems before taking action. " +
		"No side effects."
}

func (t *ThinkTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "thought",
			Type:        "string",
			Description: "Your step-by-step reasoning about the problem",
			Required:    true,
		},
	}
}

func (t *ThinkTool) Execute(_ context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	thought, _ := args["thought"].(string)
	if thought == "" {
		return &tools.ToolResult{
			ForLLM:  "Error: thought parameter is required",
			IsError: true,
		}, nil
	}

	return &tools.ToolResult{
		ForLLM:  fmt.Sprintf("[Thinking]\n%s\n[/Thinking]", thought),
		ForUser: "",
		Silent:  true,
	}, nil
}
