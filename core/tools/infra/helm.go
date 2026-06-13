package infra

import (
	"context"
	"fmt"

	"github.com/strings77wzq/golem/core/tools"
)

// HelmTool executes Helm commands.
type HelmTool struct {
	executor CommandExecutor
}

// NewHelmTool creates a new Helm tool.
func NewHelmTool() *HelmTool {
	return &HelmTool{executor: &DefaultExecutor{}}
}

func (t *HelmTool) Name() string        { return "helm" }
func (t *HelmTool) Description() string { return "Execute Helm operations: list, install, upgrade, rollback" }

func (t *HelmTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "action", Type: "string", Description: "Action: list, install, upgrade, rollback", Required: true},
		{Name: "release", Type: "string", Description: "Release name", Required: false},
		{Name: "chart", Type: "string", Description: "Chart path (for install/upgrade)", Required: false},
		{Name: "namespace", Type: "string", Description: "Kubernetes namespace", Required: false},
		{Name: "values", Type: "object", Description: "Values to set (for install/upgrade)", Required: false},
		{Name: "set", Type: "object", Description: "Set individual values (for upgrade)", Required: false},
		{Name: "revision", Type: "number", Description: "Revision number (for rollback)", Required: false},
	}
}

func (t *HelmTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return &tools.ToolResult{ForLLM: "Error: action is required", IsError: true}, nil
	}

	switch action {
	case "list":
		return t.list(ctx, args)
	case "install":
		return t.install(ctx, args)
	case "upgrade":
		return t.upgrade(ctx, args)
	case "rollback":
		return t.rollback(ctx, args)
	default:
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: unknown action %q", action), IsError: true}, nil
	}
}

func (t *HelmTool) list(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	cmdArgs := []string{"list"}
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		cmdArgs = append(cmdArgs, "-n", ns)
	}

	output, err := t.executor.Execute(ctx, "helm", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("List failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: "Helm releases listed"}, nil
}

func (t *HelmTool) install(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	release, _ := args["release"].(string)
	chart, _ := args["chart"].(string)
	if release == "" || chart == "" {
		return &tools.ToolResult{ForLLM: "Error: release and chart are required", IsError: true}, nil
	}

	cmdArgs := []string{"install", release, chart}
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		cmdArgs = append(cmdArgs, "-n", ns)
	}

	output, err := t.executor.Execute(ctx, "helm", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Install failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Installed release %s", release)}, nil
}

func (t *HelmTool) upgrade(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	release, _ := args["release"].(string)
	chart, _ := args["chart"].(string)
	if release == "" || chart == "" {
		return &tools.ToolResult{ForLLM: "Error: release and chart are required", IsError: true}, nil
	}

	cmdArgs := []string{"upgrade", release, chart}
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		cmdArgs = append(cmdArgs, "-n", ns)
	}

	output, err := t.executor.Execute(ctx, "helm", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Upgrade failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Upgraded release %s", release)}, nil
}

func (t *HelmTool) rollback(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	release, _ := args["release"].(string)
	revision, ok := args["revision"].(float64)
	if release == "" || !ok {
		return &tools.ToolResult{ForLLM: "Error: release and revision are required", IsError: true}, nil
	}

	cmdArgs := []string{"rollback", release, fmt.Sprintf("%d", int(revision))}

	output, err := t.executor.Execute(ctx, "helm", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Rollback failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Rolled back %s to revision %d", release, int(revision))}, nil
}
