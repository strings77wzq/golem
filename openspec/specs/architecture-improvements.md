# Spec: Golem Architecture Improvements (5 Areas)

> Version: 1.0
> Date: 2026-06-15
> Status: Draft — awaiting review before TDD implementation

---

## Overview

Five architecture improvements to Golem, all Option A (lightweight/recommended):

1. **Agent Decomposition** — Extract ReAct loop from Agent
2. **Provider Fallback** — Factory-level automatic failover
3. **YAML Config** — Declarative agent configuration
4. **HistoryManager Cleanup** — Delete dead code, merge into context.Manager
5. **TUI Command Framework** — Slash command registry

---

## 1. Agent Decomposition

### Problem

`agent.go` (165 lines) + `loop.go` (823 lines) = 988 lines in one package. `processMessage` is ~250 lines handling:
- Session creation/loading
- System prompt injection
- Planning mode decision
- ReAct loop (tool call → execute → repeat)
- Token usage tracking
- Streaming orchestration

### Design

Extract the ReAct loop into a standalone `ReActLoop` struct. Agent becomes a thin message router.

```
Agent (message router)
  ├── ReActLoop (core loop logic)
  │     ├── contextManager
  │     ├── toolRegistry
  │     └── providerFactory
  ├── sessionStore
  └── hooks
```

### New file: `core/agent/react.go`

```go
// ReActLoop executes the Reason+Act cycle: call LLM, dispatch tools, repeat.
type ReActLoop struct {
    contextManager *context.Manager
    toolRegistry   *tools.Registry
    providerFactory *providers.Factory
    logger         logger.Logger
    hooks          *Hooks
    tracker        *usage.Tracker
    maxIterations  int
}

type LoopConfig struct {
    ContextManager *context.Manager
    ToolRegistry   *tools.Registry
    ProviderFactory *providers.Factory
    Logger         logger.Logger
    Hooks          *Hooks
    Tracker        *usage.Tracker
    MaxIterations  int
}

func NewReActLoop(cfg LoopConfig) *ReActLoop

// Run executes the ReAct loop for a single user message.
// Returns: final response content, token usage, error
func (r *ReActLoop) Run(
    ctx context.Context,
    sess *session.Session,
    model string,
    toolDefs []tools.ToolDefinition,
    streamFinal bool,
    onToken func(string),
    emit func(bus.OutboundMessage),
    sessionID string,
) (string, *bus.TokenUsage, error)
```

### Changes to `agent.go`

Agent keeps:
- `HandleMessage`, `HandleMessageStream`, `HandleMessageStreamWithProgress`, `HandleCompact`
- `Start` (bus listener)
- Session management (create/load)
- System prompt injection

Agent delegates to:
- `reactLoop.Run(...)` for the actual LLM + tool cycle

### Files changed

| File | Action |
|------|--------|
| `core/agent/react.go` | **NEW** — ReActLoop struct + Run method |
| `core/agent/agent.go` | **EDIT** — Remove processMessage, delegate to reactLoop |
| `core/agent/loop.go` | **DELETE** — All logic moves to react.go |
| `core/agent/agent_test.go` | **EDIT** — Update setup to use ReActLoop |
| `core/agent/react_test.go` | **NEW** — Unit tests for ReActLoop |

### Migration path

- `processMessage` → `reactLoop.Run`
- `processWithPlan` stays in agent (planning is agent-level orchestration)
- `processMessageFallback` → merged into `reactLoop.Run` (the fallback IS the main loop)
- `buildToolErrorMessage` → moves to react.go (tool-level concern)

---

## 2. Provider Fallback

### Problem

If the primary provider returns an error (rate limit, auth failure, network), the entire agent fails. No automatic failover.

### Design

Add `GetProviderForModelWithFallback` to Factory. Config gets a `FallbackModels` field.

```go
// In core/providers/factory.go

// GetProviderForModelWithFallback tries the primary model, then fallbacks in order.
// Returns: provider, resolved modelName, which model was used, error
func (f *Factory) GetProviderForModelWithFallback(
    model string,
    fallbackModels []string,
) (LLMProvider, string, string, error)
```

