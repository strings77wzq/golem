# Golem

[![CI](https://github.com/strings77wzq/golem/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/golem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/golem)](https://goreportcard.com/report/github.com/strings77wzq/golem)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)

**Go-native AI agent runtime — build, embed, deploy.**

Golem is a pure-Go AI agent runtime with a ReAct reasoning loop, tool calling, multi-provider LLM support, and a single-binary deployment model. Run it from your terminal, embed it in your Go service, or deploy it on your phone via Termux.

```bash
go install github.com/strings77wzq/golem/cmd/golem@latest
golem init        # pick a provider (or Ollama for local)
golem agent -m "Hello, what can you do?"
```

## Why Golem?

| | Python frameworks | Golem |
|---|---|---|
| **Deploy** | pip install + venv + deps | Single binary, zero deps |
| **CGO** | N/A | `CGO_ENABLED=0` always |
| **Local LLM** | Requires Ollama separate | Built-in Ollama adapter |
| **Embed in Go** | FFI bridge or subprocess | `agent.QuickStart(config)` |
| **Edge/Mobile** | Not supported | Runs on Android/Termux |
| **Production** | Framework only | Gateway + auth + metrics + K8s |

## Quick Start

### As a CLI tool

```bash
golem agent                          # interactive TUI
golem agent -m "Summarize this"      # one-shot
echo "Hello" | golem agent           # pipe mode
```

### As a Go library

```go
import "github.com/strings77wzq/golem/core/agent"

ag, _ := agent.QuickStart("~/.golem/config.json")
response, _ := ag.Chat(ctx, "What is the capital of France?")
fmt.Println(response)
```

### As an OpenAI-compatible gateway

```bash
golem gateway    # starts on :18790

# Now any OpenAI SDK works:
curl http://localhost:18790/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek/deepseek-chat","messages":[{"role":"user","content":"Hello"}]}'
```

### With local LLM (Ollama)

```bash
golem init       # select "Ollama (local, no key needed)"
golem agent -M ollama/llama3 -m "Hello"
```

## Features

- **ReAct Agent Loop** — Think → Act → Observe with configurable max iterations
- **8 LLM Providers** — OpenAI, Anthropic, Ollama, DeepSeek, Kimi, GLM, MiniMax, Qwen (all OpenAI-compatible)
- **Tool System** — Pluggable registry: exec, file ops, web search, custom tools
- **MCP Protocol** — Client for external tool servers via stdio JSON-RPC
- **RAG Pipeline** — TF-IDF indexing + similarity search with OpenAI embeddings
- **Skills System** — Composable skill registry with tool-chain workflows
- **Long-term Memory** — Persistent memory with importance scoring and decay
- **Multiple Channels** — CLI, TUI (Bubble Tea), HTTP Gateway (SSE), Telegram bot
- **OpenAI-compatible API** — `/v1/chat/completions` — any OpenAI SDK connects directly
- **Embedded API** — `agent.QuickStart()`, `Chat()`, `ChatWithSession()` for Go integration
- **Agent Hooks** — BeforeMessage, AfterLLM, BeforeTool, AfterTool, OnError lifecycle callbacks
- **Cloud-Native** — Docker, Kubernetes, Helm, Prometheus metrics
- **Security** — Auth middleware, rate limiting, command sandboxing
- **Pure Go** — Zero CGO, single static binary, runs on Linux/macOS/Windows/Android

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Channels                          │
│        CLI / TUI / Gateway / Telegram               │
│        (OpenAI-compatible API)                       │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                  Agent Core                          │
│            ReAct Loop (Think→Act→Observe)            │
│            + Hooks (BeforeMessage/AfterLLM/...)      │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │  Tools   │  │  Skills  │  │   LLM Providers   │  │
│  │ Registry │  │ +Steps   │  │  8 adapters       │  │
│  │          │  │ Workflow │  │  (OpenAI-compat)  │  │
│  └──────────┘  └──────────┘  └───────────────────┘  │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │   MCP    │  │   RAG    │  │     Memory        │  │
│  │  Client  │  │ Pipeline │  │   Long-term       │  │
│  └──────────┘  └──────────┘  └───────────────────┘  │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                Infrastructure                        │
│  Session(SQLite) / Bus / Config / Logger             │
│  Security / Metrics / Routing / Embedded API         │
└─────────────────────────────────────────────────────┘
```

## LLM Provider Support

| Provider | API Format | Status | Notes |
|---|---|---|---|
| **Ollama** | OpenAI-compatible | Built-in | Local LLM, no API key |
| **OpenAI** | Chat Completions | Built-in | GPT-4o, GPT-4, etc. |
| **Anthropic** | Messages API | Built-in | Claude 3.5, Claude 3 |
| **DeepSeek** | OpenAI-compatible | Via openai adapter | |
| **Kimi/Moonshot** | OpenAI-compatible | Via openai adapter | |
| **GLM/Zhipu** | OpenAI-compatible | Via openai adapter | |
| **MiniMax** | OpenAI-compatible | Via openai adapter | |
| **Qwen/DashScope** | OpenAI-compatible | Via openai adapter | |

Any OpenAI-compatible endpoint works: vLLM, OpenRouter, LiteLLM, etc.

## Installation

```bash
# Go install
go install github.com/strings77wzq/golem/cmd/golem@latest

# Build from source (pure Go, no CGO)
git clone https://github.com/strings77wzq/golem.git && cd golem
CGO_ENABLED=0 go build -ldflags "-s -w" -o build/golem ./cmd/golem

# On Android/Termux
pkg install golang && go install github.com/strings77wzq/golem/cmd/golem@latest
```

## Usage

```bash
golem init                    # first-run setup wizard
golem agent                   # interactive TUI
golem agent -m "Hello"        # one-shot
golem gateway                 # HTTP gateway on :18790
golem version                 # version info
golem status                  # system status

# With features
golem agent --mcp ./mcp.json           # MCP tools
golem agent --rag ./docs               # RAG retrieval
golem agent --skills summarize         # built-in skills
golem agent --memory ~/.golem/mem.json # long-term memory

# Config management
golem config set default_model openai/gpt-4o
golem config get default_model
golem config list
```

## Testing

```bash
go test ./...                                    # all tests
go test -race ./...                              # race detector
go test -coverprofile=coverage.out ./...         # coverage (79.7%)
go test -bench=. -benchmem ./internal/gateway/   # benchmarks
```

## Learning Resources

The `docs/study/` directory contains Chinese learning guides:

1. **Architecture Overview** — Hexagonal architecture and design patterns
2. **Agent ReAct Loop** — How the Think→Act→Observe cycle works
3. **Tool System** — Building a pluggable tool registry
4. **Provider System** — LLM provider abstraction and adapters
5. **Message Bus** — Async event-driven communication

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## License

MIT License

## Acknowledgments

- Inspired by [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed
