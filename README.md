# Golem

[![CI](https://github.com/strings77wzq/golem/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/golem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/golem)](https://goreportcard.com/report/github.com/strings77wzq/golem)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/strings77wzq/golem)](https://github.com/strings77wzq/golem/releases)
[![Tests](https://img.shields.io/badge/tests-41%20packages-brightgreen)](https://github.com/strings77wzq/golem/actions)
[![Coverage](https://img.shields.io/badge/coverage-82.5%25-brightgreen)](https://github.com/strings77wzq/golem)

**Language:** [English](#golem) | [简体中文](#中文说明)

> **AI can write code, read files, execute commands — but it can't see your data.**
> **Golem fixes that. One binary. Zero trust assumptions.**

---

AI agents are powerful, but they have a blind spot: **your database**.

You can ask Claude to write a migration, review a query, debug a slow endpoint — but ask it "how many users signed up last month?" and it stares at you. The data exists. The agent exists. There's just no bridge between them.

Golem is that bridge. A single Go binary that sits between your AI agent and your database, understanding schemas, enforcing safety, and exposing query capabilities through the Model Context Protocol. No Python runtime. No Docker stack. No dependency chain. Just a binary you point at SQLite and start asking questions.

But Golem is more than a database connector. It's a bet on a specific idea: **that the future of AI infrastructure isn't bigger models — it's safer access patterns.** An LLM that can read your database is useful. An LLM that can read your database *and* is prevented from deleting it is trustworthy.

---

## Why Golem?

### The Problem

```
Your Database (SQLite / PostgreSQL / MySQL / Redis / Qdrant)
        │
        ▼
┌─────────────────┐
│  AI Agent       │  ← "I can help you write code about your data"
│  (Claude, etc.) │
└─────────────────┘
        │
        ✗  No bridge. The agent can't see your data.
```

### The Solution

```
Your Database (SQLite / PostgreSQL / MySQL / Redis / Qdrant)
        │
        ▼
┌─────────────────┐
│  Golem          │  ← Connects to DB, understands schema
│  (14MB binary)  │     Enforces: read-only default, WHERE clause,
└────────┬────────┘     rollback SQL, audit trail
         │ MCP Protocol
         ▼
┌─────────────────┐
│  Claude Code    │
│  Cursor         │  ← "Show me users from last 7 days"
│  Any MCP client │     Golem executes safely, returns results
└─────────────────┘
```

### What Makes It Different

**Safety is the product, not a feature.** Most AI-to-database tools give the agent full access and hope for the best. Golem assumes the agent is untrusted:

- **Read-only by default** — SELECT only, writes require explicit permission
- **WHERE enforcement** — no `DELETE FROM users` without a WHERE clause
- **Rollback SQL** — auto-generated for every destructive operation
- **Audit trail** — every allowed and denied operation is logged
- **SQL normalization** — strips comments, rejects multi-statement injection

**One binary. One command. Your data.**

```bash
golem agent --db ./myapp.db -m "Show me users registered this week"
```

No setup wizard. No config file needed. No Python environment to break. Point it at a database and ask questions.

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

### Safety Model

| Operation | Default | With Permission |
|-----------|---------|-----------------|
| SELECT queries | ✅ Allowed | ✅ |
| INSERT/UPDATE | ❌ Blocked | ✅ (requires WHERE) |
| DELETE | ❌ Blocked | ✅ (requires WHERE) |
| Shell commands | ⚠️ Allowlist only | ✅ |

- **PermissionChecker** — operation-level access control
- **QualityGate** — WHERE clause enforcement
- **Rollback SQL** — auto-generated for DELETE/UPDATE
- **Audit logging** — all operations traceable

### MCP Server — Let Other Agents Query Your Database

```bash
golem mcp-server --db ./myapp.db
```

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

### YAML Declarative Configuration

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

### Routing & Failover

```bash
golem agent --routing '{"routes":{"gpt-4o":["openai/gpt-4o","anthropic/claude-3-haiku"]}}'
```

Automatic fallback with cooldown tracking — if the primary provider fails, Golem retries with the next provider in the chain.

### Health Checks

```bash
golem gateway --health          # Default 5-minute interval
golem gateway --health '{"interval":"60s"}'  # Custom interval
```

Real-time provider health status at `GET /health/providers`.

### Debug & Validation Commands

```bash
golem debug tools      # List all registered tools
golem debug config     # Show parsed config (API keys masked)
golem config validate  # Validate config file
golem status           # Show system status with tools and features
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

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        cmd/golem/ (Composition Root)                │
│  CLI Entry │ debug tools/config │ status │ config validate │ init   │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────┐
│                    internal/wiring/ (Dependency Creation)           │
│                  Provider Registry │ Tool Registration              │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────┐
│                          core/ (Domain Logic)                       │
│                                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │  Agent   │  │ Providers│  │  Tools   │  │     Session      │   │
│  │ReAct Loop│  │ 9 LLMs  │  │ 12 Tools │  │Session Management│   │
│  └────┬─────┘  └──────────┘  └──────────┘  └──────────────────┘   │
│       │                                                             │
│  ┌────▼─────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │ Context  │  │ Compactor│  │Streaming │  │   Bus (pub/sub)  │   │
│  │ Management│  │LLM Comp. │  │  Output  │  │  Message Bus     │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────────┘   │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────┐
│                     feature/ (Optional Modules)                     │
│                                                                     │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌──────┐ │
│  │ Config │ │  RAG   │ │  MCP   │ │ Skills │ │ Memory │ │Health│ │
│  │ YAML v2│ │BM25+RRF│ │Protocol│ │Registry│ │Long-term│ │Checks│ │
│  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘ └──────┘ │
│  ┌────────┐                                                         │
│  │Routing │  Fallback Routing (cooldown-aware)                     │
│  └────────┘                                                         │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────┐
│                    internal/ (Internal Adapters)                     │
│                                                                     │
│  ┌─────┐ ┌─────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌────────┐  │
│  │ TUI │ │ CLI │ │Telegram │ │ Gateway │ │ Metrics │ │Security│  │
│  │Bubble│ │readln│ │Bot Adpt.│ │HTTP API │ │Prometheus│ │Auth/RL │  │
│  └─────┘ └─────┘ └─────────┘ └─────────┘ └─────────┘ └────────┘  │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────┐
│                foundation/ (Infrastructure Primitives)               │
│                                                                     │
│  ┌──────────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐             │
│  │ Concurrency  │ │ Logger  │ │  Store  │ │  Term   │             │
│  │Pool/Semaphore│ │slog     │ │SQLite   │ │isatty   │             │
│  └──────────────┘ └─────────┘ └─────────┘ └─────────┘             │
└─────────────────────────────────────────────────────────────────────┘
```

**Dependency direction:**
```
cmd/ → internal/wiring/ → core/*, feature/*
cmd/ → internal/* → core/*
foundation/ → stdlib only (zero project dependencies)
```

---

## Supported LLM Providers

Golem uses **2 protocol adapters** that cover all providers:

| Protocol | Adapter | Providers |
|----------|---------|-----------|
| **OpenAI-compatible** | `openai` adapter | OpenAI, DeepSeek, Kimi (Moonshot), GLM (Zhipu), MiniMax, Qwen (DashScope), MiMo, Ollama (local) |
| **Anthropic** | `anthropic` adapter | Anthropic (Claude) |

Any provider with an OpenAI-compatible API works out of the box — just set the base URL:

```yaml
model_list:
  - model: deepseek-chat
    vendor: deepseek
    api_base: https://api.deepseek.com    # OpenAI-compatible
  - model: gpt-4o
    vendor: openai
  - model: claude-3-5-sonnet
    vendor: anthropic
```

Also supports any OpenAI-compatible endpoint (vLLM, OpenRouter, LiteLLM) via custom `api_base`.

---

## Gateway API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/health/providers` | Provider health status |
| GET | `/api/version` | Version info |
| GET | `/metrics` | Prometheus metrics |
| POST | `/api/chat` | Synchronous chat |
| POST | `/api/chat/stream` | SSE streaming chat |
| POST | `/v1/chat/completions` | OpenAI-compatible API |
| GET | `/v1/models` | List available models |
| GET | `/api/sessions/{id}/export` | Export session |
| POST | `/api/sessions/import` | Import session |

---

## Metrics

Golem exposes Prometheus-compatible metrics at `GET /metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `agent_llm_calls_total` | Counter | Total LLM calls |
| `agent_llm_latency_seconds` | Histogram | LLM call latency |
| `agent_llm_tokens_total` | Counter | Total tokens consumed |
| `agent_llm_cost_usd_total` | Counter | Estimated cost (×10000) |
| `agent_tool_calls_total` | Counter | Total tool executions |
| `agent_tool_latency_seconds` | Histogram | Tool execution latency |
| `agent_sessions_active` | Gauge | Concurrent active sessions |
| `agent_context_tokens_used` | Gauge | Context window usage |
| `http_requests_total` | Counter | Total HTTP requests |
| `http_request_duration_seconds` | Histogram | HTTP request latency |

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

---

## Acknowledgments

- Design inspired by [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed
- Architecture reference: Docker Agent's declarative configuration approach

---

## 中文说明

[English](#golem) | **中文**

> **AI 能写代码、读文件、执行命令——但它看不到你的数据。**
> **Golem 解决这个问题。一个二进制文件。零信任假设。**

AI agent 很强大，但有一个盲区：**你的数据库**。

你可以让 Claude 写迁移、审查查询、调试慢接口——但问它"上个月有多少用户注册？"，它无能为力。数据存在，agent 存在，只是两者之间没有桥梁。

Golem 就是这座桥梁。一个 Go 二进制文件，坐在 AI agent 和数据库之间，理解表结构、强制安全策略、通过 MCP 协议暴露查询能力。无需 Python 运行时，无需 Docker 栈，无需依赖链。指向 SQLite，开始提问。

但 Golem 不只是数据库连接器。它基于一个信念：**AI 基础设施的未来不是更大的模型，而是更安全的访问模式。** 一个能读你数据库的 LLM 有用，一个能读你数据库**且被阻止删除数据的** LLM 才值得信任。

### 核心优势

- **安全即产品** — 默认只读，写操作需显式授权，WHERE 子句强制，自动生成回滚 SQL
- **SQL 注入防护** — 剥离注释、拒绝多语句、保守分类器
- **审计追踪** — 每次允许和拒绝的操作都被记录
- **零依赖单二进制** — 14MB，支持 Linux/macOS/Android Termux
- **严格分层架构** — Go import 系统强制执行，不是文档约定
- **LLM KV-Cache 优化** — 工具按字母序排列，最大化缓存复用
- **9 家 LLM 提供商** — 2 个协议适配器覆盖全部（OpenAI-compatible + Anthropic）
- **消息总线解耦** — TUI/Gateway/Telegram 通道独立演进
- **LLM 驱动压缩** — 替代简单截断，保留对话上下文
- **BM25 + 向量混合检索** — 支持中英文，RRF 融合排序
- **Provider Fallback 路由** — 自动故障转移，带 cooldown 追踪
- **健康检查** — 定时探测 provider 状态
- **安全头 + 审计日志** — CSP、X-Frame-Options 等安全头，401/429/403 事件结构化日志
- **Prometheus 指标** — LLM 调用、token 消耗、工具执行、成本估算全覆盖

### 快速开始

```bash
go install github.com/strings77wzq/golem/cmd/golem@latest
golem demo-db
golem agent --db .golem-demo.db -m "分析这个数据库"
```

详细文档请参考 [docs/](docs/) 目录。
