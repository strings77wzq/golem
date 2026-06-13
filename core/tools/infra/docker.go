package infra

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/strings77wzq/golem/core/tools"
)

// CommandExecutor runs system commands.
type CommandExecutor interface {
	Execute(ctx context.Context, name string, args ...string) (string, error)
}

// DefaultExecutor uses os/exec to run commands.
type DefaultExecutor struct{}

func (e *DefaultExecutor) Execute(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stdout.String(), fmt.Errorf("%s: %w\n%s", err, err, stderr.String())
	}
	return stdout.String(), nil
}

// DockerTool executes Docker commands.
type DockerTool struct {
	executor CommandExecutor
}

// NewDockerTool creates a new Docker tool.
func NewDockerTool() *DockerTool {
	return &DockerTool{executor: &DefaultExecutor{}}
}

func (t *DockerTool) Name() string        { return "docker" }
func (t *DockerTool) Description() string { return "Execute Docker operations: build, run, stop, ps, logs, exec, push, images" }

func (t *DockerTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "action", Type: "string", Description: "Action: build, run, stop, ps, logs, exec, push, images", Required: true},
		{Name: "image", Type: "string", Description: "Image name (for build/run)", Required: false},
		{Name: "tag", Type: "string", Description: "Image tag (for build/push)", Required: false},
		{Name: "container", Type: "string", Description: "Container name/ID (for stop/logs/exec)", Required: false},
		{Name: "command", Type: "array", Description: "Command to execute (for exec)", Required: false},
		{Name: "context", Type: "string", Description: "Build context path (for build)", Required: false},
		{Name: "dockerfile", Type: "string", Description: "Dockerfile path (for build)", Required: false},
		{Name: "tail", Type: "number", Description: "Number of log lines (for logs)", Required: false},
		{Name: "filter", Type: "string", Description: "Filter expression (for ps/images)", Required: false},
		{Name: "all", Type: "boolean", Description: "Show all containers (for ps)", Required: false},
	}
}

func (t *DockerTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return &tools.ToolResult{ForLLM: "Error: action is required", IsError: true}, nil
	}

	switch action {
	case "build":
		return t.build(ctx, args)
	case "run":
		return t.run(ctx, args)
	case "stop":
		return t.stop(ctx, args)
	case "ps":
		return t.ps(ctx, args)
	case "logs":
		return t.logs(ctx, args)
	case "exec":
		return t.exec(ctx, args)
	case "push":
		return t.push(ctx, args)
	case "images":
		return t.images(ctx, args)
	default:
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: unknown action %q", action), IsError: true}, nil
	}
}

func (t *DockerTool) build(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	contextPath, _ := args["context"].(string)
	if contextPath == "" {
		contextPath = "."
	}
	tag, _ := args["tag"].(string)
	dockerfile, _ := args["dockerfile"].(string)

	cmdArgs := []string{"build"}
	if tag != "" {
		cmdArgs = append(cmdArgs, "-t", tag)
	}
	if dockerfile != "" {
		cmdArgs = append(cmdArgs, "-f", dockerfile)
	}
	cmdArgs = append(cmdArgs, contextPath)

	output, err := t.executor.Execute(ctx, "docker", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Build failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Built image %s", tag)}, nil
}

func (t *DockerTool) run(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	image, _ := args["image"].(string)
	if image == "" {
		return &tools.ToolResult{ForLLM: "Error: image is required", IsError: true}, nil
	}

	cmdArgs := []string{"run", "-d"}
	if name, ok := args["container"].(string); ok && name != "" {
		cmdArgs = append(cmdArgs, "--name", name)
	}
	cmdArgs = append(cmdArgs, image)

	output, err := t.executor.Execute(ctx, "docker", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Run failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Started container from %s", image)}, nil
}

func (t *DockerTool) stop(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	container, _ := args["container"].(string)
	if container == "" {
		return &tools.ToolResult{ForLLM: "Error: container is required", IsError: true}, nil
	}

	output, err := t.executor.Execute(ctx, "docker", "stop", container)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Stop failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Stopped container %s", container)}, nil
}

func (t *DockerTool) ps(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	cmdArgs := []string{"ps"}
	if all, ok := args["all"].(bool); ok && all {
		cmdArgs = append(cmdArgs, "-a")
	}
	if filter, ok := args["filter"].(string); ok && filter != "" {
		cmdArgs = append(cmdArgs, "--filter", filter)
	}
	cmdArgs = append(cmdArgs, "--format", "table {{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Names}}")

	output, err := t.executor.Execute(ctx, "docker", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("List failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: "Containers listed"}, nil
}

func (t *DockerTool) logs(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	container, _ := args["container"].(string)
	if container == "" {
		return &tools.ToolResult{ForLLM: "Error: container is required", IsError: true}, nil
	}

	cmdArgs := []string{"logs", "--tail", "100"}
	if tail, ok := args["tail"].(float64); ok {
		cmdArgs[2] = fmt.Sprintf("%d", int(tail))
	}
	cmdArgs = append(cmdArgs, container)

	output, err := t.executor.Execute(ctx, "docker", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Logs failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Logs for %s", container)}, nil
}

func (t *DockerTool) exec(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	container, _ := args["container"].(string)
	if container == "" {
		return &tools.ToolResult{ForLLM: "Error: container is required", IsError: true}, nil
	}

	command, _ := args["command"].([]interface{})
	if len(command) == 0 {
		return &tools.ToolResult{ForLLM: "Error: command is required", IsError: true}, nil
	}

	cmdArgs := []string{"exec", container}
	for _, c := range command {
		cmdArgs = append(cmdArgs, fmt.Sprintf("%v", c))
	}

	output, err := t.executor.Execute(ctx, "docker", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Exec failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Executed in %s", container)}, nil
}

func (t *DockerTool) push(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	tag, _ := args["tag"].(string)
	if tag == "" {
		return &tools.ToolResult{ForLLM: "Error: tag is required", IsError: true}, nil
	}

	output, err := t.executor.Execute(ctx, "docker", "push", tag)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Push failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: fmt.Sprintf("Pushed %s", tag)}, nil
}

func (t *DockerTool) images(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	cmdArgs := []string{"images", "--format", "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedSince}}"}
	if filter, ok := args["filter"].(string); ok && filter != "" {
		cmdArgs = append(cmdArgs, "--filter", filter)
	}

	output, err := t.executor.Execute(ctx, "docker", cmdArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Images failed: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: output, ForUser: "Images listed"}, nil
}
