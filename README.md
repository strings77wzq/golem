# Golem

[![CI](https://github.com/strings77wzq/golem/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/golem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/golem)](https://goreportcard.com/report/github.com/strings77wzq/golem)
[![License: MIT](https://img.shields.io/badge/License-MMIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/strings77wzq/golem)](https://github.com/strings77wzq/golem/releases)
[![GitHub Stars](https://img.shields.io/github/stars/strings77wzq/golem)](https://github.com/strings77wzq/golem/stargazers)

**Go-native AI agent SDK with CLI and Gateway — build, embed, deploy.**

---

## What is Golem?

Golem is an **AI agent SDK for Go**. It gives your Go application the ability to:

- Talk to any LLM (OpenAI, Anthropic, Ollama, DeepSeek, etc.)
- Call tools (read files, execute commands, search the web)
- Maintain conversation memory across turns
- Run a ReAct reasoning loop (think → act → observe)

It ships in **three layers** — use whichever fits your needs:

```
┌─────────────────────────────────────────────┐
│  Layer 3: Gateway                            │
│  HTTP API server with OpenAI-compatible      │
│  /v1/chat/completions endpoint               │
│  Any OpenAI SDK connects directly            │
├─────────────────────────────────────────────┤
│  Layer 2: CLI                                │
│  Terminal AI assistant with TUI              │
│  golem agent, golem init, golem gateway      │
├─────────────────────────────────────────────┤
│  Layer 1: SDK                                │
│  Go library — embed AI agent in any project  │
│  agent.QuickStart() / agent.Chat()           │
└─────────────────────────────────────────────┘
```

---

## 30-Second Demos

### Layer 1: SDK — embed in your Go project

```go
package main

import (
    "context"
    "fmt"
    "github.com/strings77wzq/golem/core/agent"
)

func main() {
    // One line to create an agent
    ag, _ := agent.QuickStart("~/.golem/config.json")

    // One line to chat
    response, _ := ag.Chat(context.Background(), "What is the capital of France?")
    fmt.Println(response)
    // Output: The capital of France is Paris.
}
```

**That's it.** Your Go app now has AI agent capabilities — tool calling, memory, multi-provider support. No Python, no pip, no subprocess.

### Layer 2: CLI — terminal AI assistant

```bash
# Install
go install github.com/strings77wzq/golem/cmd/golem@latest

# Setup (interactive wizard)
golem init

# Chat
golem agent -m "What can you do?"

# Use tools
golem agent -m "Read the file README.md and summarize it"

# Pipe mode
echo "Summarize this document" | golem agent
```

### Layer 3: Gateway — OpenAI-compatible API

```bash
# Start gateway
golem gateway    # runs on :18790

# Now any OpenAI SDK works:
curl http://localhost:18790/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek/deepseek-chat","messages":[{"role":"user","content":"Hello"}]}'
```

```python
# Or use the Python OpenAI SDK:
from openai import OpenAI
client = OpenAI(base_url="http://localhost:18790/v1", api_key="any")
response = client.chat.completions.create(
    model="deepseek/deepseek-chat",
    messages=[{"role": "user", "content": "Hello"}]
)
print(response.choices[0].message.content)
```

---

## Architecture — How It Works

### The Agent Loop (ReAct Pattern)

Every request goes through this cycle:

```
User: "What's the weather in Beijing?"
  │
  ▼
┌─────────────────────────────────┐
│ 1. THINK                        │
│    LLM receives message + tools │
│    LLM decides: call web_search │
└──────────────┬──────────────────┘
               │ tool_call: web_search("Beijing weather")
               ▼
┌─────────────────────────────────┐
│ 2. ACT                          │
│    Agent executes the tool      │
│    Returns: "Sunny, 25°C"      │
└──────────────┬──────────────────┘
               │ tool_result: "Sunny, 25°C"
               ▼
┌─────────────────────────────────┐
│ 3. OBSERVE                      │
│    LLM receives tool result     │
│    LLM decides: done, compose   │
│    response                     │
└──────────────┬──────────────────┘
               │
               ▼
Response: "The weather in Beijing is sunny, 25°C."
```

### Data Flow (Request → Response)

```
                    ┌──────────────┐
                    │   User       │
                    └──────┬───────┘
                           │ message
                    ┌──────▼───────┐
                    │   Channel    │  CLI / TUI / Gateway / Telegram
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │   Agent      │  ReAct loop (max 25 iterations)
                    │   ┌────────┐ │
                    │   │Session │ │  conversation history (SQLite)
                    │   └────────┘ │
                    │   ┌────────┐ │
                    │   │History │ │  context window management
                    │   └────────┘ │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
     ┌────────▼───┐ ┌──────▼─────┐ ┌───▼────────┐
     │   LLM      │ │   Tools    │ │   Skills   │
     │  Provider  │ │  Registry  │ │  Registry  │
     │            │ │            │ │            │
     │ OpenAI     │ │ exec       │ │ summarize  │
     │ Anthropic  │ │ file_read  │ │ code-review│
     │ Ollama     │ │ web_search │ │ + custom   │
     │ DeepSeek   │ │ + MCP      │ │            │
     │ ...        │ │ + custom   │ │            │
     └────────────┘ └────────────┘ └────────────┘
```

### Package Structure

```
golem/
├── core/                    # Layer 1: SDK (the library)
│   ├── agent/              #   Agent + ReAct loop + embedded API
│   ├── providers/          #   LLM adapters (OpenAI, Anthropic, Ollama...)
│   ├── tools/              #   Tool interface + registry
│   ├── session/            #   Conversation history (SQLite)
│   ├── bus/                #   Internal message bus
│   └── config/             #   Configuration system
│
├── feature/                 # Optional extensions
│   ├── mcp/                #   MCP protocol client
│   ├── rag/                #   RAG pipeline
│   ├── skills/             #   Skill registry + workflows
│   └── memory/             #   Long-term memory
│
├── internal/               # Layer 2+3: Channels
│   ├── channels/tui/       #   Bubble Tea TUI
│   ├── channels/cli/       #   Plain interactive mode
│   ├── channels/telegram/  #   Telegram bot
│   └── gateway/            #   HTTP API server
│
├── cmd/golem/              # CLI entry point
├── foundation/             # Infrastructure (logger, store, concurrency)
├── k8s/                    # Kubernetes manifests
├── helm/                   # Helm charts
└── docker/                 # Docker build
```

---

## LLM Provider Support

| Provider | API Format | Setup |
|---|---|---|
| **Ollama** | OpenAI-compatible | `golem init` → select Ollama (no API key) |
| **OpenAI** | Chat Completions | `export OPENAI_API_KEY=sk-...` |
| **Anthropic** | Messages API | `export ANTHROPIC_API_KEY=sk-ant-...` |
| **DeepSeek** | OpenAI-compatible | `export DEEPSEEK_API_KEY=sk-...` |
| **Kimi/Moonshot** | OpenAI-compatible | `export MOONSHOT_API_KEY=sk-...` |
| **GLM/Zhipu** | OpenAI-compatible | `export ZHIPU_API_KEY=...` |
| **MiniMax** | OpenAI-compatible | `export MINIMAX_API_KEY=...` |
| **Qwen/DashScope** | OpenAI-compatible | `export DASHSCOPE_API_KEY=sk-...` |

Any OpenAI-compatible endpoint works: vLLM, OpenRouter, LiteLLM, etc.

---

## Installation

```bash
# Go install (recommended)
go install github.com/strings77wzq/golem/cmd/golem@latest

# Pre-built binaries (no Go required)
# https://github.com/strings77wzq/golem/releases

# Build from source
git clone https://github.com/strings77wzq/golem.git && cd golem
CGO_ENABLED=0 go build -o build/golem ./cmd/golem

# Android/Termux
pkg install golang && go install github.com/strings77wzq/golem/cmd/golem@latest
```

---

## Usage

```bash
# Setup
golem init                         # first-run wizard

# CLI agent
golem agent                        # interactive TUI
golem agent -m "Hello"             # one-shot
golem agent -M ollama/llama3       # use specific model

# Gateway
golem gateway                      # HTTP API on :18790

# Features
golem agent --mcp ./mcp.json       # MCP tools
golem agent --rag ./docs           # RAG retrieval
golem agent --skills summarize     # built-in skills
golem agent --memory ./mem.json    # long-term memory

# Config
golem config set default_model openai/gpt-4o
golem config list
golem status
```

---

## Embedding as a Go Library

```go
import "github.com/strings77wzq/golem/core/agent"

// Create agent from config
ag, _ := agent.QuickStart("~/.golem/config.json")

// Single turn
response, _ := ag.Chat(ctx, "Hello")

// Persistent session (maintains context)
response, _ := ag.ChatWithSession(ctx, "session-123", "Hello")
response, _ = ag.ChatWithSession(ctx, "session-123", "What did I just say?")

// Streaming
ag.ChatStream(ctx, "Tell me a story", func(token string) {
    fmt.Print(token)
})

// Custom hooks
ag := agent.New(bus, registry, factory, store, history, log, cfg,
    agent.WithHooks(&agent.Hooks{
        BeforeTool: func(ctx context.Context, call providers.ToolCall) error {
            log.Printf("calling tool: %s", call.Name)
            return nil
        },
    }),
)
```

---

## Testing

```bash
go test ./...                                    # all tests (31 packages)
go test -race ./...                              # race detector
go test -coverprofile=coverage.out ./...         # coverage (79.7%)
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT License

## Acknowledgments

- Inspired by [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed
