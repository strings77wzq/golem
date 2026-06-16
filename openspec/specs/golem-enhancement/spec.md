# Golem Enhancement SDD — Docker-Agent Inspired Improvements

## [S1] Problem

Golem v0.5.0 has strong foundations (zero-CGO binary, database safety model, strict layering, KV-cache optimization) but lacks discoverability, configuration ergonomics, and advanced agent capabilities that Docker Agent demonstrates at scale. Users cannot easily discover golem's tools, must pass 8+ CLI flags per invocation, and the agent loop has accumulated technical debt (80% code duplication in loop.go).

**Root causes:**
1. CLI-flag-only configuration creates cognitive overhead and prevents "declare once, run everywhere"
2. TUI has 5 slash commands vs Docker Agent's 20+, limiting discoverability
3. Single-agent architecture limits complex database workflows
4. String-truncation context management loses important information
5. No provider resilience (no retry, no fallback routing)

## [S2] Solution Overview

Phased improvement plan across 5 phases, each independently deployable:

| Phase | Goal | Risk | ROI |
|-------|------|------|-----|
| P1: Discoverability | Users can find and understand golem's capabilities | Low | High |
| P2: Config Unification | YAML becomes first-class, CLI flags become overrides | Medium | High |
| P3: TUI + Dedup | Richer TUI + loop.go technical debt removal | Low | Medium |
| P4: Agent Capabilities | Multi-agent, LLM compaction, provider resilience | Medium | High |
| P5: RAG Enhancement | BM25 + hybrid retrieval + better chunking | Medium | High |

## [S3] Phase 1 — Discoverability & UX (1 week)

### Task 1.1: `golem debug` Subcommand
**Files:** `cmd/golem/debug.go` (new), `cmd/golem/main.go` (modify)
**Spec:**
- `golem debug tools` — lists all registered tools with name, description, parameters (JSON schema)
- `golem debug config` — loads config and prints YAML with API keys masked as `***`
- Must use `yaml.Marshal` for output (not JSON) to align with YAML-first direction
**TDD:**
- `cmd/golem/debug_test.go`: mock registry with 3 tools, verify output contains tool names
- `cmd/golem/debug_test.go`: mock config with API key, verify key is masked
**Estimated:** ~80 lines

### Task 1.2: `golem status` Enhancement
**Files:** `cmd/golem/status.go` (modify)
**Spec:**
- Append tool list (count + names) after existing status output
- Show feature module status: MCP (enabled/disabled + server count), RAG (enabled/disabled), Memory (enabled/disabled), Skills (enabled/disabled + count)
**TDD:**
- Extend `cmd/golem/status_test.go`: verify output contains "Tools:" section
- Verify feature status lines appear when flags are set
**Estimated:** ~50 lines

### Task 1.3: `golem init` YAML Default
**Files:** `cmd/golem/onboard.go` (modify), `cmd/golem/config.go` (modify)
**Spec:**
- Default output format changes from JSON to `~/.golem/agent.yaml`
- Preserve `--format json` for backward compatibility
- After writing config, validate connectivity:
  - Ollama: check `localhost:11434` reachability
  - Others: verify env var exists (OPENAI_API_KEY etc.)
- Print "next steps": `golem agent -m "hello"`
**TDD:**
- Extend `cmd/golem/onboard_test.go`: mock stdin + filesystem, verify YAML output
- Verify next-steps text appears in output
**Estimated:** ~60 lines

### Task 1.4: `golem config validate`
**Files:** `cmd/golem/config.go` (modify)
**Spec:**
- Load config → validate required fields → validate model format → report results
- Exit 0 on success, exit 1 on validation failure
- Output: `✓ valid` or `✗ <field>: <reason>` per issue
**TDD:**
- `cmd/golem/config_test.go`: valid config passes, missing model fails, invalid provider fails
**Estimated:** ~30 lines

## [S4] Phase 2 — Config Unification (2 weeks)