### Config change

```go
// In core/config/config.go

type AgentDefaults struct {
    ModelName      string   `json:"model_name"`
    MaxTokens      int      `json:"max_tokens"`
    SystemPrompt   string   `json:"system_prompt"`
    FallbackModels []string `json:"fallback_models,omitempty"` // NEW
}
```

Example config:
```json
{
  "agents": {
    "defaults": {
      "model_name": "openai/gpt-4o",
      "fallback_models": ["anthropic/claude-3-haiku", "ollama/qwen3"]
    }
  }
}
```

### Agent integration

In `react.go` (the new ReActLoop), replace:
```go
provider, modelName, err := a.providerFactory.GetProviderForModel(model)
```
with:
```go
provider, modelName, usedModel, err := r.providerFactory.GetProviderForModelWithFallback(
    model, r.fallbackModels,
)
```

### Retry logic

- Try primary model first
- On error: log warning, try next fallback
- Max 3 fallback attempts (configurable)
- Each fallback is a single retry, no exponential backoff (LLM calls are slow anyway)

### Files changed

| File | Action |
|------|--------|
| `core/providers/factory.go` | **EDIT** — Add `GetProviderForModelWithFallback` |
| `core/providers/factory_test.go` | **NEW** — Fallback tests |
| `core/config/config.go` | **EDIT** — Add `FallbackModels` field |
| `core/agent/react.go` | **EDIT** — Use fallback method |

---

## 3. YAML Config

### Problem

Golem's config is JSON-only (`~/.golem/config.json`). No declarative agent definition. Docker Agent's YAML config is a key differentiator.

### Design

New module `feature/config/` that parses YAML agent definitions. CLI flag `--config agent.yaml` activates it.

### YAML Schema

```yaml
# agent.yaml — declarative agent definition
version: 1

agent:
  model: openai/gpt-4o
  fallback_models:
    - anthropic/claude-3-haiku
  system_prompt: |
    You are a database assistant. Help users query and analyze their data.
  max_tokens: 8192
  max_tool_iterations: 25

tools:
  - name: sql_query
    enabled: true
  - name: sql_schema
    enabled: true
  - name: think
    enabled: true
  - name: exec
    enabled: false  # disable shell access

database:
  path: ./myapp.db
  # Future: postgres, mysql, redis support

mcp:
  - name: github
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]

hooks:
  pre_tool_use:
    command: ./scripts/validate-sql.sh
  post_tool_use:
    command: ./scripts/log-audit.sh
```

### New package: `feature/config/`

```go
package config

// AgentConfig represents a YAML agent definition.
type AgentConfig struct {
    Version int         `yaml:"version"`
    Agent   AgentSpec   `yaml:"agent"`
    Tools   []ToolSpec  `yaml:"tools,omitempty"`
    Database *DBSpec    `yaml:"database,omitempty"`
    MCPServers []MCPDef `yaml:"mcp,omitempty"`
    Hooks   *HooksSpec  `yaml:"hooks,omitempty"`
}

type AgentSpec struct {
    Model           string   `yaml:"model"`
    FallbackModels  []string `yaml:"fallback_models,omitempty"`
    SystemPrompt    string   `yaml:"system_prompt"`
    MaxTokens       int      `yaml:"max_tokens,omitempty"`
    MaxToolIterations int    `yaml:"max_tool_iterations,omitempty"`
}

type ToolSpec struct {
    Name    string `yaml:"name"`
    Enabled bool   `yaml:"enabled"`
}

type DBSpec struct {
    Path string `yaml:"path"`
}

type MCPDef struct {
    Name    string   `yaml:"name"`
    Command string   `yaml:"command,omitempty"`
    Args    []string `yaml:"args,omitempty"`
    URL     string   `yaml:"url,omitempty"`
}

type HooksSpec struct {
    PreToolUse  *HookDef `yaml:"pre_tool_use,omitempty"`
    PostToolUse *HookDef `yaml:"post_tool_use,omitempty"`
}

type HookDef struct {
    Command string `yaml:"command"`
}

// LoadYAML parses a YAML agent config file.
func LoadYAML(path string) (*AgentConfig, error)

// ToCoreConfig converts AgentConfig to the core config.Config format.
func (c *AgentConfig) ToCoreConfig() *config.Config
```

