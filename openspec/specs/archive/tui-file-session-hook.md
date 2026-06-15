# Spec: @file Reference, Session Fork, Minimal Hook System

> Version: 1.0
> Date: 2026-06-15
> Status: Approved — TDD implementation in progress

---

## 1. @file Reference in TUI

### Goal

Users type `@path/to/file` in TUI input. File content is read and injected into the prompt sent to the LLM.

### Expected Behavior

```
User input:  分析表结构 @schema.sql
LLM receives: 分析表结构
<attachments>
<file path="schema.sql" size="2048">
CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));
</file>
</attachments>
```

### Design

#### Core function: `resolveFileRefs`

Location: `internal/channels/tui/tui.go` (new method on Model)

```go
// resolveFileRefs replaces @path references with file contents.
// @path must be at word boundary (space, start, end).
// Supports relative paths, ~ expansion, absolute paths.
// Files > 50KB are truncated. Binary files are skipped.
func (m Model) resolveFileRefs(text string) string
```

#### Regex pattern

```go
var fileRefPattern = regexp.MustCompile(`(?:^|\s)@(\S+)`)
```

Matches `@path` at start of string or after whitespace. Captures the path without `@`.

#### File resolution logic

1. Extract path from `@path`
2. Expand `~` to home directory
3. Resolve relative paths against working directory
4. Read file content
5. If read fails → keep original `@path` text unchanged
6. If file > 50KB → truncate to 50KB + `[truncated]`
7. If binary (check first 512 bytes for null bytes) → skip, keep original
8. Replace `@path` with `<file path="..." size="N">content</file>`

#### Integration point

In `tui.go`, `handleKey`, `KeyEnter` branch — before sending to agent:

```go
case tea.KeyEnter:
    // ... existing checks ...
    text := m.input
    m.input = ""

    // NEW: resolve @file references
    text = m.resolveFileRefs(text)

    // existing slash command check
    if strings.HasPrefix(text, "/") {
        return m.handleSlashCommand(text)
    }
    // ... send to agent ...
```

#### Files changed

| File | Action |
|------|--------|
| `internal/channels/tui/tui.go` | **EDIT** — Add resolveFileRefs, call in handleKey |
| `internal/channels/tui/fileref_test.go` | **NEW** — Unit tests |

#### Tests

| Test | Input | Expected |
|------|-------|----------|
| Simple file | `@test.txt` | File content injected |
| Inline file | `hello @test.txt world` | Content between words |
| Nonexistent file | `@nope.txt` | Original text preserved |
| Large file | `@big.txt` (>50KB) | Truncated to 50KB |
| Binary file | `@image.png` | Original text preserved |
| Multiple refs | `@a.txt and @b.txt` | Both replaced |
| Tilde path | `~/file.txt` | Home dir expanded |
| No @ | `just text` | Unchanged |

---

## 2. Session Fork

### Goal

User can edit a historical message and re-process from that point. Original session is preserved, new session is created as a branch.

### Expected Behavior

```
Session A (original):
  [0] system: You are helpful.
  [1] user: 查询所有用户
  [2] assistant: SELECT * FROM users;
  [3] user: 加个 WHERE 条件
  [4] assistant: SELECT * FROM users WHERE active = 1;

User clicks message [1], edits to "查询所有订单"
→ Fork creates Session B:
  [0] system: You are helpful.
  [1] user: 查询所有订单
  (agent continues from here)
```

Session A remains unchanged. User can switch between A and B.

### Design

#### Session.Fork method

Location: `core/session/session.go`

```go
// Fork creates a new session with messages up to (excluding) the given index,
// plus the provided new messages. Original session is unchanged.
func (s *Session) Fork(upToIndex int, newMessages ...providers.Message) *Session {
    s.mu.RLock()
    defer s.mu.RUnlock()

    forked := NewSession(generateID())
    if upToIndex > len(s.Messages) {
        upToIndex = len(s.Messages)
    }
    forked.Messages = make([]providers.Message, upToIndex)
    copy(forked.Messages, s.Messages[:upToIndex])
    forked.Messages = append(forked.Messages, newMessages...)
    return forked
}
```

#### TUI integration

Two sub-parts:

**Part A: Message selection mode**

Add a "select mode" to TUI. When user presses `Up` arrow enough times to reach a user message, pressing `Enter` enters edit mode instead of sending.

Implementation:
- Track `selectedIdx int` in Model (default -1 = not selecting)
- `Up` arrow: if at top of viewport, enter select mode → highlight message
- `Enter` in select mode: enter edit mode with that message's text pre-filled
- `Esc`: exit select mode

**Part B: Fork on edit**

When user edits a historical message and presses Enter:
1. Find the index of the original message in session
2. Call `session.Fork(index, editedUserMsg)`
3. Save forked session to store
4. Switch TUI to new session ID
5. Send the edited message to agent

#### Agent.HandleFork method

Location: `core/agent/loop.go`

```go
// HandleFork creates a forked session from an existing one.
// Returns the new session ID.
func (a *Agent) HandleFork(ctx context.Context, originalSessionID string, upToIndex int, newMessage string) (string, error)
```

#### Files changed

| File | Action |
|------|--------|
| `core/session/session.go` | **EDIT** — Add Fork method |
| `core/session/session_test.go` | **NEW** — Fork tests |
| `core/agent/loop.go` | **EDIT** — Add HandleFork |
| `internal/channels/tui/tui.go` | **EDIT** — Add select mode + fork logic |
| `internal/channels/tui/tui_test.go` | **EDIT** — Fork-related tests |

