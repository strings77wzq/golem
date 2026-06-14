# Golem

[![CI](https://github.com/strings77wzq/golem/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/golem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/golem)](https://goreportcard.com/report/github.com/strings77wzq/golem)
[![License: MIT](https://img.shields.io/badge/License-MMIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/strings77wzq/golem)](https://github.com/strings77wzq/golem/releases)
[![GitHub Stars](https://img.shields.io/github/stars/strings77wzq/golem)](https://github.com/strings77wzq/golem/stargazers)

**Go-native AI agent SDK — LLM providers, database intelligence, MCP tool exposure, single binary.**

---

## What is Golem?

Golem is an **AI agent SDK for Go** with first-class database intelligence:

- **9 LLM providers** — OpenAI, Anthropic, Ollama (local), DeepSeek, MiMo, Kimi, GLM, MiniMax, Qwen
- **Database operations** — SQLite query/analyze/schema with security gates
- **MCP Server** — expose your tools to Claude Code, Cursor, OpenClaw via stdio
- **Single binary** — zero CGO, pure Go, runs on Linux/macOS/Android Termux
- **Safety built-in** — permission checks, WHERE clause enforcement, rollback SQL, audit logging

```
┌─────────────────────────────────────────────┐
│  Layer 3: Gateway                            │
│  HTTP API + OpenAI-compatible endpoint       │
├─────────────────────────────────────────────┤
│  Layer 2: CLI + TUI                          │
│  Terminal assistant with streaming output    │
├─────────────────────────────────────────────┤
│  Layer 1: SDK                                │
│  Go library — embed AI agent in any project  │
└─────────────────────────────────────────────┘
```

---

## Quick Start

```bash
# Install
go install github.com/strings77wzq/golem/cmd/golem@latest

# Setup (pick a provider)
golem init

# Chat
golem agent -m "Hello, what can you do?"

# Database analysis
golem demo-db                         # Create demo database
golem agent --db .golem-demo.db -m "Analyze this database"
```

---

## Database Intelligence

Connect to SQLite and let the agent analyze your data:

```bash
# Connect to database
golem agent --db ./mydata.db -m "What tables exist?"

# Query with natural language
golem agent --db ./mydata.db -m "Show me users with admin role"

# Analyze table structure
golem agent --db ./mydata.db -m "Analyze the orders table for performance issues"
```

**Security built-in:**
- SELECT-only by default (write operations require explicit permission)
- UPDATE/DELETE without WHERE clause are blocked
- Rollback SQL auto-generated for write operations
- All operations logged for audit trail

---

## MCP Server

Expose Golem's tools to other AI agents via MCP (Model Context Protocol):

```bash
# Start MCP server with database
golem mcp-server --db ./mydata.db

# Connect from Claude Code / Cursor / OpenClaw
# Add to your MCP config:
{
  "command": "golem",
  "args": ["mcp-server", "--db", "./mydata.db"]
}
```

**Available tools via MCP:**
- `sql_query` — Execute SQL SELECT queries
- `sql_schema` — Get database schema
- `sql_analyze` — Analyze table data distribution
- `exec` — Execute shell commands (disabled in read-only mode)
- `file_read` / `file_write` — File operations
- `web_search` — Search the web

---

## Safety Model

| Operation | Default | With Permission |
|-----------|---------|-----------------|
| SQL SELECT | ✅ Free | ✅ |
| SQL INSERT/UPDATE | ❌ Blocked | ✅ (requires WHERE) |
| SQL DELETE | ❌ Blocked | ✅ (requires WHERE) |
| Shell exec | ⚠️ Allowlist only | ✅ |

- **PermissionChecker** — operation-level access control
- **QualityGate** — WHERE clause enforcement
- **Rollback SQL** — auto-generated for DELETE/UPDATE
- **Audit logging** — callback-based operation recording

---

## Architecture

```
User Request
    │
    ▼
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Agent     │────▶│    Tools     │────▶│   LLM API   │
│  (ReAct)    │     │  Registry    │     │  (9 vendors)│
└──────┬──────┘     └──────┬───────┘     └─────────────┘
       │                   │
       ▼                   ▼
┌─────────────┐     ┌──────────────┐
│   Session   │     │   Database   │
│  (SQLite)   │     │  (SQLite)    │
└─────────────┘     └──────────────┘
```

**Layer dependencies (enforced by Go import system):**
```
cmd/  →  imports ALL layers (composition root)
internal/  →  imports core/ + foundation/
core/  →  imports foundation/ only
feature/  →  imports core/ + foundation/
foundation/  →  stdlib only
```

---

## Project Structure

```
golem/
├── cmd/golem/         # CLI entry point and composition root
├── core/              # Domain logic (interfaces + implementations)
│   ├── agent/         # ReAct agent loop
│   ├── tools/         # Tool implementations (exec, fileops, database, websearch)
│   ├── database/      # Database drivers (SQLite, PostgreSQL, MySQL, Redis, Qdrant)
│   ├── providers/     # LLM providers (OpenAI, Anthropic, Ollama, etc.)
│   ├── security/      # Permission checker, quality gates
│   └── session/       # Session management (SQLite + memory)
├── feature/           # Optional features (wired via CLI flags)
│   ├── mcp/           # MCP client + server
│   ├── rag/           # RAG pipeline (TF-IDF)
│   ├── skills/        # Skill registry
│   └── memory/        # Long-term memory with importance decay
├── internal/          # Internal adapters (not importable outside module)
│   ├── channels/      # CLI, TUI (Bubble Tea), Telegram
│   ├── gateway/       # HTTP gateway with OpenAI-compatible API
│   └── metrics/       # Prometheus metrics
└── foundation/        # Infrastructure primitives (concurrency, logger, store)
```

---

## Gateway API

```bash
golem gateway    # Starts on :18790
```

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/metrics` | GET | Prometheus metrics |
| `/api/chat` | POST | Synchronous chat |
| `/api/chat/stream` | POST | SSE streaming chat |
| `/v1/chat/completions` | POST | OpenAI-compatible API |
| `/v1/models` | GET | List available models |
| `/api/sessions/{id}/export` | GET | Export session |
| `/api/sessions/import` | POST | Import session |

---

## Testing

```bash
go test ./...                    # 38 packages
go test -race ./...              # race detector
go test -coverprofile=out ./...  # coverage
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

MIT License

## Acknowledgments

- Inspired by [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed
