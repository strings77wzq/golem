// Package infra provides infrastructure tools for Docker, Kubernetes, and Helm.
package infra

import (
	"context"
	"fmt"

	"github.com/strings77wzq/golem/core/tools"
)

// KubectlTool executes Kubernetes commands.
type KubectlTool struct {
	executor CommandExecutor
}

// NewKubectlTool creates a new kubectl tool.
func NewKubectlTool() *KubectlTool {
	return &KubectlTool{executor: &DefaultExecutor{}}
}

func (t *KubectlTool) Name() string { return "kubectl" }
func (t *KubectlTool) Description() string {
	return "Execute kubectl operations: get, apply, delete, describe, logs, scale"
}

func (t *KubectlTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "action", Type: "string", Description: "Action: get, apply, delete, describe, logs, scale", Required: true},
		{Name: "resource", Type: "string", Description: "Resource type (e.g. pods, deployments, services)", Required: false},
		{Name: "name", Type: "string", Description: "Resource name", Required: false},
		{Name: "namespace", Type: "string", Description: "Kubernetes namespace (default: default)", Required: false},
		{Name: "file", Type: "string", Description: "YAML file path (for apply)", Required: false},
		{Name: "yaml", Type: "string", Description: "Inline YAML content (for apply)", Required: false},
		{Name: "output", Type: "string", Description: "Output format: wide, json, yaml (for get)", Required: false},
		{Name: "tail", Type: "number", Description: "Number of log lines (for logs)", Required: false},
		{Name: "deployment", Type: "string", Description: "Deployment name (for scale)", Required: false},
		{Name: "replicas", Type: "number", Description: "Replica count (for scale)", Required: false},
	}
}

func (t *KubectlTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return &tools.ToolResult{ForLLM: "Error: action is required", IsError: true}, nil
	}

	switch action {
	case "get":
		return t.get(ctx, args)
	case "apply":
		return t.apply(ctx, args)
	case "delete":
		return t.delete(ctx, args)
	case "describe":
		return t.describe(ctx, args)
	case "logs":
		return t.logs(ctx, args)
	case "scale":
		return t.scale(ctx, args)
	default:
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: unknown action %q", action), IsError: true}, nil
	}
}

func (t *KubectlTool) get(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	resource, _ := args["resource"].(string)
	if resource == "" {
		return &tools.ToolResult{ForLLM: "Error: resource is required", IsError: true}, nil
	}

	cmdArgs := []string{"get", resource}
	if name, ok := args["name"].(string); ok && name != "" {
		cmdArgs = append(cmdArgs, name)
	}
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		cmdArgs = append(cmdArgs, "-n", ns)
	}
	if output, ok := args["output"].(string); ok && output != "" {
		cmdArgs = append(cmdArgs, "-o", output)
	}

	output, err := t.executor.Execute(ctx, "kubectl", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Get failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Got %s", resource)}, nil
}

func (t *KubectlTool) apply(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	cmdArgs := []string{"apply"}

	if file, ok := args["file"].(string); ok && file != "" {
		cmdArgs = append(cmdArgs, "-f", file)
	} else if yaml, ok := args["yaml"].(string); ok && yaml != "" {
		cmdArgs = append(cmdArgs, "-f", "-")
	} else {
		return &tools.ToolResult{ForLLM: "Error: file or yaml is required", IsError: true}, nil
	}

	if ns, ok := args["namespace"].(string); ok && ns != "" {
		cmdArgs = append(cmdArgs, "-n", ns)
	}

	output, err := t.executor.Execute(ctx, "kubectl", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Apply failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: "Resources applied"}, nil
}

func (t *KubectlTool) delete(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	resource, _ := args["resource"].(string)
	name, _ := args["name"].(string)
	if resource == "" || name == "" {
		return &tools.ToolResult{ForLLM: "Error: resource and name are required", IsError: true}, nil
	}

	cmdArgs := []string{"delete", resource, name}
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		cmdArgs = append(cmdArgs, "-n", ns)
	}

	output, err := t.executor.Execute(ctx, "kubectl", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Delete failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Deleted %s/%s", resource, name)}, nil
}

func (t *KubectlTool) describe(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	resource, _ := args["resource"].(string)
	if resource == "" {
		return &tools.ToolResult{ForLLM: "Error: resource is required", IsError: true}, nil
	}

	cmdArgs := []string{"describe", resource}
	if name, ok := args["name"].(string); ok && name != "" {
		cmdArgs = append(cmdArgs, name)
	}
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		cmdArgs = append(cmdArgs, "-n", ns)
	}

	output, err := t.executor.Execute(ctx, "kubectl", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Describe failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Described %s", resource)}, nil
}

func (t *KubectlTool) logs(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	pod, _ := args["name"].(string)
	if pod == "" {
		pod, _ = args["pod"].(string)
	}
	if pod == "" {
		return &tools.ToolResult{ForLLM: "Error: pod name is required", IsError: true}, nil
	}

	cmdArgs := []string{"logs", pod}
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		cmdArgs = append(cmdArgs, "-n", ns)
	}
	if tail, ok := args["tail"].(float64); ok {
		cmdArgs = append(cmdArgs, "--tail", fmt.Sprintf("%d", int(tail)))
	}

	output, err := t.executor.Execute(ctx, "kubectl", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Logs failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Logs for pod %s", pod)}, nil
}

func (t *KubectlTool) scale(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	deployment, _ := args["deployment"].(string)
	if deployment == "" {
		return &tools.ToolResult{ForLLM: "Error: deployment is required", IsError: true}, nil
	}

	replicas, ok := args["replicas"].(float64)
	if !ok {
		return &tools.ToolResult{ForLLM: "Error: replicas is required", IsError: true}, nil
	}

	cmdArgs := []string{"scale", "deployment", deployment, fmt.Sprintf("--replicas=%d", int(replicas))}
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		cmdArgs = append(cmdArgs, "-n", ns)
	}

	output, err := t.executor.Execute(ctx, "kubectl", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Scale failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Scaled %s to %d replicas", deployment, int(replicas))}, nil
}