#### Tests

| Test | Description |
|------|-------------|
| Fork preserves prefix | Messages [0..upTo] copied to new session |
| Fork adds new message | New message appended after prefix |
| Fork original unchanged | Original session messages intact |
| Fork new ID | Forked session has different UUID |
| Fork boundary: index 0 | Only system prompt preserved |
| Fork boundary: index = len | Full copy + new message |
| TUI select mode | Up arrow enters select, Enter enters edit |
| TUI fork flow | Edit message → new session created |

---

## 3. Minimal Hook System

### Goal

Execute shell commands before/after tool calls for security audit and logging.

### Expected Behavior

```yaml
# agent.yaml
hooks:
  pre_tool_use:
    command: ./scripts/validate-sql.sh
  post_tool_use:
    command: ./scripts/log-audit.sh
```

```bash
# validate-sql.sh receives JSON on stdin:
{"tool_name":"sql_query","tool_input":{"sql":"SELECT * FROM users"}}

# Returns JSON on stdout:
{"allowed":true}
# or
{"allowed":false,"reason":"DROP not permitted"}
```

### Design

#### ShellHook type

New file: `core/agent/shellhook.go`

```go
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

    cmd := exec.CommandContext(ctx, "sh", "-c", h.Command)
    cmd.Stdin = bytes.NewReader(inputJSON)

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        // Non-zero exit = blocked (safe default)
        return &HookOutput{
            Allowed: false,
            Reason:  fmt.Sprintf("hook command failed: %v, stderr: %s", err, stderr.String()),
        }, nil
    }

    var output HookOutput
    if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
        // Invalid JSON = blocked
        return &HookOutput{
            Allowed: false,
            Reason:  fmt.Sprintf("invalid hook output: %v", err),
        }, nil
    }

    return &output, nil
}
```

#### Config extension

Location: `core/config/config.go`

```go
type AgentDefaults struct {
    // ... existing fields
    PreToolUseHook  string `json:"pre_tool_use_hook,omitempty"`
    PostToolUseHook string `json:"post_tool_use_hook,omitempty"`
}
```

#### YAML config (already supported by feature/config)

```yaml
hooks:
  pre_tool_use:
    command: ./scripts/validate-sql.sh
  post_tool_use:
    command: ./scripts/log-audit.sh
```

`feature/config/config.go` already has `HooksSpec` — just wire it to `AgentDefaults`.

#### Agent integration

Location: `core/agent/loop.go` — in the tool execution loop

```go
// Before tool execution
if a.hooks.PreToolShell != nil {
    output, err := a.hooks.PreToolShell.Execute(&HookInput{
        SessionID: msg.SessionID,
        ToolName:  tc.Name,
        ToolInput: tc.Arguments,
    })
    if err != nil || !output.Allowed {
        reason := "hook error"
        if output != nil {
            reason = output.Reason
        }
        sess.AddMessage(providers.Message{
            Role:       providers.RoleTool,
            Content:    fmt.Sprintf("Tool blocked by policy: %s", reason),
            ToolCallID: tc.ID,
        })
        continue // skip this tool call
    }
}

// After tool execution
if a.hooks.PostToolShell != nil {
    a.hooks.PostToolShell.Execute(&HookInput{
        SessionID:  msg.SessionID,
        ToolName:   tc.Name,
        ToolInput:  tc.Arguments,
        ToolOutput: result.ForLLM,
    })
}
```

#### Hooks struct update

Location: `core/agent/hooks.go`

```go
type Hooks struct {
    // ... existing Go function hooks
    PreToolShell  *ShellHook // NEW: shell command hook
    PostToolShell *ShellHook // NEW: shell command hook
}
```

#### Files changed

| File | Action |
|------|--------|
| `core/agent/shellhook.go` | **NEW** — ShellHook type + Execute |
| `core/agent/shellhook_test.go` | **NEW** — Unit tests |
| `core/agent/hooks.go` | **EDIT** — Add PreToolShell, PostToolShell |
| `core/agent/loop.go` | **EDIT** — Call hooks before/after tool |
| `core/config/config.go` | **EDIT** — Add hook config fields |
| `feature/config/config.go` | **EDIT** — Wire HooksSpec to config |

#### Tests

| Test | Description |
|------|-------------|
| Execute success | Command returns valid JSON |
| Execute blocked | Command returns allowed:false |
| Execute timeout | Command exceeds timeout → blocked |
| Execute crash | Non-zero exit → blocked |
| Execute invalid JSON | stdout not JSON → blocked |
| Execute empty command | No command → allowed (no-op) |
| Agent integration: blocked | Pre hook blocks → tool not executed |
| Agent integration: allowed | Pre hook allows → tool executes |
| Agent integration: post | Post hook receives tool output |
| Config wiring | YAML hooks → AgentDefaults hook fields |

---

## Implementation Order

1. **@file Reference** (lowest risk, pure additive)
2. **Minimal Hook System** (medium risk, additive)
3. **Session Fork** (highest complexity, modifies session model)

Each phase: write tests → implement → verify → commit.

---

## Test Strategy

All three improvements follow TDD:

- **Unit tests** first (red → green → refactor)
- **Existing tests** must continue passing
- **Integration tests** for agent-level hook behavior
- **Coverage target**: 80%+ for new code
