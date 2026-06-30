package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const defaultHookTimeout = 30 * time.Second

// ShellHook executes a shell command with JSON I/O.
type ShellHook struct {
	Command string
	Timeout time.Duration
}

// HookInput is the JSON payload sent to the hook on stdin.
type HookInput struct {
	SessionID  string                 `json:"session_id"`
	ToolName   string                 `json:"tool_name"`
	ToolInput  map[string]interface{} `json:"tool_input,omitempty"`
	ToolOutput string                 `json:"tool_output,omitempty"`
}

// HookOutput is the JSON payload received from the hook on stdout.
type HookOutput struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// Execute runs the hook command with input JSON on stdin, parses output.
// Empty command = no-op (allowed). Timeout/error = blocked.
func (h *ShellHook) Execute(input *HookInput) (*HookOutput, error) {
	if h.Command == "" {
		return &HookOutput{Allowed: true}, nil
	}

	timeout := h.Timeout
	if timeout == 0 {
		timeout = defaultHookTimeout
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshaling hook input: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", h.Command) // #nosec G204 -- subprocess with controlled input
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &HookOutput{
			Allowed: false,
			Reason:  fmt.Sprintf("hook command failed: %v, stderr: %s", err, stderr.String()),
		}, nil
	}

	var output HookOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return &HookOutput{
			Allowed: false,
			Reason:  fmt.Sprintf("invalid hook output: %v", err),
		}, nil
	}

	return &output, nil
}
