# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**Golem** is a Go-native AI database agent. Single binary, three modes: `agent` (interactive), `mcp-server` (MCP for Claude Code/Cursor), `gateway` (HTTP, OpenAI-compatible on `:18790`). Connects to SQLite/PostgreSQL/MySQL/Redis/Qdrant, understands schemas, enforces a 3-layer safety model (read-only default, WHERE enforcement, rollback SQL).

Go 1.25+, pure Go, `CGO_ENABLED=0` always. SQLite via `modernc.org/sqlite`.

## Commands

```bash
make build              # build/golem (CGO_ENABLED=0, -trimpath, -ldflags "-s -w")
make build-all          # cross-compile linux/darwin amd64/arm64
make test               # go test ./... -v -race
make lint               # golangci-lint run
make vet                # go vet ./...
make fmt                # gofmt -s -w .
make check              # deps + vet + test
make e2e                # tests/e2e (requires Ollama with qwen3:0.5b)
make clean

# Single package / test
go test ./core/agent/...
go test -run TestName ./pkg/...
go test -bench=. -benchmem ./feature/rag/...

# Run
./build/golem init                 # first-run setup wizard
./build/golem agent                # TUI (auto on TTY)
./build/golem agent -m "query"     # one-shot
./build/golem gateway              # HTTP on :18790
./build/golem mcp-server --db x.db
./build/golem debug tools|config
./build/golem config validate
./build/golem status
```

## Architecture — Layer Rules (enforced by Go imports)

```
cmd/golem/         composition root; imports ALL layers
                   feature wiring (skills, mcp, rag, etc.) lives here as
                   <feature>-adapter.go files in package main
internal/wiring/   dependency creation for core+foundation; imports core + foundation ONLY
internal/*         adapters (channels/tui, channels/cli, channels/telegram,
                   gateway, security); imports core + foundation ONLY
core/*             domain logic; imports foundation ONLY
                   NEVER imports internal/, feature/, cmd/
feature/*          optional modules (mcp, rag, skills, memory, routing,
                   health, config); imports core + foundation ONLY
                   wired via CLI flags in cmd/golem/<feature>-adapter.go
foundation/*       stdlib ONLY
                   exceptions: mattn/go-isatty (term/), modernc.org/sqlite (store/)
                   includes metrics (Counter/Gauge/Histogram/Registry)
```

Circular imports are always wrong. Use interfaces in `core/` to break cycles.

## Hard Constraints