### CLI integration

In `cmd/golem/main.go`, the `--config` flag already exists. When the file is `.yaml`/`.yml`, use the YAML loader:

```go
// In runAgent()
configPath, _ := cmd.Flags().GetString("config")
if strings.HasSuffix(configPath, ".yaml") || strings.HasSuffix(configPath, ".yml") {
    // YAML path — use feature/config loader
    agentCfg, err := featureconfig.LoadYAML(configPath)
    // ... wire up from agentCfg
} else {
    // JSON path — existing behavior
    cfg, err := loadConfig(cmd)
}
```

### Dependency

Requires `gopkg.in/yaml.v3` — add to go.mod.

### Files changed

| File | Action |
|------|--------|
| `feature/config/config.go` | **NEW** — YAML schema + loader |
| `feature/config/config_test.go` | **NEW** — YAML parsing tests |
| `cmd/golem/main.go` | **EDIT** — Route YAML vs JSON config |
| `cmd/golem/run.go` | **EDIT** — Accept AgentConfig parameter |
| `go.mod` | **EDIT** — Add yaml.v3 dependency |

---

## 4. HistoryManager Cleanup

### Problem

`session.HistoryManager` and `context.Manager` both do token estimation and message compression. The agent only uses `contextManager.BuildContext()`. `HistoryManager` is dead code.

### Current usage

- `cmd/golem/main.go:172` — `history := session.NewHistoryManager(cfg.Agents.Defaults.MaxTokens)`
- `cmd/golem/main.go:173` — Passed to `agent.New(..., history, ...)`
- `agent.New()` stores it but **never uses it** — `processMessage` calls `a.contextManager.BuildContext()`
- `embedded.go:60` — Same pattern

### Design

1. Delete `HistoryManager` struct and its methods
2. Merge `EstimateTokens` logic into `context.DefaultTokenEstimator` (already exists and is better)
3. Remove `history` parameter from `agent.New()`
4. Update all callers

### Files changed

| File | Action |
|------|--------|
| `core/session/history.go` | **DELETE** |
| `core/session/history_test.go` | **DELETE** |
| `core/agent/agent.go` | **EDIT** — Remove `historyManager` field and `history` param from `New()` |
| `core/agent/agent_test.go` | **EDIT** — Remove history from setup |
| `core/agent/embedded.go` | **EDIT** — Remove history from `NewFromConfig` |
| `cmd/golem/main.go` | **EDIT** — Remove history creation |
| `cmd/golem/run.go` | **EDIT** — Remove history from agent creation |

### Verification

After deletion, `go build ./...` and `go test ./...` must pass with zero references to `HistoryManager`.

---

## 5. TUI Command Framework

### Problem

Slash commands are hardcoded in `handleSlashCommand`'s switch statement. Adding `/sessions`, `/model`, `/cost` will bloat the function.

### Design

Define a `SlashCommand` interface and a registry. Commands self-register.

### New file: `internal/channels/tui/command.go`

```go
package tui

import "context"

// SlashCommand is a TUI slash command.
type SlashCommand interface {
    Name() string        // e.g. "/compact"
    Description() string // help text
    Execute(ctx context.Context, m *Model) (tea.Cmd, error)
}

// CommandRegistry manages available slash commands.
type CommandRegistry struct {
    commands map[string]SlashCommand
}

func NewCommandRegistry() *CommandRegistry

// Register adds a command. Panics on duplicate names.
func (r *CommandRegistry) Register(cmd SlashCommand)

// Get returns a command by name, or nil if not found.
func (r *CommandRegistry) Get(name string) SlashCommand

// List returns all registered commands (for /help).
func (r *CommandRegistry) List() []SlashCommand
```

