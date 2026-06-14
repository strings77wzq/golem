# Golem

[![CI](https://github.com/strings77wzq/golem/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/golem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/golem)](https://goreportcard.com/report/github.com/strings77wzq/golem)
[![License: MIT](https://img.shields.io/badge/License-MMIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/strings77wzq/golem)](https://github.com/strings77wzq/golem/releases)
[![GitHub Stars](https://img.shields.io/github/stars/strings77wzq/golem)](https://github.com/strings77wzq/golem/stargazers)

**Go-native AI agent SDK — build, embed, deploy.**

Golem understands your data and infrastructure. Connect to any database, query with natural language, analyze logs, and deploy with Docker/K8s — all with safety guarantees.

---

## What is Golem?

Golem is an **AI agent SDK for Go** that understands your full stack:

- **Talk to any LLM** — OpenAI, Anthropic, Ollama, DeepSeek, MiMo, 9 providers
- **Query any database** — SQLite, MySQL, PostgreSQL, Redis, VectorDB with auto-discovery
- **Operate infrastructure** — Docker, Kubernetes, Helm commands with safety gates
- **Analyze logs** — Identify issues in SQL/Docker/K8s/DB logs and suggest fixes
- **Plan complex tasks** — Decompose multi-step operations before executing
- **Observe everything** — 18+ Prometheus metrics, trace IDs, structured logging

It ships in **three layers**:

```
┌─────────────────────────────────────────────┐
│  Layer 3: Gateway                            │
│  HTTP API with OpenAI-compatible endpoint    │
├─────────────────────────────────────────────┤
│  Layer 2: CLI                                │
│  Terminal assistant with TUI                 │
├─────────────────────────────────────────────┤
│  Layer 1: SDK                                │
│  Go library — embed AI agent in any project  │
└─────────────────────────────────────────────┘
```

---

## 30-Second Demo

### Layer 1: SDK — embed in your Go project

```go
package main

import (
    "context"
    "fmt"
    "github.com/strings77wzq/golem/core/agent"
)

func main() {
    ag, _ := agent.QuickStart("~/.golem/config.json")
    response, _ := ag.Chat(context.Background(), "What is the capital of France?")
    fmt.Println(response)
}
```

### Layer 2: CLI — terminal AI assistant

```bash
go install github.com/strings77wzq/golem/cmd/golem@latest
golem init    # pick a provider (or Ollama for local)
golem agent -m "Hello"
```

### Layer 3: Gateway — OpenAI-compatible API

```bash
golem gateway    # starts on :18790

curl http://localhost:18790/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek/deepseek-chat","messages":[{"role":"user","content":"Hello"}]}'
```

---

## Features

### Database Intelligence
- **Auto-discover schema** — connect to any database, agent sees your tables
- **SQL operations** — sql_query, sql_schema, sql_analyze, sql_insert/update/delete
- **Redis** — redis_get, redis_set, redis_keys, redis_hgetall
- **VectorDB** — vector_search, vector_insert, vector_delete (Qdrant)
- **Multi-database** — query across SQLite + MySQL + PostgreSQL simultaneously

### MCP Server
- **Expose tools via MCP** — other agents (Claude Code, Cursor, OpenClaw) can call your tools
- **stdio transport** — connect via stdin/stdout JSON-RPC
- **Real tools** — not stubs. SQL queries, file ops, exec — all functional
- **Read-only mode** — expose only read operations for safety

```bash
# Start MCP server with database
golem mcp-server --db ./mydata.db

# Expose specific tools
golem mcp-server --db ./mydata.db --tools sql_query,sql_schema

# Read-only mode (no exec, no file_write)
golem mcp-server --db ./mydata.db --read-only
```

### Infrastructure Operations
- **Docker** — build, run, stop, ps, logs, exec, push, images
- **Kubernetes** — get, apply, delete, describe, logs, scale
- **Helm** — install, upgrade, list, rollback
- **Advisory mode** — all writes need user confirmation

### Security
- **3-layer safety** — permission verification → quality gates → user confirm
- **SQL safety** — WHERE clause required for DELETE/UPDATE, affected rows check
- **Rollback SQL** — auto-generate undo scripts for DELETE/UPDATE
- **Audit trail** — record all operations with timestamps

### Observability
- **18+ Prometheus metrics** — LLM calls, tool executions, DB queries, security gates
- **Trace ID** — unique ID per request, propagated through all operations
- **Structured logging** — each agent step logged with context

### Agent Intelligence
- **ReAct loop** — Think → Act → Observe with configurable max iterations
- **Tool selection** — smart tool picker based on task context
- **Reflection** — evaluate step results before continuing
- **Planning** — decompose complex tasks into structured plans
- **Progress display** — real-time TUI shows planning, tool calls, and results

### Multi-Provider LLM
- Ollama (local), OpenAI, Anthropic, DeepSeek, MiMo, Kimi, GLM, MiniMax, Qwen
- Any OpenAI-compatible endpoint (vLLM, OpenRouter, LiteLLM)

### Context Management
- **Token budget** — 20% system, 30% tools, 50% history
- **Smart compression** — truncate tool outputs, keep recent messages
- **Dynamic prompts** — assemble system prompt from tools + skills + context

---

## Quick Start

```bash
# Install
go install github.com/strings77wzq/golem/cmd/golem@latest

# Setup
golem init

# Chat
golem agent -m "Hello, what can you do?"

# Use tools
golem agent -m "Read the file README.md and summarize it"

# Gateway mode
golem gateway    # HTTP API on :18790
```

---

## Database Operations

```bash
# Connect to SQLite
golem agent --db ./mydata.db -m "What tables exist?"

# Query data
golem agent --db ./mydata.db -m "SELECT * FROM users WHERE role = 'admin'"

# Analyze table
golem agent --db ./mydata.db -m "Analyze the orders table"

# Create demo database
golem demo-db .golem-demo.db

# Analyze demo database
golem agent --db .golem-demo.db -m "Analyze this database and suggest optimizations"
```

---

## MCP Server

Golem can act as an MCP (Model Context Protocol) server, exposing its tools to other AI agents:

```bash
# Start MCP server
golem mcp-server --db ./mydata.db

# Connect from Claude Code / Cursor / OpenClaw
# Add to your MCP config:
{
  "command": "golem",
  "args": ["mcp-server", "--db", "./mydata.db"]
}
```

Available tools via MCP:
- `sql_query` — Execute SQL SELECT queries
- `sql_schema` — Get database schema information
- `sql_analyze` — Analyze table data distribution
- `exec` — Execute shell commands (disabled in read-only mode)
- `file_read` / `file_write` — File operations
- `web_search` — Search the web

---

## Safety Model

All database and infrastructure operations have safety gates:

| Operation | Default | With --allow-writes | With --confirm-delete |
|-----------|---------|---------------------|----------------------|
| SQL SELECT | ✅ Free | ✅ | ✅ |
| SQL INSERT/UPDATE | ❌ Blocked | ✅ Allowed | ✅ |
| SQL DELETE | ❌ Blocked | ❌ Blocked | ✅ Allowed |
| Docker read | ✅ Free | ✅ | ✅ |
| Docker write | ❌ Blocked | ✅ Allowed | ✅ |
| K8s delete | ❌ Blocked | ❌ Blocked | ✅ Allowed |

---

## Architecture

```
User Request → Planner → Tool Selector → Agent Loop → LLM + Tools → Response
                    ↓                    ↓
              Plan Execution      Context Manager
                    ↓                    ↓
              Reflector           Token Budget
                    ↓                    ↓
              Plan Revision       Message Compression
```

---

## Project Structure

```
golem/
├── cmd/golem/         # CLI entry point and composition root
├── core/              # Domain logic
│   ├── agent/         # Agent loop (ReAct)
│   ├── tools/         # Tool implementations
│   ├── database/      # Database drivers
│   ├── providers/     # LLM providers
│   └── session/       # Session management
├── feature/           # Optional features (wired via CLI flags)
│   ├── mcp/           # MCP client + server
│   ├── rag/           # RAG pipeline
│   ├── skills/        # Skill registry
│   └── memory/        # Long-term memory
├── internal/          # Internal adapters
│   ├── channels/      # CLI, TUI, Telegram
│   ├── gateway/       # HTTP gateway
│   └── metrics/       # Prometheus metrics
└── foundation/        # Infrastructure primitives
```

---

## Testing

```bash
go test ./...                                    # 38 packages
go test -race ./...                              # race detector
go test -coverprofile=coverage.out ./...         # coverage
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

MIT License

## Acknowledgments

- Inspired by [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed
