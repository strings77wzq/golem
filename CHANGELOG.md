# Changelog

All notable changes to Golem will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.10.1] - 2026-07-03

### Added
- Feature layer metrics: RAG, Memory, MCP, Skills tools now report latency/error/call metrics at `/metrics`
- `metricsToolWrapper` decorates feature tools at registration time (zero侵入 feature code)
- Bus `SetOnDropped` callback: operators can observe dropped messages via logger or custom handler

### Fixed
- Trace ID propagation: agent now reuses trace_id from gateway context instead of overwriting
- MCP Client `Close()` now waits for receiveLoop goroutine to exit (prevents goroutine leak)
- `interpolateString` infinite loop: added `maxInterpolationDepth=10` guard for self-referential variables
- Hook errors now logged (was silently discarded with `_ = err`)
- CLI status messages (`onboard.go`, `config.go`) now use structured logger instead of `fmt.Print`

## [0.10.0] - 2026-07-03

### Added
- Structured logging system with trace propagation, component tagging, and field templates
- Code verification principle: all code analysis must be based on verified Read/grep output
- BM25 keyword search with CJK character-level tokenization (`foundation/bm25/`)
- Docker multi-arch image publishing to GHCR on tag push (amd64/arm64)
- errcheck linter re-enabled with nolint annotations across 30+ files
- Gateway SSE streaming tests: handleOpenAICompatStream (non-streaming, streaming, error paths)
- Gateway session/metrics tests: handleSessionExport/Import, handleMetrics, SetSessionStore, MountHandler
- Fileops coverage tests: safePath edge cases, tool interfaces, error paths, symlink handling
- Openspec proposals: phase2-quality-gates, safety-and-correctness-fixes
- Evolution roadmap: axiom-driven analysis with 3 approaches, trust build plan

### Fixed
- Error handling for tool/skill registration: registry.Register() failures now logged or returned
- Safety and correctness fixes across core modules
- Phase 1 quality gates compliance (golangci-lint v2 config)
- CI: e2e model pull made non-fatal
- AGENTS.md module status markers synced with code reality (memory, routing, metrics)

### Changed
- AGENTS.md: memory → ✅ Wired, routing → ✅ Wired, metrics → ✅ Wired at /metrics endpoint
- CI workflow: Linux + macOS test matrix, GoReleaser 6-platform build
- golangci-lint: errcheck re-enabled, gocritic/gobern/ineffassign/prealloc/revive remain disabled
- Safety gates: SQL normalization, permission checks, rollback SQL generation hardened

## [0.9.1] - 2026-06-16

### Added
- Auto-compaction: triggers LLM-driven compression when context exceeds 80% of token budget
- Provider Registry: dynamic vendor registration via `init()`, adding new vendors requires zero code changes in wiring
- Streaming logic extracted to `streaming.go` for better code organization

### Fixed
- BM25 duplicate document handling: `Add()` now replaces existing document with same ID
- BM25 negative topK panic: `Search()` returns nil for `topK <= 0`
- Compactor nil provider panic: `Compact()` checks for nil provider before calling `summarize()`
- Provider factory now wraps all providers with RetryProvider for exponential backoff

### Changed
- `internal/wiring/providers.go`: replaced 9-case switch-case with Provider Registry pattern
- `core/agent/loop.go`: reduced from 982 to 929 lines via streaming extraction
- All providers registered via `init()` in respective packages (openai, anthropic, ollama)

## [0.9.0] - 2026-06-16

### Added
- `golem debug tools` — list all registered tools with descriptions and parameters
- `golem debug config` — show parsed config in YAML format (API keys masked)
- `golem config validate` — validate config file format and model names
- `golem status` — enhanced with tool list and feature module status display
- `golem init --format yaml` — YAML-first config generation (default, JSON via `--format json`)
- `golem init` — connectivity validation after config creation
- TUI slash commands: `/tools`, `/new`, `/sessions`, `/model`
- YAML config schema v2: typed tool config (database/mcp/memory/rag/infra)
- AgentSpec.commands for named prompt templates
- LLM-driven session compactor (replaces string truncation)
- RetryProvider with exponential backoff + jitter for provider resilience
- BM25 keyword search with CJK character-level tokenization
- Reciprocal Rank Fusion for merging ranked retrieval lists
- HybridRetriever combining BM25 + vector similarity search

