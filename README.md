# Golem

[![CI](https://github.com/strings77wzq/golem/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/golem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/golem)](https://goreportcard.com/report/github.com/strings77wzq/golem)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/strings77wzq/golem)](https://github.com/strings77wzq/golem/releases)
[![Tests](https://img.shields.io/badge/tests-41%20packages-brightgreen)](https://github.com/strings77wzq/golem/actions)
[![Coverage](https://img.shields.io/badge/coverage-82.5%25-brightgreen)](https://github.com/strings77wzq/golem)
[![E2E](https://img.shields.io/badge/E2E-passing-brightgreen)](https://github.com/strings77wzq/golem/actions/workflows/e2e.yml)

**Language:** [English](#golem) | [简体中文](#中文说明)

> **AI that understands your data — and knows its limits.**
> **AI 数据库安全代理——理解数据，强制安全。**

---

### The Problem

```
  You: "How many users signed up last week?"
  AI:   "I can't access your database."
  
  The data exists. The AI exists. No bridge.
```

### The Solution

```bash
$ golem agent --db ./myapp.db -m "Show me users from last week"
```

```
  Golem: ✓ Understands schema → SELECT * FROM users WHERE created_at > ...
         ✓ Enforces safety    → Read-only. WHERE required. Rollback ready.
         ✓ Returns results    → 47 users registered in the last 7 days.
```

---

## Quick Start

```bash
go install github.com/strings77wzq/golem/cmd/golem@latest
golem init
golem agent -m "hello"
```

→ [Full getting started guide](docs/QUICKSTART.md) | [Configuration](docs/CONFIG-REFERENCE.md) | [API reference](docs/GATEWAY-API.md)

---

## Why Golem?

Most AI database tools give you intelligence but no safety. Most database proxies give you safety but no intelligence. Golem is both — an AI agent that understands your schema **and** enforces security policies on every query.

| Capability | What Golem does |
|------------|-----------------|
| **Schema-aware** | Knows your tables, columns, relationships |
| **Natural language → SQL** | Ask in plain English, get results |
| **Read-only by default** | Writes require explicit permission |
| **WHERE enforcement** | No `DELETE FROM users` without a WHERE clause |
| **Rollback SQL** | Auto-generated for every destructive operation |
| **Audit trail** | Every allowed and denied operation is logged |

---

## Three Modes

**One binary. Three ways to use it.**

### Agent — Ask Your Database Directly

```bash
golem agent --db ./myapp.db -m "Show me users registered in the last 7 days"
```

### MCP Server — Let Claude Code / Cursor Query Your Data

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

---

## Safety Model

Golem enforces a 3-layer security model on every database operation:

| Operation | Default | With Permission |
|-----------|---------|-----------------|
| SELECT queries | ✅ Allowed | ✅ |
| INSERT/UPDATE | ❌ Blocked | ✅ (requires WHERE) |
| DELETE | ❌ Blocked | ✅ (requires WHERE) |
| Shell commands | ⚠️ Allowlist only | ✅ |

- **PermissionChecker** — operation-level access control (read/write/admin)
- **QualityGate** — WHERE clause enforcement on DELETE/UPDATE
- **Rollback SQL** — auto-generated for every destructive operation
- **SQL normalization** — strips comments, rejects multi-statement injection
- **Audit logging** — all operations traceable via structured events

---

## Configuration

### YAML Declarative Config

```yaml
version: 2
agent:
  model: openai/gpt-4o
  fallback_models:
    - anthropic/claude-3-haiku
  system_prompt: |
    You are a database assistant. Help users query their data.
  max_tokens: 8192
database:
  path: ./myapp.db
tools:
  - type: database
    path: ./myapp.db
  - type: mcp
    command: npx
    args: ["-y", "@modelcontextprotocol/server-duckduckgo"]
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

### Logging

Golem uses structured logging with `log/slog` (stdlib). All logs include `trace_id` for cross-component request correlation and `component` for module identification.

```bash
# JSON format (for production/log aggregation)
golem agent --db ./myapp.db --log-format json

# Text format (default, for development)
golem agent --db ./myapp.db --log-format text

# Debug level (verbose)
golem agent --db ./myapp.db --log-level debug
```

**Log output example:**
```
time=2026-06-30T11:15:00.000Z level=INFO msg="tool executed" component=agent tool=sql_query duration_ms=45 trace_id=trace-a1b2c3d4e5f67890
time=2026-06-30T11:15:00.045Z level=INFO msg="http request" component=gateway method=POST path=/api/chat status=200 duration_ms=120 trace_id=trace-a1b2c3d4e5f67890
```

For detailed logging architecture, see [docs/logging.md](docs/logging.md).

### Health Checks

```bash
golem gateway --health          # Default 5-minute interval
golem gateway --health '{"interval":"60s"}'  # Custom interval
```

Real-time provider health status at `GET /health/providers`.

### Debug Commands

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
make e2e                         # E2E tests (requires Ollama)
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

> **AI 数据库安全代理——理解数据，强制安全。**

### 问题

```
  你："上周有多少用户注册？"
  AI： "我无法访问你的数据库。”

  数据存在。AI 存在。没有桥梁。
```

### 解决方案

```bash
$ golem agent --db ./myapp.db -m "查询上周注册的用户"
```

```
  Golem：✓ 理解 schema → SELECT * FROM users WHERE created_at > ...
         ✓ 强制安全    → 只读。需要 WHERE。回滚 SQL 就绪。
         ✓ 返回结果    → 上周注册了 47 个用户。
```

**一个二进制文件。三种模式。零依赖。**

| 模式 | 命令 | 场景 |
|------|------|------|
| **Agent** | `golem agent --db ./myapp.db` | 独立运行：直接向数据库提问 |
| **MCP Server** | `golem mcp-server --db ./myapp.db` | 让 Claude Code / Cursor 查询数据 |
| **API Gateway** | `golem gateway` | 团队部署，OpenAI 兼容，端口 `:18790` |

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

### 快速开始

```bash
go install github.com/strings77wzq/golem/cmd/golem@latest
golem demo-db
golem agent --db .golem-demo.db -m "分析这个数据库"
```

详细文档请参考 [docs/](docs/) 目录。
