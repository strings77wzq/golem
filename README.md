# Golem

[![CI](https://github.com/strings77wzq/golem/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/golem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/golem)](https://goreportcard.com/report/github.com/strings77wzq/golem)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/strings77wzq/golem)](https://github.com/strings77wzq/golem/releases)
[![Tests](https://img.shields.io/badge/tests-41%20packages-brightgreen)](https://github.com/strings77wzq/golem/actions)
[![Coverage](https://img.shields.io/badge/coverage-82.5%25-brightgreen)](https://github.com/strings77wzq/golem)

**Language:** [English](#golem) | [简体中文](#中文说明)

> **Let AI see your data.** / **让 AI 看见你的数据。**

Golem is a Go-native database AI agent. It connects to your databases, understands table schemas, and exposes query capabilities to Claude Code, Cursor, or any MCP-compatible agent via the Model Context Protocol.

**Zero Python. Zero Docker. Zero dependencies.** Download a 14MB binary, point it at your database, and start querying.

---

## Why Golem?

If you use AI agents (Claude Code, Cursor, OpenClaw), you've hit a common problem: **they can't see your data.**

You can ask AI to write code, read files, execute commands — but it can't directly query your SQLite database to answer "how many users signed up last month?"

```
Your Database (SQLite / PostgreSQL / MySQL / Redis / Qdrant)
        │
        ▼
┌─────────────────┐
│  Golem Agent    │  ← Connects to DB, understands schema
│  (14MB binary)  │
└────────┬────────┘
         │ MCP Protocol
         ▼
┌─────────────────┐
│  Claude Code    │
│  Cursor         │  ← Queries data via sql_query tool
│  Any MCP client │
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

Golem 是一个 Go 原生的数据库 AI Agent。它连接你的数据库，理解表结构，然后通过 MCP 协议把查询能力暴露给 Claude Code、Cursor 或任何支持 MCP 的 AI agent。

**零 Python。零 Docker。零依赖。** 下载一个 14MB 的二进制文件，指向你的数据库，即可开始查询。

### 为什么需要 Golem？

如果你使用 AI agent（Claude Code、Cursor、OpenClaw），你一定遇到过这个问题：**它们看不到你的数据。**

你可以让 AI 写代码、读文件、执行命令——但它无法直接查询你的 SQLite 数据库来回答"上个月有多少用户注册？"

### 核心优势

- **数据库安全模型** — 默认只读，写操作需要 WHERE 子句，自动生成回滚 SQL
- **零依赖单二进制** — 14MB，支持 Linux/macOS/Android Termux
- **严格分层架构** — Go import 系统强制执行，不是文档约定
- **LLM KV-Cache 优化** — 工具按字母序排列，最大化缓存复用
- **9 家 LLM 提供商** — OpenAI、Anthropic、DeepSeek、Kimi、GLM、MiniMax、Qwen、Ollama、MiMo（2 个协议适配器覆盖全部）
- **消息总线解耦** — TUI/Gateway/Telegram 通道独立演进
- **LLM 驱动压缩** — 替代简单截断，保留对话上下文
- **BM25 + 向量混合检索** — 支持中英文，RRF 融合排序
- **Provider Fallback 路由** — 自动故障转移，带 cooldown 追踪
- **健康检查** — 定时探测 provider 状态，`/health/providers` 实时暴露
- **安全头 + 审计日志** — CSP、X-Frame-Options 等安全头，401/429/403 事件结构化日志
- **Prometheus 指标** — LLM 调用、token 消耗、工具执行、成本估算全覆盖

### 快速开始

```bash
go install github.com/strings77wzq/golem/cmd/golem@latest
golem demo-db
golem agent --db .golem-demo.db -m "分析这个数据库"
```

### 架构概览

```
┌─────────────────────────────────────────────────────────────────────┐
│                        cmd/golem/ (组合根)                          │
│  CLI 入口 │ debug tools/config │ status │ config validate │ init    │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────┐
│                  internal/wiring/ (依赖注入层)                       │
│                  Provider Registry │ Tool Registration              │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────┐
│                        core/ (领域逻辑)                              │
│                                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │  Agent   │  │ Providers│  │  Tools   │  │     Session      │   │
│  │ReAct 循环│  │ 9 个 LLM │  │ 12 个工具│  │    会话管理       │   │
│  └────┬─────┘  └──────────┘  └──────────┘  └──────────────────┘   │
│       │                                                             │
│  ┌────▼─────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │ Context  │  │ Compactor│  │Streaming │  │   Bus (pub/sub)  │   │
│  │ 上下文管理│  │LLM 压缩  │  │  流式输出 │  │    消息总线       │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────────┘   │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────┐
│                   feature/ (可选模块)                                │
│                                                                     │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌──────┐ │
│  │ Config │ │  RAG   │ │  MCP   │ │ Skills │ │ Memory │ │Health│ │
│  │ YAML v2│ │BM25+RRF│ │协议    │ │技能注册│ │长期记忆│ │健康检│ │
│  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘ └──────┘ │
│  ┌────────┐                                                         │
│  │Routing │  Fallback 路由（带 cooldown 追踪）                       │
│  └────────┘                                                         │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────┐
│                   internal/ (适配器层)                               │
│                                                                     │
│  ┌─────┐ ┌─────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌────────┐  │
│  │ TUI │ │ CLI │ │Telegram │ │ Gateway │ │ Metrics │ │Security│  │
│  │终端UI│ │命令行│ │机器人   │ │HTTP API │ │Prometheus│ │安全中间│  │
│  └─────┘ └─────┘ └─────────┘ └─────────┘ └─────────┘ └────────┘  │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────┐
│              foundation/ (基础设施原语)                               │
│                                                                     │
│  ┌──────────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐             │
│  │ Concurrency  │ │ Logger  │ │  Store  │ │  Term   │             │
│  │并发原语       │ │结构化日志│ │SQLite   │ │终端检测  │             │
│  └──────────────┘ └─────────┘ └─────────┘ └─────────┘             │
└─────────────────────────────────────────────────────────────────────┘
```

**依赖方向：** `cmd/ → internal/ → core/ → foundation/`，foundation 层只导入标准库。

详细文档请参考 [docs/](docs/) 目录。