- **`CGO_ENABLED=0` always.** No CGO. Ever.
- **`LLMProvider` / `StreamingProvider` interfaces in `core/providers/types.go` are frozen.** New adapters conform; signatures don't change.
- **Bubble Tea isolated to `internal/channels/tui/`.** No other package imports it. No `WithoutMouseCellMotion` (doesn't exist in v1.3.10). No Alt+key shortcuts (breaks Termux).
- **Tools sorted alphabetically in registry.** `ListTools()` / `ListDefinitions()` alphabetical order is intentional — maximizes LLM KV-cache reuse.
- **Streaming disabled during tool calls.** `canStream := ok && streamFinal && len(toolDefs) == 0`. Mid-stream tool calls would require buffering anyway.
- **Error handling:** return errors, never panic in library code. Wrap with `fmt.Errorf("context: %w", err)`.

### 代码验证原则（CRITICAL — 每次分析必须遵守）

**代码是唯一可信来源。文档可能过时或不准确。所有代码分析必须基于 Read/grep 的已验证输出。**

执行顺序：
1. 先读代码（Read/grep），获取实际实现
2. 再读文档（README/AGENTS.md/Spec），理解设计意图
3. 如果代码和文档冲突，以代码为准，并在回答中标注差异

禁止行为：
- ❌ 只读文档就声称"实现了 X 功能"
- ❌ 引用文档中的功能描述而没有代码证据
- ❌ 基于过时文档做决策

必须行为：
- ✅ 引用的功能/接口/路由路径必须来自 Read/grep 的已验证输出
- ✅ 如果代码和文档不一致，明确指出差异
- ✅ 回答中标注证据来源（文件路径:行号）

## Key Interfaces

```go
// core/providers/types.go
type LLMProvider interface {
    Chat(ctx, messages, toolDefs, model, opts) (*LLMResponse, error)
    Name() string
}
type StreamingProvider interface {
    LLMProvider
    ChatStream(ctx, messages, toolDefs, model, opts, onToken) (*LLMResponse, error)
}

// core/agent/agent.go — how channels talk to the agent
type MessageHandler interface {
    HandleMessage(ctx, sessionID, message) (string, error)
    HandleMessageStream(ctx, sessionID, message, tokens chan<- string) error
}

// core/tools/tool.go — tool contract
type Tool interface {
    Name() string
    Description() string
    Parameters() []ToolParameter
    Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}
```

## Agent ReAct Loop

Message flow: `HandleMessage()` → `processMessage()` → `preProcess()` → `reactLoop()` → `invokeProvider()` → tool execution → response.

**preProcess gate** (`core/agent/loop.go`): Before the ReAct loop, checks:
1. BeforeMessage hooks
2. Planning mode (if enabled and `isComplexTask()` — message > 30 words)
3. Auto-compaction at 80% of token budget

If planning handles the message, returns early with `handled=true` and skips the ReAct loop.

**ReAct loop** runs `maxToolIterations` (default 25) times:
1. `contextManager.BuildContext()` — assembles messages within token budget
2. `resolveProvider()` — uses router if set, otherwise fallback chain
3. `invokeProvider()` — streaming only when `len(toolDefs) == 0`
4. If `resp.ToolCalls == 0`: save session, track cost, return
5. If `resp.ToolCalls != 0`: `runToolExecution()` then loop again

**Tool execution pipeline** (`core/agent/loop_helpers.go`):
1. **PreToolShell hooks** — run sequentially per tool call; if blocked, tool gets "blocked by policy" result
2. **Parallel execution** — remaining tools via `errgroup.Group`
3. **PostToolShell hooks** — run after execution with access to output

**Dual-channel output**: `ToolResult.ForLLM` always appended to session as `RoleTool`; `ToolResult.ForUser` emitted to event stream (unless `Silent: true`).

**Plan/Reflect architecture**: When planning enabled, complex tasks are decomposed by `planner.Decompose()`. Each step runs a mini ReAct loop (max 5 iterations). `Reflector` checks if output matches expected keywords. Failed steps trigger `planner.Revise()`.

## Context Management

Two-tier system:
1. **`context.Manager`** — per-LLM-call context assembly (token budgeting, prompt building, message compression)
2. **`agent.Compactor`** — cross-conversation session compaction (LLM-driven summarization)

**Token budget allocation** (`core/context/budget.go`): System 20%, Tools 30%, History 50%. Unused system budget flows to history.

**CJK-aware token estimation** (`core/context/manager.go`): CJK = 2 chars/token, ASCII = 4 chars/token, tool calls = 50 tokens each. Covers Chinese (Han), CJK Unified Extensions, Hiragana, Katakana, Hangul.

**Three-stage compression** (`core/context/compressor.go`):
1. Check if everything fits within budget
2. Keep recent 4 messages untouched, truncate oversized tool outputs in older messages (threshold 1000 tokens, max 2000)
3. Drop oldest messages preserving tool chains atomically (assistant + all tool results as a batch)

**Tool chain atomicity**: When dropping old messages, tool batches (assistant with tool_calls + corresponding tool results) are treated as atomic units. If a batch cannot fit, it is replaced by a summary. Orphaned tool results are removed.

**Auto-compaction**: Triggered in `preProcess()` when estimated tokens exceed 80% of budget. `Compactor` uses LLM to summarize old messages, with fallback to simple truncation.

## Database Safety Model

Three layers enforced in `core/tools/database/sql_query.go` and `core/security/gates.go`:

**Layer 1: PermissionChecker** — Four levels: `PermRead`, `PermWrite`, `PermDelete`, `PermAdmin`. SQLQueryTool defaults to `PermRead`. Operations classified by `classifyOperation()` which normalizes SQL (strips comments, collapses whitespace, rejects multi-statement with semicolons). Denied ops: `DROP DATABASE`, `TRUNCATE`, `DROP TABLE`, `ALTER USER`.

**Layer 2: QualityGate** — Applied to write operations (PermWrite+). Checks: `RequireWhere: true` (DELETE/UPDATE without WHERE blocked), `MaxAffectedRows: 100` (flagged for confirmation), `RequireBackup: true` (rollback SQL generated).

**Layer 3: RollbackSQL generation** — For DELETE: `INSERT INTO table_backup SELECT * FROM table WHERE ...`. For UPDATE: `UPDATE table SET ... WHERE ...` (with old values if available).

**SQL normalization** (`core/tools/database/sql_query.go`): `normalizeSQL()` strips block comments (`/* */`), single-line comments (`--` and `#`), collapses whitespace, rejects multi-statement SQL (any semicolon). Conservative: if it cannot classify, returns empty string which maps to `PermAdmin` (denied).

**Soft error handling**: Permission/safety gate failures return `ToolResult{IsError: true}` (not Go `error`), allowing the LLM to see the error and adjust rather than terminating the ReAct loop.

**Audit trail**: `AuditEntry` records operation, database, table, SQL, affected rows, rollback SQL, status, executed by. Audited on both denied and successful operations.

## Providers

Two protocol adapters cover all vendors: `openai` (OpenAI-compatible: DeepSeek, Kimi, GLM, MiniMax, Qwen, MiMo, Ollama, vLLM, OpenRouter, LiteLLM) and `anthropic`. Add a vendor by creating `core/providers/<vendor>/vendor.go` implementing the interface, registering in `cmd/golem/main.go`:`registerProviders()`, and adding a preset in `cmd/golem/onboard.go`:`providerPresets`.

## Feature Modules

All optional, disabled by default, enabled via CLI flags. Wiring pattern:
1. Adapter file `cmd/golem/<feature>-adapter.go`
2. CLI flag in `cmd/golem/main.go`
3. Register feature-provided tools into global registry only when flag enabled
4. Never modify `core/` or `internal/` to support a feature

Module rules:
- `feature/mcp/` — MCP logic stays here. STDIO only (HTTP planned). Tools registered with `mcp_` prefix. Lazy start.
- `feature/rag/` — RAG logic stays here. TF-IDF + cosine by default (no external embeddings). `rag_retrieve` tool registered only when `--rag` enabled.
- `feature/skills/` — Skills logic stays here. Built-ins in `feature/skills/builtins/`, auto-registered when `--skills` list provided. No side effects on agent state.

## Config System

Dual format:
1. **Core JSON** (`~/.golem/config.json`) — agents defaults, gateway settings, telegram settings, `model_list` array
2. **YAML v2** (`feature/config/`) — declarative agent configuration. Converts to core config via `ToCoreConfig()`. Supports `${ENV_VAR}` expansion and type-based ToolSpec.

**Hot reload** (`core/config/reload.go`): `Reloader` uses `atomic.Pointer[Config]` for lock-free reads. `WatchSIGHUP()` listens for SIGHUP signal and calls `Reload()`, triggering registered `OnReload` callbacks.

## Session Persistence

Two-level storage abstraction:
- `foundation/store.Store` — low-level key-value interface (`SessionRecord` with JSON-encoded `Messages`)
- `core/session.SessionStore` — higher-level interface (`Get`, `Save`, `Delete`, `List` returning `*Session`)
- `SQLiteAdapter` bridges the two: converts `Session` to/from `SessionRecord` via JSON marshal/unmarshal

**SQLite PRAGMAs** (`foundation/store/sqlite.go`): `journal_mode=WAL` (concurrent read/write), `foreign_keys=ON`, `busy_timeout=5000` (5s wait on lock).

**Session operations**: Thread-safe via `sync.RWMutex`. `Fork()` creates new session by copying messages up to an index. JSON export/import with version field.

## Streaming Flow

```
TUI handleKey → startStream (goroutine) ─► Agent.HandleMessageStream
                tokens chan (buf 64 TUI / 32 gateway)           │
                ◄── onToken(tok) → tokens <- tok                │
                waitNextToken (recursive tea.Cmd)     defer close(tokens)
```

Non-streaming fallback: if `streamed==false` after processing, entire content sent as one token chunk.

## Trace IDs

Every message gets a `trace-` prefixed hex ID attached to context (`core/agent/trace.go`). Used for observability correlation across logs and metrics.

## Testing

- 42 packages, 82.5% coverage, race-detector mandatory
- E2E in `tests/e2e/` requires Ollama with `qwen3:0.5b`
- **E2E isolation**: Separate `go.mod`, no internal imports (only CLI/gateway interfaces), sentinel errors with `errors.Is()`, Ollama auto-detection with clean skip
- Tools are pure functions — unit-test without a real agent
- `MockTool` accepts `ExecuteFn` callback for configurable test behavior
- `make e2e` runs the E2E suite

## Commit / PR

Conventional Commits: `<type>(<scope>): <description>` — types: `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `ci`.

## Workflow (from AGENTS.md — mandatory)

```
Brainstorm → Spec → TDD → Verify → Review → Commit → Push → Archive
```

Every implementation task needs a spec first. Exceptions: trivial one-liners, typos, import reordering. Thinking methodology: First Principles / Reverse Thinking / Systems Thinking before any significant change.

### Commit discipline

- One logical unit per commit (Conventional Commits format)
- `git diff --name-only` must match task's declared file boundary
- No unrelated changes in the same commit
- Run `go test -race ./...` before every commit — green only

### Push strategy

- Push after each commit if CI is green
- Or batch push at end of work session if multiple related commits
- **After push: MUST poll CI status (`gh pr checks` or `gh run watch`) until all checks pass.** CI failure = stop, analyze root cause, fix, re-push, re-poll. Never claim "done" while CI is red.

> The commit → push → CI-poll loop is codified as the global `/git-commit-push-ci` skill (conflict check, atomic commit, push, poll until green), decoupled from the development workflow.

## Reference

- Full architecture, guardrails, and decision log: `AGENTS.md` (read this)
- Contributing: `CONTRIBUTING.md`
- OpenSpec changes: `openspec/changes/` (proposals), `openspec/specs/` (stable)
- Study guides: `docs/study/`
