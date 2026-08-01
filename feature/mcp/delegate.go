package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/strings77wzq/golem/core/tools"
)

// DelegateTool allows an agent to delegate tasks to other agents via MCP.
// The spawned command is caller-controlled, so the tool is fail-closed by
// default: no commands are allowed unless an explicit allowlist is provided
// via NewDelegateToolWithAllowlist. The tool is currently not registered
// into any production registry; wire it in only with an allowlist.
type DelegateTool struct {
	timeout   time.Duration
	allowlist map[string]struct{}
}

// NewDelegateTool creates a delegate tool that rejects every command
// (empty allowlist = fail closed).
func NewDelegateTool() *DelegateTool {
	return &DelegateTool{
		timeout:   30 * time.Second,
		allowlist: make(map[string]struct{}),
	}
}

// NewDelegateToolWithAllowlist creates a delegate tool that only runs the
// given commands.
func NewDelegateToolWithAllowlist(commands ...string) *DelegateTool {
	t := NewDelegateTool()
	for _, c := range commands {
		t.allowlist[c] = struct{}{}
	}
	return t
}

func (t *DelegateTool) Name() string { return "delegate" }
func (t *DelegateTool) Description() string {
	return "Delegate a task to another agent via MCP stdio"
}
func (t *DelegateTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "command", Type: "string", Description: "Command to start the agent (e.g. 'golem mcp-server')", Required: true},
		{Name: "args", Type: "array", Description: "Arguments for the command", Required: false},
		{Name: "tool", Type: "string", Description: "Tool name to call on the agent", Required: true},
		{Name: "arguments", Type: "object", Description: "Arguments for the tool call", Required: true},
		{Name: "timeout", Type: "number", Description: "Timeout in seconds (default 30, capped at 30)", Required: false},
	}
}

// Execute delegates a tool call to another agent via MCP using the official
// go-sdk client with a CommandTransport.
func (t *DelegateTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	command, _ := args["command"].(string)
	toolName, _ := args["tool"].(string)
	toolArgs, _ := args["arguments"].(map[string]interface{})
	commandArgs, _ := args["args"].([]interface{})

	if command == "" || toolName == "" {
		return &tools.ToolResult{ForLLM: "Error: command and tool are required", IsError: true}, nil
	}

	// Fail closed: only allowlisted commands may be spawned.
	if _, allowed := t.allowlist[command]; !allowed {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Error: command %q not in allowlist", command),
			IsError: true,
		}, nil
	}

	// Honor the caller-provided timeout, capped at the tool default so it
	// cannot be used to escalate past the built-in bound.
	timeout := t.timeout
	if v, ok := args["timeout"].(float64); ok && v > 0 {
		if d := time.Duration(v) * time.Second; d < timeout {
			timeout = d
		}
	}

	// Convert args
	argsList := make([]string, len(commandArgs))
	for i, a := range commandArgs {
		argsList[i] = fmt.Sprintf("%v", a)
	}

	// Start the agent process
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, argsList...) // #nosec G204 -- caller-controlled delegate command
	cmd.Stderr = nil                                      // suppress stderr

	client := sdk.NewClient(&sdk.Implementation{Name: "golem", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, &sdk.CommandTransport{Command: cmd}, nil)
	if err != nil {
		// cmd.Start may never have succeeded (nonexistent/unexecutable
		// command) — Process is nil then, and Kill would panic.
		if cmd.Process != nil {
			cmd.Process.Kill() //nolint:errcheck
		}
		return &tools.ToolResult{ForLLM: fmt.Sprintf("error initializing agent: %v", err), IsError: true}, nil
	}
	defer session.Close() //nolint:errcheck

	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: toolName, Arguments: toolArgs})
	if cmd.Process != nil {
		cmd.Process.Kill() //nolint:errcheck
	}

	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("error calling tool %s: %v", toolName, err), IsError: true}, nil
	}

	// Extract text from result
	var text string
	for _, block := range result.Content {
		if tc, ok := block.(*sdk.TextContent); ok {
			text += tc.Text
		}
	}

	return &tools.ToolResult{ForLLM: text, ForUser: text}, nil
}
