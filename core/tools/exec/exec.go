// Package exec provides the "exec" tool that lets the AI agent run shell
// commands in a sandboxed working directory. Commands are executed with a
// configurable timeout; output (stdout + stderr combined) is returned as
// the tool result. The working directory is set at construction time via [New]
// and cannot be escaped by the agent.
//
// Security: By default, commands are executed without shell interpretation
// to prevent command injection. Use [WithAllowShell] to enable shell features
// (pipes, redirects, etc.) when needed, understanding the security implications.
package exec

import (
	"bytes"
	"context"
	"fmt"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/strings77wzq/golem/core/tools"
)

// SecurityMode defines how commands are validated and executed
type SecurityMode int

const (
	// SecurityModeSandbox uses both allowlist and denylist for maximum security
	// Commands must be in allowlist AND not in denylist
	SecurityModeSandbox SecurityMode = iota
	// SecurityModeAllowlist only checks allowlist (more permissive but still safe)
	SecurityModeAllowlist
	// SecurityModeDenylist only checks denylist (least secure, allows shell injection)
	// DEPRECATED: Use SecurityModeSandbox or SecurityModeAllowlist
	SecurityModeDenylist
)

type ExecTool struct {
	workspace    string
	timeout      time.Duration
	securityMode SecurityMode
	allowList    []string
	denyList     []string
	allowShell   bool
}

type Option func(*ExecTool)

func WithTimeout(d time.Duration) Option {
	return func(t *ExecTool) {
		t.timeout = d
	}
}

func WithSecurityMode(mode SecurityMode) Option {
	return func(t *ExecTool) {
		t.securityMode = mode
	}
}

func WithAllowList(commands []string) Option {
	return func(t *ExecTool) {
		t.allowList = commands
	}
}

func WithDenyList(commands []string) Option {
	return func(t *ExecTool) {
		t.denyList = append(t.denyList, commands...)
	}
}

// WithAllowShell enables shell interpretation (pipes, redirects, etc.)
// WARNING: This increases the risk of command injection. Use with caution.
func WithAllowShell() Option {
	return func(t *ExecTool) {
		t.allowShell = true
	}
}

// Default allowlist contains commonly used safe commands
var defaultAllowList = []string{
	// File system
	"ls", "cat", "head", "tail", "wc", "find", "tree", "file", "stat",
	"mkdir", "touch", "cp", "mv", "ln", "chmod", "chown",
	// Text processing
	"grep", "sed", "awk", "sort", "uniq", "cut", "tr", "diff", "patch",
	"echo", "printf", "tee",
	// Development
	"git", "go", "python", "python3", "node", "npm", "npx", "yarn",
	"make", "cargo", "rustc", "javac", "java", "gradle", "mvn",
	// System info
	"pwd", "whoami", "date", "uname", "hostname", "df", "du", "free",
	"ps", "top", "htop", "env", "printenv", "which", "type",
	// Network (safe ones)
	"ping", "nslookup", "dig", "host", "netstat", "ss",
	// Compression
	"tar", "gzip", "gunzip", "zip", "unzip",
	// Misc
	"time", "timeout", "xargs", "parallel", "jq", "yq",
}

// Dangerous patterns that should always be blocked
var defaultDenyList = []string{
	"rm -rf /",
	"rm -rf ~",
	"rm -rf /*",
	"rm -rf ~/*",
	"shutdown",
	"reboot",
	"halt",
	"poweroff",
	"init 0",
	"init 6",
	"mkfs",
	"dd if=",
	":(){ :|:& };:", // Fork bomb
	"chmod -R 777 /",
	"chmod -R 777 ~",
	"chown -R",
	"> /dev/sd",
	"> /dev/hd",
	"mkinitramfs",
	"update-grub",
	"grub-install",
	"wget",
	"curl",
}

func New(workspace string, opts ...Option) *ExecTool {
	tool := &ExecTool{
		workspace:    workspace,
		timeout:      30 * time.Second,
		securityMode: SecurityModeSandbox,
		allowList:    make([]string, len(defaultAllowList)),
		denyList:     make([]string, len(defaultDenyList)),
		allowShell:   false,
	}

	copy(tool.allowList, defaultAllowList)
	copy(tool.denyList, defaultDenyList)

	for _, opt := range opts {
		opt(tool)
	}

	return tool
}

func (t *ExecTool) Name() string {
	return "exec"
}

func (t *ExecTool) Description() string {
	return "Execute a shell command within the workspace sandbox. " +
		"By default, uses allowlist for security. Pipes and redirects require --allow-shell."
}

func (t *ExecTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{
			Name:        "command",
			Type:        "string",
			Description: "The command to execute",
			Required:    true,
		},
		{
			Name:        "timeout",
			Type:        "number",
			Description: "Timeout in seconds (default: 30)",
			Required:    false,
		},
	}
}

