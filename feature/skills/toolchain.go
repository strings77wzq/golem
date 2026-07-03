package skills

import (
	"context"
	"fmt"
	"strings"

	"github.com/strings77wzq/golem/core/tools"
)

// ToolChain executes a skill's steps as an ordered workflow.
type ToolChain struct {
	registry *tools.Registry
}

// NewToolChain creates a tool-chain executor backed by a tool registry.
func NewToolChain(registry *tools.Registry) *ToolChain {
	return &ToolChain{registry: registry}
}

// Execute runs all steps in a skill's workflow sequentially.
// Each step's output is stored by OutputVar for use in subsequent steps.
// Returns the final step's output, or an error if any step fails.
func (tc *ToolChain) Execute(ctx context.Context, skill *Skill, initialInput map[string]interface{}) (string, error) {
	if !skill.HasSteps() {
		return "", fmt.Errorf("skill %q has no steps", skill.Name)
	}

	variables := make(map[string]interface{})
	for k, v := range initialInput {
		variables[k] = v
	}

	var lastOutput string

	for i, step := range skill.Steps {
		if step.Condition != "" {
			if !evaluateCondition(step.Condition, variables) {
				continue
			}
		}

		tool, found := tc.registry.Get(step.Tool)
		if !found {
			return "", fmt.Errorf("step %d: tool %q not found", i, step.Tool)
		}

		args := resolveArgs(step.Input, variables)

		result, err := tool.Execute(ctx, args)
		if err != nil {
			return "", fmt.Errorf("step %d (%s): %w", i, step.Tool, err)
		}

		lastOutput = result.ForLLM

		if step.OutputVar != "" {
			variables[step.OutputVar] = result.ForLLM
		}
	}

	return lastOutput, nil
}

func resolveArgs(template map[string]string, variables map[string]interface{}) map[string]interface{} {
	if template == nil {
		return nil
	}
	args := make(map[string]interface{}, len(template))
	for k, v := range template {
		args[k] = interpolateString(v, variables)
	}
	return args
}

const maxInterpolationDepth = 10

func interpolateString(s string, variables map[string]interface{}) string {
	for i := 0; i < maxInterpolationDepth; i++ {
		start := strings.Index(s, "{{")
		if start == -1 {
			return s
		}
		end := strings.Index(s[start:], "}}")
		if end == -1 {
			return s
		}
		end += start
		varName := strings.TrimSpace(s[start+2 : end])
		if val, ok := variables[varName]; ok {
			s = s[:start] + fmt.Sprintf("%v", val) + s[end+2:]
		} else {
			break
		}
	}
	return s
}

func evaluateCondition(condition string, variables map[string]interface{}) bool {
	condition = strings.TrimSpace(condition)
	if strings.HasPrefix(condition, "var:") {
		varName := strings.TrimPrefix(condition, "var:")
		val, ok := variables[varName]
		if !ok {
			return false
		}
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v != "" && v != "false" && v != "0"
		case nil:
			return false
		default:
			return true
		}
	}
	return true
}
