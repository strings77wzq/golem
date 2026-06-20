# Golem

[![CI](https://github.com/strings77wzq/golem/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/golem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/golem)](https://goreportcard.com/report/github.com/strings77wzq/golem)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/strings77wzq/golem)](https://github.com/strings77wzq/golem/releases)
[![Tests](https://img.shields.io/badge/tests-41%20packages-brightgreen)](https://github.com/strings77wzq/golem/actions)
[![Coverage](https://img.shields.io/badge/coverage-82.5%25-brightgreen)](https://github.com/strings77wzq/golem)

**Language:** [English](#golem) | [简体中文](#中文说明)

> **AI that understands your data — and knows its limits.**
> **AI 理解你的数据——也知道自己的边界。**

---

AI agents can write code, review PRs, debug systems — but they can't answer "how many users signed up last month?" The data exists. The AI exists. There's no bridge.

Golem is that bridge. It connects AI to your database, understands schemas, translates natural language to SQL, and enforces safety at every step. Not because AI is untrustworthy — because **unconstrained access to data is always risky**, whether the caller is a human or an LLM.

```bash
# Ask your database a question
golem agent --db ./myapp.db -m "Show me users registered this week"

# Let Claude Code query your database
golem mcp-server --db ./myapp.db

# Team API gateway
golem gateway    # OpenAI-compatible on :18790
```

One binary. Three modes. Zero dependencies.

---

## Why Golem?

### The Problem

```
Your Database (SQLite / PostgreSQL / MySQL / Redis / Qdrant)
        │
        ▼
┌─────────────────┐
│  AI Agent       │  ← Can write code, but can't see your data
│  (Claude, etc.) │
└─────────────────┘
        │
        ✗  No bridge.
```

### The Solution

```
Your Database (SQLite / PostgreSQL / MySQL / Redis / Qdrant)
        │
        ▼
┌─────────────────┐
│  Golem          │  ← Understands schema + enforces safety
│  (14MB binary)  │     Read-only default, WHERE enforcement,
└────────┬────────┘     rollback SQL, audit trail
         │
    ┌────┴────┐
    │  MCP    │  ← Claude Code, Cursor, any MCP client
    │  Agent  │  ← Standalone: no external agent needed
    │  API    │  ← OpenAI-compatible HTTP gateway
    └─────────┘
```

### What Makes It Different

**It understands your data AND knows its limits.**

Most database AI tools focus on one side: either they give AI full access (dangerous) or they block AI entirely (useless). Golem does both:

**Understanding (AI side):**
- Schema-aware: knows your tables, columns, relationships
- Natural language → SQL translation
- Query optimization suggestions
- Multi-database support (SQLite, PostgreSQL, MySQL, Redis, Qdrant)

**Safety (Proxy side):**
- Read-only by default — writes require explicit permission
- WHERE enforcement — no `DELETE FROM users` without a WHERE clause
- Rollback SQL — auto-generated for every destructive operation
- SQL normalization — strips comments, rejects multi-statement injection
- Audit trail — every allowed and denied operation is logged

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

> **AI 理解你的数据——也知道自己的边界。**

AI agent 能写代码、审查 PR、调试系统——但无法回答"上个月有多少用户注册？" 数据存在，AI 存在，只是缺少桥梁。

Golem 就是这座桥梁。它连接 AI 和数据库，理解表结构，将自然语言翻译为 SQL，并在每一步强制安全策略。不是因为 AI 不可信——而是**无论调用者是人还是 LLM，不受约束的数据访问始终是风险。**

```bash
# 向数据库提问
golem agent --db ./myapp.db -m "查询本周注册的用户"

# 让 Claude Code 查询数据库
golem mcp-server --db ./myapp.db

# 团队 API 网关
golem gateway    # OpenAI 兼容，端口 :18790
```

一个二进制文件。三种模式。零依赖。

### 核心优势

**理解（AI 侧）：**
- Schema 感知：知道你的表、列、关系
- 自然语言 → SQL 翻译
- 查询优化建议
- 多数据库支持（SQLite、PostgreSQL、MySQL、Redis、Qdrant）

**安全（代理侧）：**
- 默认只读——写操作需显式授权
- WHERE 强制——不允许无 WHERE 的 DELETE/UPDATE
- 回滚 SQL——每次破坏性操作自动生成
- SQL 归一化——剥离注释、拒绝多语句注入
- 审计追踪——每次允许和拒绝的操作都被记录

**基础设施：**
- 零依赖单二进制（14MB），支持 Linux/macOS/Android Termux
- 9 家 LLM 提供商（2 个协议适配器覆盖全部）
- Provider Fallback 路由 + 健康检查
- 安全头 + Prometheus 指标 + 结构化审计日志

### 快速开始

```bash
go install github.com/strings77wzq/golem/cmd/golem@latest
golem demo-db
golem agent --db .golem-demo.db -m "分析这个数据库"
```

详细文档请参考 [docs/](docs/) 目录。