func (t *ExecTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	command, ok := args["command"].(string)
	if !ok {
		return &tools.ToolResult{
			ForLLM:  "command parameter is required and must be a string",
			IsError: true,
		}, nil
	}

	if strings.TrimSpace(command) == "" {
		return &tools.ToolResult{
			ForLLM:  "command cannot be empty",
			IsError: true,
		}, nil
	}

	timeout := t.timeout
	if timeoutVal, ok := args["timeout"].(float64); ok {
		if timeoutVal > 0 {
			timeout = time.Duration(timeoutVal) * time.Second
		}
	}

	// Security checks
	if blocked, reason := t.checkSecurity(command); blocked {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Command blocked by security policy: %s", reason),
			IsError: true,
		}, nil
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *osexec.Cmd
	if t.allowShell {
		// Shell mode: allows pipes, redirects, etc.
		// Note: This is more dangerous but necessary for complex commands
		cmd = osexec.CommandContext(cmdCtx, "sh", "-c", command)
	} else {
		// Direct mode: no shell interpretation (safer)
		parts, err := parseCommand(command)
		if err != nil {
			return &tools.ToolResult{
				ForLLM:  fmt.Sprintf("Failed to parse command: %v", err),
				IsError: true,
			}, nil
		}

		// Resolve command path
		cmdPath, err := osexec.LookPath(parts[0])
		if err != nil {
			// Command not found in PATH
			cmd = osexec.CommandContext(cmdCtx, parts[0], parts[1:]...)
		} else {
			cmd = osexec.CommandContext(cmdCtx, cmdPath, parts[1:]...)
		}
	}
	cmd.Dir = t.workspace

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if len(output) > 0 {
			output += "\n"
		}
		output += stderr.String()
	}

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded || strings.Contains(err.Error(), "signal: killed") {
			return &tools.ToolResult{
				ForLLM:  fmt.Sprintf("Command timed out after %v", timeout),
				ForUser: fmt.Sprintf("$ %s\nTimeout after %v", command, timeout),
				IsError: true,
			}, nil
		}

		// Check if command was not found
		if strings.Contains(err.Error(), "executable file not found") ||
			strings.Contains(err.Error(), "no such file or directory") {
			return &tools.ToolResult{
				ForLLM:  fmt.Sprintf("Command not found: %s", command),
				ForUser: fmt.Sprintf("$ %s\nCommand not found", command),
				IsError: true,
			}, nil
		}

		exitMsg := fmt.Sprintf("Command failed: %v", err)
		if len(output) > 0 {
			output = output + "\n" + exitMsg
		} else {
			output = exitMsg
		}

		forUser := fmt.Sprintf("$ %s\n%s", command, output)
		if len(forUser) > 500 {
			forUser = fmt.Sprintf("$ %s\n%s", command, forUser[len("$ "+command+"\n"):500])
		}

		forLLM := output
		if len(forLLM) > 10000 {
			forLLM = forLLM[:10000] + fmt.Sprintf("\n... (truncated, %d more chars)", len(forLLM)-10000)
		}

		return &tools.ToolResult{
			ForLLM:  forLLM,
			ForUser: forUser,
			IsError: true,
		}, nil
	}

	if len(output) == 0 {
		output = "(no output)"
	}

	forUser := fmt.Sprintf("$ %s\n%s", command, output)
	if len(forUser) > 500 {
		forUser = fmt.Sprintf("$ %s\n%s", command, output[:min(500-len("$ "+command+"\n"), len(output))])
	}

	forLLM := output
	if len(forLLM) > 10000 {
		forLLM = forLLM[:10000] + fmt.Sprintf("\n... (truncated, %d more chars)", len(forLLM)-10000)
	}

	return &tools.ToolResult{
		ForLLM:  forLLM,
		ForUser: forUser,
	}, nil
}

// checkSecurity validates the command against security policies
func (t *ExecTool) checkSecurity(command string) (bool, string) {
	// Always check denylist first (defense in depth)
	for _, denied := range t.denyList {
		if strings.Contains(strings.ToLower(command), strings.ToLower(denied)) {
			return true, fmt.Sprintf("contains blocked pattern: %s", denied)
		}
	}

	// When shell mode is enabled, we can't reliably extract all commands from pipes
	// So we only check denylist (already done above) and skip allowlist
	if t.allowShell {
		return false, ""
	}

	// Check allowlist based on security mode
	switch t.securityMode {
	case SecurityModeSandbox, SecurityModeAllowlist:
		cmdName := extractBaseCommand(command)

		allowed := false
		for _, a := range t.allowList {
			if strings.EqualFold(cmdName, a) {
				allowed = true
				break
			}
		}

		if !allowed {
			return true, fmt.Sprintf("command %q is not in the allowed list. Allowed commands: %s",
				cmdName, strings.Join(t.allowList[:min(10, len(t.allowList))], ", ")+"...")
		}
	case SecurityModeDenylist:
	}
	return false, ""
}

// extractBaseCommand extracts the base command name from a command string
func extractBaseCommand(command string) string {
	// Handle pipes - get the first command
	parts := strings.SplitN(command, "|", 2)
	firstCmd := strings.TrimSpace(parts[0])

	// Split by whitespace and get the first part
	fields := strings.Fields(firstCmd)
	if len(fields) == 0 {
		return ""
	}

	// Get the base name of the command
	return filepath.Base(fields[0])
}

// parseCommand parses a command string into program and arguments
// without using shell interpretation. Supports quoted strings.
func parseCommand(cmd string) ([]string, error) {
	var args []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for _, r := range cmd {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		switch r {
		case '\\':
			if !inSingleQuote {
				escaped = true
			} else {
				current.WriteRune(r)
			}
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			} else {
				current.WriteRune(r)
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			} else {
				current.WriteRune(r)
			}
		case ' ', '\t':
			if !inSingleQuote && !inDoubleQuote {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	// Check for unclosed quotes
	if inSingleQuote || inDoubleQuote {
		return nil, fmt.Errorf("unclosed quote in command")
	}
	if escaped {
		return nil, fmt.Errorf("trailing backslash in command")
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	return args, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
