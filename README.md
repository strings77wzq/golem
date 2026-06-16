# Golem

[![CI](https://github.com/strings77wzq/golem/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/golem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/golem)](https://goreportcard.com/report/github.com/strings77wzq/golem)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/strings77wzq/golem)](https://github.com/strings77wzq/golem/releases)

**One binary. Connect your database. Let AI agents query your data.**

[中文说明](#中文说明)

---

## What is Golem?

Golem is a Go-native database AI agent. It connects to your SQLite database, understands your table schema, and exposes query capabilities to Claude Code, Cursor, or any MCP-compatible agent via the Model Context Protocol.

No Python. No pip. Download a binary, point it at your database, and start querying.

### Why Golem?

If you use AI agents (Claude Code, Cursor, OpenClaw), you've hit a common problem: **they can't see your data.**

You can ask AI to write code, read files, execute commands — but it can't directly query your SQLite database to answer "how many users signed up last month?"

Golem solves this:

```
Your SQLite Database
        │
        ▼
┌─────────────────┐
│  Golem Agent    │  ← Connects to DB, understands schema
│  (single binary)│
└────────┬────────┘
         │ MCP protocol (stdio)
         ▼
┌─────────────────┐
│  Claude Code    │  ← Calls sql_query tool via MCP
│  / Cursor       │
│  / OpenClaw     │
└─────────────────┘
```

---

## Quick Start (5 minutes)

```bash
# 1. Install
go install github.com/strings77wzq/golem/cmd/golem@latest

# 2. Create demo database
golem demo-db

# 3. Let the agent analyze your data
golem agent --db .golem-demo.db -m "What tables exist? Any performance issues?"
```

---

## Core Features

### Database Intelligence

```bash
# Connect to database, query with natural language
golem agent --db ./myapp.db -m "Show me users registered in the last 7 days"

# Analyze table structure
golem agent --db ./myapp.db -m "Check if orders table indexes are optimal"

# Find performance issues
golem agent --db ./myapp.db -m "What needs optimization in this database?"
```

**Safety built-in:**
- Read-only by default (SELECT only)
- UPDATE/DELETE must include WHERE clause
- Auto-generated rollback SQL for write operations
- Audit logging for all operations

### MCP Server — Let Other Agents Query Your Database

```bash
# Start MCP server with database
golem mcp-server --db ./myapp.db
```

Other agents can call these tools via MCP:

| Tool | Function |
|------|----------|
| `sql_query` | Execute SQL SELECT queries |
| `sql_schema` | Get database schema |
| `sql_analyze` | Analyze table data distribution |
| `think` | Reasoning scratchpad for step-by-step thinking |
| `exec` | Execute shell commands (can be disabled) |
| `file_read` / `file_write` / `file_list` | File operations |
| `web_search` | Web search |

Add to Claude Code's MCP config:

```json
{
  "golem": {
    "command": "golem",
    "args": ["mcp-server", "--db", "./myapp.db"]
  }
}
```

### HTTP Gateway — OpenAI-Compatible API

```bash
golem gateway    # Starts on :18790
```

```bash
curl http://localhost:18790/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"List all admins"}]}'
```

### YAML Configuration — Declarative Agent Definition

Define your agent in YAML instead of CLI flags:

```yaml
# agent.yaml (v2)
version: 2
agent:
  model: openai/gpt-4o
  fallback_models:
    - anthropic/claude-3-haiku
  system_prompt: |
    You are a database assistant. Help users query their data.
  max_tokens: 8192
  commands:
    schema: "Analyze the database schema and list all tables"
    optimize: "Review the last query and suggest optimizations"
database:
  path: ./myapp.db
tools:
  - type: database
    path: ./myapp.db
  - type: mcp
    command: npx
    args: ["-y", "@modelcontextprotocol/server-duckduckgo"]
  - type: memory
    path: ./memory.db
hooks:
  pre_tool_use:
    command: ./scripts/validate.sh
```

```bash
golem agent --config agent.yaml
```

### Debug & Validation Commands

```bash
# List all registered tools
golem debug tools

# Show parsed config (API keys masked)
golem debug config

# Validate config file
golem config validate

# Show system status with tools and features
golem status
```

### Provider Fallback — Automatic Failover

If the primary model is unavailable, Golem automatically tries fallback models:

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

### TUI Slash Commands

| Command | Description |
|---------|-------------|
| `/tools` | List available tools |
| `/new` | Start a new session |
| `/sessions` | Browse past sessions |
| `/model` | Switch the current model |
| `/compact` | Compact conversation history (LLM-driven) |
| `/clear` | Clear conversation history |
| `/fork` | Fork the current session |
| `/help` | Show available commands |
| `/quit` | Exit the application |

---

## Safety Model

| Operation | Default | With Permission |
|-----------|---------|-----------------|
| SELECT queries | ✅ Allowed | ✅ |
| INSERT/UPDATE | ❌ Blocked | ✅ (requires WHERE) |
| DELETE | ❌ Blocked | ✅ (requires WHERE) |
| Shell commands | ⚠️ Allowlist only | ✅ |

- **PermissionChecker** — operation-level access control
- **QualityGate** — WHERE clause enforcement
- **Rollback SQL** — auto-generated for DELETE/UPDATE
- **Audit logging** — callback-based operation recording

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│                    cmd/golem/                     │
│         CLI entry point (composition root)        │
├─────────────────────────────────────────────────┤
│                internal/wiring/                   │
│      Dependency creation and configuration        │
├─────────────────────────────────────────────────┤
│                    core/                          │
│    agent (ReAct)  │  tools  │  database driver    │
│    providers      │  security │  session           │
├─────────────────────────────────────────────────┤
│                  feature/                         │
│       MCP client/server │ RAG │ Skills │ Memory   │
├─────────────────────────────────────────────────┤
│                  internal/                        │
│     TUI (Bubble Tea) │ Gateway │ Metrics          │
├─────────────────────────────────────────────────┤
│                 foundation/                       │
│         Concurrency │ Logger │ Store              │
└─────────────────────────────────────────────────┘

Dependency direction:
cmd/ → internal/wiring/ → core/*, feature/skills/*
cmd/ → internal/* → core/*
internal/wiring/ → core/*, feature/skills/*
foundation/ → stdlib only
```

### Project Structure

```
golem/
├── cmd/golem/              # CLI entry point
├── core/                   # Domain logic
│   ├── agent/              # ReAct agent loop
│   ├── tools/              # Tool implementations
│   ├── database/           # Database drivers (SQLite, PG, MySQL, Redis, Qdrant)
│   ├── providers/          # LLM providers (OpenAI, Anthropic, Ollama, etc.)
│   ├── security/           # Permission checker, quality gates
│   └── session/            # Session management
├── feature/                # Optional features (wired via CLI flags)
│   ├── config/             # YAML declarative agent configuration
│   ├── mcp/                # MCP client + server
│   ├── rag/                # RAG pipeline (TF-IDF)
│   ├── skills/             # Skill registry
│   └── memory/             # Long-term memory
├── internal/               # Internal adapters
│   ├── channels/           # CLI, TUI, Telegram
│   ├── gateway/            # HTTP gateway
│   ├── metrics/            # Prometheus metrics
│   ├── security/           # Auth, rate limiting, sandbox
│   └── wiring/             # Dependency creation
└── foundation/             # Infrastructure primitives
```

---

## Supported LLM Providers

| Provider | Local/Cloud | Notes |
|----------|-------------|-------|
| Ollama | Local | Fully offline |
| OpenAI | Cloud | GPT-4o, GPT-4 |
| Anthropic | Cloud | Claude 3.5 Sonnet |
| DeepSeek | Cloud | DeepSeek Chat |
| MiMo | Cloud | MiMo |
| Kimi | Cloud | Moonshot |
| GLM | Cloud | Zhipu |
| MiniMax | Cloud | MiniMax |
| Qwen | Cloud | Tongyi Qianwen |

Also supports any OpenAI-compatible API (vLLM, OpenRouter, LiteLLM).

---

## Gateway API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/metrics` | Prometheus metrics |
| POST | `/api/chat` | Synchronous chat |
| POST | `/api/chat/stream` | SSE streaming chat |
| POST | `/v1/chat/completions` | OpenAI-compatible API |
| GET | `/v1/models` | List available models |
| GET | `/api/sessions/{id}/export` | Export session |
| POST | `/api/sessions/import` | Import session |

---

## Testing

```bash
go test ./...                    # 41 packages
go test -race ./...              # Race detector
go test -coverprofile=out ./...  # Coverage
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

Core principles:
- `CGO_ENABLED=0` is mandatory
- Layer boundaries must be strict (cmd → wiring → core → foundation)
- Every PR needs tests
- Use Conventional Commits format

---

## License

MIT License

## Acknowledgments

- Inspired by [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed

---

## 中文说明

Golem 是一个 Go 原生的数据库 AI agent。它连接你的 SQLite 数据库，理解你的表结构，然后通过 MCP 协议把查询能力暴露给 Claude Code、Cursor 或任何支持 MCP 的 agent。

### 核心功能

- **数据库智能** — 连接 SQLite，用自然语言查询和分析数据
- **MCP Server** — 把你的数据库工具暴露给其他 AI agent
- **安全内置** — 默认只读，写操作需要 WHERE 子句，自动生成回滚 SQL
- **YAML 配置** — 用 YAML 声明式定义 agent（v2 支持 typed tools），无需编写代码
- **Provider Fallback** — 主模型不可用时自动切换备选模型，支持指数退避重试
- **Think 工具** — 推理草稿本，让 agent 分步思考复杂问题
- **LLM 压缩** — 基于 LLM 的会话压缩，替代简单截断
- **BM25 搜索** — 支持中英文的关键词搜索，混合检索融合
- **单二进制** — 零 CGO，纯 Go，支持 Linux/macOS/Android Termux

### 快速开始

```bash
go install github.com/strings77wzq/golem/cmd/golem@latest
golem demo-db
golem agent --db .golem-demo.db -m "分析这个数据库"
```

### YAML 配置示例

```yaml
version: 2
agent:
  model: openai/gpt-4o
  fallback_models:
    - anthropic/claude-3-haiku
  system_prompt: |
    你是一个数据库助手，帮助用户查询和分析数据。
  commands:
    schema: "分析数据库结构并列出所有表"
    optimize: "审查最后一条查询并建议优化"
database:
  path: ./myapp.db
tools:
  - type: database
    path: ./myapp.db
  - type: memory
    path: ./memory.db
```

```bash
golem agent --config agent.yaml
```

### 项目结构

```
cmd/golem/        → CLI 入口
internal/wiring/  → 依赖创建
core/             → 领域逻辑
feature/          → 可选功能（MCP、RAG、Skills、YAML Config）
internal/         → 内部适配器
foundation/       → 基础设施原语
```

详细文档请参考 [docs/](docs/) 目录。
