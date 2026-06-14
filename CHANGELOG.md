# Changelog

All notable changes to Golem will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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