### Task 2.1: YAML Schema Extension
**Files:** `feature/config/config.go` (modify)
**Spec:**
```yaml
version: 2
agent:
  model: openai/gpt-4o
  fallback_models:
    - anthropic/claude-3-5-sonnet
  max_tokens: 16384
  system_prompt: "You are a database expert."
tools:
  - type: database
    path: ./data.db
  - type: mcp
    command: npx
    args: ["-y", "@modelcontextprotocol/server-duckduckgo"]
  - type: rag
    docs: ["./docs"]
  - type: memory
    path: ./memory.db
  - type: infra
hooks:
  pre_tool_use:
    command: ./scripts/validate.sh
  post_tool_use:
    command: ./scripts/log.sh
```
- `AgentConfig` struct gains `Tools []ToolConfig`, `Hooks HookConfig`, `FallbackModels []string`
- `ToCoreConfig()` must map new fields to existing `config.Config` structure
- JSON config path remains supported via `--config` flag
**TDD:**
- `feature/config/config_test.go`: parse full YAML, verify all fields populated
- `feature/config/config_test.go`: `ToCoreConfig()` maps tools/hooks/fallback correctly
- Backward compat: existing JSON configs still parse
**Estimated:** ~200 lines

### Task 2.2: Declarative Tool Loading
**Files:** `cmd/golem/adapter.go` (modify), `cmd/golem/toolset_loader.go` (new)
**Spec:**
- `LoadToolsetsFromConfig(cfg *AgentConfig, registry *tools.Registry)` reads `tools:` list
- Each `ToolConfig` type maps to a loading function:
  - `database` → `wiring.BuildDBTools(path)`
  - `mcp` → `loadMCPTools(ctx, jsonConfig, registry, log)`
  - `rag` → `loadRAGTools(ctx, cfg, ragConfig, registry)`
  - `memory` → `loadMemoryTools(ctx, path, registry, log)`
  - `infra` → `wiring.RegisterInfraTools(registry)`
  - `filesystem`, `exec`, `think`, `todo`, `fetch`, `websearch` → direct registration
- CLI flags (`--mcp`, `--rag`, etc.) still work as overrides
**TDD:**
- `cmd/golem/toolset_loader_test.go`: mock each toolset type, verify registration
- Error case: unknown tool type returns descriptive error
**Estimated:** ~150 lines

### Task 2.3: Provider Registry (Replace switch-case)
**Files:** `internal/wiring/providers.go` (modify), `core/providers/factory.go` (modify)
**Spec:**
- Replace `switch vendor` with `registry.Register(name, constructor)` pattern
- Each provider package registers itself via `RegisterProvider(name, factory)` called from `cmd/golem/main.go`
- `cmd/golem/main.go` explicitly calls each provider's registration function (NOT init())
- Preserves `feature/` module isolation — providers only register when explicitly wired
**TDD:**
- `core/providers/factory_test.go`: register custom provider, verify lookup
- `core/providers/factory_test.go`: duplicate registration returns error
**Estimated:** ~100 lines

## [S5] Phase 3 — TUI + Loop Dedup (1 week)

### Task 3.1: New Slash Commands
**Files:** `internal/channels/tui/cmd_model.go` (new), `cmd_new.go` (new), `cmd_sessions.go` (new), `cmd_tools.go` (new)
**Spec:**
- `/model` — lists available models, user selects via number, updates session config
- `/new` — clears session history, starts fresh conversation
- `/sessions` — lists recent sessions with ID, timestamp, message count
- `/tools` — lists registered tools with descriptions
- Each command implements `Command` interface from existing `CommandRegistry`
**TDD:**
- Each `cmd_*_test.go`: verify command name, description, parse logic, output format
- Integration: `tui_test.go` verifies commands register correctly
**Estimated:** ~200 lines (50 per command)

### Task 3.2: Agent Loop Deduplication
**Files:** `core/agent/loop.go` (modify)
**Spec:**
- Extract `processMessageCore()` containing shared logic (context build, LLM call, tool dispatch, response)
- `processMessage()` calls `processMessageCore()` with bus publish callback
- `processMessageFallback()` calls `processMessageCore()` without streaming
- Target: loop.go from ~1016 lines to ~700 lines
- No behavioral changes — pure refactor
**TDD:**
- All 84 existing test files must pass unchanged
- Add `core/agent/loop_dedup_test.go`: verify both paths produce identical results for same input
- Run `go test -race ./core/agent/...`
**Estimated:** Net -300 lines

## [S6] Phase 4 — Agent Capabilities (3 weeks)

### Task 4.1: LLM-Driven Session Compaction
**Files:** `core/agent/compactor.go` (new), `core/agent/loop.go` (modify)
**Spec:**
- `Compactor` struct holds LLM provider + model reference
- `Compact(ctx, messages []Message, tokenBudget int) ([]Message, error)`:
  1. Estimate current token count
  2. If over budget, split into old (to summarize) and recent (to keep, last 4 messages)
  3. Call LLM with summarization prompt on old messages
  4. Return [system, summary, ...recent]