### Built-in commands

Each command is a small struct implementing `SlashCommand`:

```go
// internal/channels/tui/cmd_compact.go
type compactCmd struct{}
func (c compactCmd) Name() string { return "/compact" }
func (c compactCmd) Description() string { return "Compress conversation history" }
func (c compactCmd) Execute(ctx context.Context, m *Model) (tea.Cmd, error) {
    return m.doCompact(), nil
}

// internal/channels/tui/cmd_clear.go
type clearCmd struct{}
func (c clearCmd) Name() string { return "/clear" }
func (c clearCmd) Description() string { return "Clear conversation history" }
func (c clearCmd) Execute(ctx context.Context, m *Model) (tea.Cmd, error) {
    m.messages = nil
    m.lastError = ""
    if m.ready {
        m.viewport.SetContent(m.buildTranscript())
    }
    return nil, nil
}

// internal/channels/tui/cmd_help.go
type helpCmd struct{ registry *CommandRegistry }
func (c helpCmd) Name() string { return "/help" }
func (c helpCmd) Description() string { return "Show available commands" }
func (c helpCmd) Execute(ctx context.Context, m *Model) (tea.Cmd, error) {
    // Build help text from registry.List()
    ...
}
```

### TUI integration

```go
// In tui.go, Model gets a registry field:
type Model struct {
    ...
    commands *CommandRegistry
}

// New() initializes with built-in commands:
func New(ctx context.Context, sessionID string, handler MessageHandler) Model {
    cmds := NewCommandRegistry()
    cmds.Register(compactCmd{})
    cmds.Register(clearCmd{})
    cmds.Register(helpCmd{registry: cmds})
    cmds.Register(quitCmd{})
    ...
}

// handleSlashCommand becomes:
func (m Model) handleSlashCommand(text string) (tea.Model, tea.Cmd) {
    parts := strings.Fields(text)
    name := parts[0]
    args := parts[1:]

    cmd := m.commands.Get(name)
    if cmd == nil {
        // unknown command message
        return m, nil
    }
    return cmd.Execute(m.ctx, &m)
}
```

### Files changed

| File | Action |
|------|--------|
| `internal/channels/tui/command.go` | **NEW** — SlashCommand interface + registry |
| `internal/channels/tui/cmd_compact.go` | **NEW** — /compact command |
| `internal/channels/tui/cmd_clear.go` | **NEW** — /clear command |
| `internal/channels/tui/cmd_help.go` | **NEW** — /help command |
| `internal/channels/tui/cmd_quit.go` | **NEW** — /quit command |
| `internal/channels/tui/tui.go` | **EDIT** — Use registry in handleSlashCommand |
| `internal/channels/tui/command_test.go` | **NEW** — Registry + command tests |

---

## Implementation Order

Recommended order based on dependencies and risk:

1. **HistoryManager Cleanup** (lowest risk, zero behavior change)
2. **Agent Decomposition** (medium risk, pure refactor)
3. **Provider Fallback** (low risk, additive)
4. **TUI Command Framework** (low risk, refactor + extend)
5. **YAML Config** (highest complexity, new dependency)

Each phase: TDD — write tests first, then implement, then verify.

---

## Test Strategy

All improvements follow TDD:

- **Unit tests** for each new struct/function
- **Integration tests** for agent loop with fallback
- **Existing tests** must continue passing (no regressions)
- **Coverage target**: 80%+ for new code

### Key test scenarios

| Improvement | Key Tests |
|-------------|-----------|
| Agent Decomposition | ReActLoop.Run with mock provider, tool calls, max iterations |
| Provider Fallback | Primary fails → fallback succeeds, all fail → error, no fallbacks → primary only |
| YAML Config | Parse valid YAML, parse invalid YAML, merge with defaults, CLI flag routing |
| HistoryManager | Verify deletion doesn't break anything, context.Manager still works |
| TUI Commands | Register/get/list commands, execute each command, unknown command handling |
