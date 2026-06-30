package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/strings77wzq/golem/core/tools"
)

// DelegateTool allows an agent to delegate tasks to other agents via MCP.
type DelegateTool struct {
	timeout time.Duration
}

// NewDelegateTool creates a new delegate tool.
func NewDelegateTool() *DelegateTool {
	return &DelegateTool{timeout: 30 * time.Second}
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
		{Name: "timeout", Type: "number", Description: "Timeout in seconds (default 30)", Required: false},
	}
}

// Execute delegates a tool call to another agent via MCP.
func (t *DelegateTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	command, _ := args["command"].(string)
	toolName, _ := args["tool"].(string)
	toolArgs, _ := args["arguments"].(map[string]interface{})
	commandArgs, _ := args["args"].([]interface{})

	if command == "" || toolName == "" {
		return &tools.ToolResult{ForLLM: "Error: command and tool are required", IsError: true}, nil
	}

	// Convert args
	argsList := make([]string, len(commandArgs))
	for i, a := range commandArgs {
		argsList[i] = fmt.Sprintf("%v", a)
	}

	// Start the agent process
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, argsList...) // #nosec G204 -- subprocess with controlled input
	cmd.Stderr = nil // suppress stderr

	// Create pipe for stdin/stdout
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error starting agent: %v", err), IsError: true}, nil
	}
	defer cmd.Wait() // reap child process on all exit paths

	// Create MCP client connected to the agent
	transport := &stdioTransport{stdin: stdin, stdout: stdout}
	client := NewClient(transport)

	// Initialize
	initCtx, initCancel := context.WithTimeout(ctx, 5*time.Second)
	defer initCancel()
	if _, err := client.Initialize(initCtx); err != nil {
		cmd.Process.Kill()
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error initializing agent: %v", err), IsError: true}, nil
	}

	// Call the tool
	result, err := client.CallTool(ctx, toolName, toolArgs)
	cmd.Process.Kill()

	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error calling tool %s: %v", toolName, err), IsError: true}, nil
	}

	// Extract text from result
	var text string
	for _, block := range result.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}

	return &tools.ToolResult{ForLLM: text, ForUser: text}, nil
}

// stdioTransport wraps stdin/stdout as a Transport.
type stdioTransport struct {
	stdin  interface{ Write([]byte) (int, error) }
	stdout interface{ Read([]byte) (int, error) }
}

func (t *stdioTransport) Send(req *JSONRPCRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')
	_, err = t.stdin.Write(data)
	return err
}

func (t *stdioTransport) SendNotification(n *JSONRPCNotification) error {
	data, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}
	data = append(data, '\n')
	_, err = t.stdin.Write(data)
	return err
}

func (t *stdioTransport) Receive() (*JSONRPCResponse, error) {
	line := make([]byte, 4096)
	n, err := t.stdout.Read(line)
	if err != nil {
		return nil, err
	}
	var resp JSONRPCResponse
	if err := json.Unmarshal(line[:n], &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (t *stdioTransport) Close() error {
	if closer, ok := t.stdin.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}