### Fixed
- BM25 race condition: concurrent Add/Search now thread-safe with sync.RWMutex
- Compactor preserves tool call information in summaries
- Rate limit (429) errors now properly retried (removed from isClientError)
- Config validate checks default model exists in model_list
- Chinese text tokenization for BM25 search

### Changed
- Agent loop deduplication: extracted shared ReAct loop helpers (executeTools, processToolResults, saveAndEmitFinal)
- `processMessageFallback` reduced from ~120 to ~50 lines
- Status command now uses io.Writer for testability
- Init command outputs YAML by default (backward compatible with `--format json`)

## [0.8.0] - 2026-06-14

### Added
- Security gates integration: `PermissionChecker` and `QualityGate` wired into SQL tools
- Infrastructure tools: `kubectl`, `docker`, `helm` registered via `--infra` flag
- Gateway metrics: `MetricsMiddleware` + `/metrics` endpoint for Prometheus
- Qdrant vector search: proper search API with fallback to scroll
- Rollback SQL generation for DELETE/UPDATE operations
- Audit logging callback via `SetAuditFunc()`
- Redis TTL support in `redis_set` tool
- `CommandValidator` interface for exec tool customization
- Gateway session persistence via SQLite
- `--db` flag for agent command to connect databases
- MCP server with real tools (not stubs)
- Adapter tests for loadSkills, buildSystemPrompt, config parsers
- PG/MySQL/Redis driver interface tests
- Memory module coverage: 55.4% → 91.6%

### Fixed
- TUI table row count display bug
- MCP server stub tools replaced with real implementations
- Redis `redis_set` TTL parameter now passed to driver
- Qdrant search no longer returns hardcoded score 1.0
- demo-db performance: batch inserts (410K rows in seconds)

### Changed
- README updated to reflect actual capabilities (removed false claims)
- Agent `--db` flag wires SQLite tools automatically
- MCP server `buildMCPTools` uses real `core/tools/database/` tools
- Memory module tests: coverage 55.4% → 91.6%
- PG/MySQL/Redis driver interface tests

### Fixed
- TUI table row count display bug
- MCP server stub tools replaced with real implementations
- Redis `redis_set` TTL parameter now passed to driver
- Qdrant search no longer returns hardcoded score 1.0

### Changed
- README updated to reflect actual capabilities
- Agent `--db` flag wires SQLite tools automatically
- MCP server `buildMCPTools` uses real `core/tools/database/` tools

## [0.6.0] - 2026-06-13

### Added
- MCP Server (stdio mode) — expose tools to other agents
- TUI table rendering for SQL query results
- Database demo DB creation (`golem demo-db`)
- Multi-agent delegation via MCP stdio

### Changed
- Repositioned as "Go-native AI agent SDK"
- Updated README with new features

## [0.5.0] - 2026-06-10

### Added
- RAG pipeline with TF-IDF
- Skills module with built-in skills
- Memory module with importance decay
- Provider fallback routing
- Health check scheduler

### Changed
- Feature modules wired via CLI flags
- Improved TUI with scroll support

## [0.4.0] - 2026-06-05

### Added
- HTTP Gateway with OpenAI-compatible API
- Telegram bot adapter
- Session persistence (SQLite)
- Prometheus metrics system

## [0.3.0] - 2026-05-28

### Added
- Multi-provider LLM support (OpenAI, Anthropic, Ollama)
- Tool system with exec, fileops, websearch
- Bubble Tea TUI
- Config system with presets

## [0.2.0] - 2026-05-20

### Added
- Basic agent loop (ReAct)
- SQLite database driver
- CLI interactive mode

## [0.1.0] - 2026-05-15

### Added
- Initial release
- Basic LLM integration
- Project structure and architecture