- Auto-trigger: when `contextManager.EstimateTokens() > model.MaxTokens * 0.8`
- Manual trigger: existing `/compact` command uses same `Compactor`
- Replaces current 200-character string truncation
**TDD:**
- `core/agent/compactor_test.go`: mock LLM returns fixed summary, verify old messages replaced
- `core/agent/compactor_test.go`: verify system prompt preserved, recent messages kept
- `core/agent/compactor_test.go`: verify under-budget messages unchanged
**Estimated:** ~200 lines

### Task 4.2: Provider Resilience — Exponential Backoff
**Files:** `core/providers/backoff.go` (new), `core/providers/factory.go` (modify)
**Spec:**
- `RetryProvider` wraps any `LLMProvider` with retry logic
- Config: `maxRetries=3`, `baseDelay=1s`, `maxDelay=30s`, `totalTimeout=30s`
- Retry on: HTTP 429, 500-599, network errors
- No retry on: 400-499 (client errors), context cancellation
- Exponential backoff: `delay = baseDelay * 2^attempt + jitter`
- Integrate into `Factory.GetProviderForModel()` — wrap returned provider
**TDD:**
- `core/providers/backoff_test.go`: mock provider fails 2x then succeeds, verify 3 calls made
- `core/providers/backoff_test.go`: verify total timeout enforced
- `core/providers/backoff_test.go`: verify 400 error NOT retried
**Estimated:** ~150 lines

### Task 4.3: Multi-Agent Delegation (sub_agents)
**Files:** `feature/config/config.go` (modify), `core/agent/agent.go` (modify), `core/agent/delegate.go` (new)
**Spec:**
- YAML config supports multiple agents:
  ```yaml
  agents:
    root:
      model: openai/gpt-4o
      sub_agents: [schema_expert, query_expert]
    schema_expert:
      model: anthropic/claude-3-5-sonnet
      system_prompt: "You analyze database schemas."
      tools: [database_read]
    query_expert:
      model: openai/gpt-4o
      system_prompt: "You write optimized SQL."
      tools: [database_read, database_write]
  ```
- `transfer_task` tool: parent agent calls it to delegate work to child
  - Input: `{agent: "schema_expert", task: "Analyze the users table schema"}`
  - Child runs in isolated session, returns result to parent
  - Parent continues with child's result
- Child agent has own tool registry subset, own session, own LLM calls
- Single-agent path (no `sub_agents` in config) remains default — zero overhead
- Implementation lives in `core/agent/delegate.go`, uses bus for child communication
**TDD:**
- `core/agent/delegate_test.go`: mock parent + child agents, verify delegation flow
- `core/agent/delegate_test.go`: verify child session isolated from parent
- `core/agent/delegate_test.go`: verify result returned to parent correctly
- `feature/config/config_test.go`: parse multi-agent YAML, verify agent map
**Estimated:** ~400 lines

## [S7] Phase 5 — RAG Enhancement (2 weeks)

### Task 5.1: BM25 Keyword Search
**Files:** `feature/rag/bm25.go` (new)
**Spec:**
- Pure Go BM25 implementation (no external dependencies)
- `BM25Index` struct with `Add(doc ID, text)`, `Search(query, topK) []ScoredDoc`
- Parameters: `k1=1.5`, `b=0.75` (standard defaults)
- Tokenization: simple whitespace + lowercasing (no stemming needed for Chinese+English mix)
- Must work alongside existing vector search
**TDD:**
- `feature/rag/bm25_test.go`: index 3 documents, search, verify top result is most relevant
- `feature/rag/bm25_test.go`: verify empty query returns empty results
- `feature/rag/bm25_test.go`: verify scoring consistency (same query = same order)
**Estimated:** ~200 lines

### Task 5.2: RRF Fusion + Hybrid Retrieval
**Files:** `feature/rag/fusion.go` (new), `feature/rag/retriever.go` (modify)
**Spec:**
- `ReciprocalRankFusion(rankLists [][]ScoredDoc, k int) []ScoredDoc`
  - Standard RRF formula: `score = sum(1 / (k + rank_i))` where `k=60`
- `HybridRetriever` combines BM25 + vector search:
  1. Run both searches in parallel
  2. Fuse results via RRF
  3. Return top-K merged results
- Replace current single-strategy retriever in `rag_retrieve` tool
**TDD:**
- `feature/rag/fusion_test.go`: two rank lists with overlap, verify fusion order
- `feature/rag/fusion_test.go`: verify empty input returns empty output
- `feature/rag/retriever_test.go`: mock BM25 + vector, verify hybrid returns merged results
**Estimated:** ~150 lines

### Task 5.3: Improved Chunking Strategy
**Files:** `feature/rag/chunker.go` (modify)
**Spec:**
- Add Markdown-aware splitting: split on `##` headings, preserve code blocks
- Configurable parameters: `chunk_size` (default 512 tokens), `overlap` (default 50 tokens)
- Preserve document metadata (filename, heading hierarchy)
**TDD:**
- `feature/rag/chunker_test.go`: Markdown input splits on headings
- `feature/rag/chunker_test.go`: code blocks not split mid-block
- `feature/rag/chunker_test.go`: overlap parameter respected
**Estimated:** ~80 lines

## [S8] Implementation Order & Dependencies

```
Phase 1 (1 week) ─────────────────────── 无依赖，立即开始
  ├─ 1.1 debug command
  ├─ 1.2 status enhance
  ├─ 1.3 init YAML
  └─ 1.4 config validate

Phase 3.2 (1 week) ───────────────────── 可与 Phase 1 并行
  └─ loop.go dedup

Phase 2 (2 weeks) ────────────────────── 依赖 Phase 1（YAML 格式确定）
  ├─ 2.1 YAML Schema ──┐
  ├─ 2.2 toolset loader ┤── 2.2 依赖 2.1
  └─ 2.3 provider registry ── 独立

Phase 3.1 (1 week) ───────────────────── 可与 Phase 2 并行
  └─ TUI slash commands

Phase 4 (3 weeks) ────────────────────── 依赖 Phase 2（多 agent YAML）
  ├─ 4.1 LLM compaction ── 独立
  ├─ 4.2 provider backoff ── 独立
  └─ 4.3 multi-agent ── 依赖 2.1

Phase 5 (2 weeks) ────────────────────── 独立，可任何时候进行
  ├─ 5.1 BM25
  ├─ 5.2 RRF fusion
  └─ 5.3 chunking
```

## [S9] TDD Verification Protocol

### Per-Task
```bash
go test ./affected/package/...
go vet ./affected/package/...
```

### Per-Phase
```bash
go test -race ./...
go vet ./...
```

### Full Build Verification
```bash
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o build/golem ./cmd/golem
```

### Coverage Target
- New modules: ≥80% coverage
- Modified modules: no coverage regression
- Existing 84 test files: all must pass unchanged (except loop.go refactor)

## [S10] Risk Mitigation

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| YAML migration breaks JSON users | Medium | Medium | Keep JSON via `--config` flag, `--format json` |
| Loop refactor introduces regression | Medium | High | 84 existing test files as safety net + `-race` |
| Multi-agent adds complexity | High | Medium | Opt-in only, single-agent path unaffected |
| BM25 external dependency | Low | Low | Pure Go hand-written implementation |
| Provider registry breaks existing vendors | Low | Medium | Migrate one vendor at a time, test each |

## [S11] What NOT to Copy from Docker-Agent

| Feature | Reason |
|---------|--------|
| OCI Registry distribution | Over-engineered for independent project; GoReleaser + Homebrew sufficient |
| Docker sandbox isolation | Golem's exec tool already has workspace limits; no Docker dependency |
| A2A protocol | Premature; implement sub_agents foundation first |
| HCL alternative syntax | YAML sufficient; HCL adds unnecessary complexity |
| Full RAG reranking | Progressive enhancement; BM25 + RRF fusion first |
| Config versioned auto-migration | Project still iterating fast; premature migration framework |
| Telemetry collection | Privacy concern for database agent use case |

## [S12] What Golem Must Keep

| Strength | Reason |
|----------|--------|
| Zero-CGO single binary | Termux/Android deployment; Docker Agent cannot do this |
| Database safety model | PermissionChecker + QualityGate + rollback SQL — unique differentiator |
| Strict layer imports | Go-enforced architecture; cleaner than Docker Agent |
| KV-cache tool ordering | Performance insight Docker Agent lacks |
| Chinese conversation | Target audience requirement |
| Message bus decoupling | Enables clean channel evolution |
| Feature modules via CLI flags | Clean optional wiring pattern |
| ~19K LOC readability | Learning project advantage |
