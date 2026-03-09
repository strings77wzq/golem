# Golem

[![CI](https://github.com/strings77wzq/golem/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/golem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/golem)](https://goreportcard.com/report/github.com/strings77wzq/golem)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)

✨ **纯Go实现的轻量AI Agent框架，单二进制无依赖，支持7大LLM厂商、MCP/RAG/技能系统，可直接在Android/Termux运行，既是生产可用的AI助手，也是极佳的Go语言AI开发学习项目。**

Golem 是一个云原生AI助理框架，从零实现了完整的AI Agent系统，包含ReAct推理循环、工具调用系统、多LLM提供商适配、MCP协议支持、RAG检索增强、技能系统等核心能力，同时支持Docker/K8s部署，完美适配云原生场景。

## Features

- **Agent ReAct Loop** — Think → Act → Observe reasoning cycle with configurable max iterations
- **Tool System** — Pluggable tool registry with built-in exec, file operations, and web search
- **LLM Providers** — OpenAI, Anthropic, DeepSeek, Kimi, GLM, MiniMax, and Qwen adapters with streaming support
- **MCP Client** — Model Context Protocol client for external tool integration
- **RAG Pipeline** — Retrieval-Augmented Generation with TF-IDF indexing, similarity search, and OpenAI embedding support
- **Skills System** — Composable skill registry with built-in skills (summarize, code-review)
- **Long-term Memory** — Persistent memory with importance scoring and exponential decay
- **Multiple Channels** — CLI, interactive TUI (Bubble Tea, auto-detected on TTY), HTTP Gateway (with SSE streaming), and Telegram bot adapters
- **First-run Wizard** — `golem init` interactive setup with 7 provider presets
- **Message Bus** — Async pub/sub event system for decoupled communication
- **Session Management** — Conversation history with SQLite persistence
- **Security** — Auth middleware, rate limiting, and command sandboxing
- **Concurrency** — Worker pool, semaphore, and rate limiter primitives
- **Prometheus Metrics** — Pure Go metrics (counter/gauge/histogram) with exposition endpoint
- **Cloud-Native** — Docker, Kubernetes, Helm, CI/CD, monitoring stack, and config hot reload (SIGHUP)

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Channels                          │
│              CLI / Gateway / Telegram                │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                  Agent Core                          │
│            ReAct Loop (Think→Act→Observe)            │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │  Tools   │  │  Skills  │  │   LLM Providers   │  │
│  │  Registry│  │  Registry│  │  OpenAI / Anthropic│  │
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
│  Session / Store(SQLite) / Bus / Config / Logger     │
│  Security / Concurrency / Metrics / Routing          │
└─────────────────────────────────────────────────────┘
```

## Installation

### From Source (go install)

```bash
go install github.com/strings77wzq/golem/cmd/golem@latest
```

This installs the `golem` binary to your `$GOPATH/bin` (or `$HOME/go/bin`). Make sure it's in your `PATH`.

### From Release Binaries

Download pre-built binaries from the [Releases](https://github.com/strings77wzq/golem/releases) page. Available for Linux, macOS, and Windows (amd64/arm64).

### Build from Source

```bash
git clone https://github.com/strings77wzq/golem.git
cd golem

# Build binary (pure Go, no CGO)
CGO_ENABLED=0 go build -ldflags "-s -w" -o build/golem ./cmd/golem

# Or use Makefile
make build
```

### On Android/Termux (ARM64)

Golem builds and runs natively on Android via [Termux](https://termux.dev/) — no root required.

```bash
# Install Go in Termux
pkg install golang

# Install directly via go install
go install github.com/strings77wzq/golem/cmd/golem@latest
# Binary lands at $HOME/go/bin/golem

# Or build from source
git clone https://github.com/strings77wzq/golem.git
cd golem
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath \
    -o ~/bin/golem ./cmd/golem
```

> **Termux notes:**
> - The TUI auto-activates when stdin is a TTY (standard Termux terminal); pipe/redirect falls back to plain output automatically.
> - Mouse input is disabled by default — compatible with all Termux terminal emulators and Android keyboards.
> - Alt+key shortcuts are not used; all keybindings work with standard terminal key sequences.
> - Use `golem init` for the first-run setup wizard to configure your API key.

## 🚀 三分钟快速上手

### 1. 下载安装
```bash
# 方式1：Go 直接安装（推荐）
go install github.com/strings77wzq/golem/cmd/golem@latest

# 方式2：下载预编译二进制
# 从 https://github.com/strings77wzq/golem/releases 下载对应平台版本
```

### 2. 初始化配置
运行交互式配置向导，选择你使用的LLM厂商并配置API密钥：
```bash
golem init
```
支持7大LLM厂商：OpenAI、Anthropic、DeepSeek、Moonshot/Kimi、Zhipu/GLM、MiniMax、DashScope/Qwen，开箱即用。

### 3. 第一次对话
启动TUI交互界面，直接开始对话：
```bash
golem agent
```
或者直接执行单次查询：
```bash
golem agent -m "用Go写一个快速排序算法"
```

### 4. 体验进阶功能
#### 👉 开启RAG本地知识库
```bash
# 索引./docs目录下的所有文档，构建本地知识库
golem agent --rag ./docs
# 现在可以提问关于文档的问题了
```

#### 👉 开启MCP外部工具
```bash
# 加载MCP服务器，接入外部工具能力
golem agent --mcp '[{"command": "python", "args": ["path/to/mcp-server.py"]}]'
```

#### 👉 开启内置技能
```bash
# 启用代码评审和总结技能
golem agent --skills summarize,code-review
```

---

## 完整使用文档

### Prerequisites

- Go 1.25+
- (Optional) Docker for containerized deployment

### First-run Setup

```bash
# Interactive setup wizard — configures provider, API key, and default model
golem init
```

The wizard supports 7 provider presets: OpenAI, Anthropic, DeepSeek, Moonshot/Kimi, Zhipu/GLM, MiniMax, and DashScope/Qwen. It writes to `~/.golem/config.json`.

### Usage

```bash
# Show help
golem --help

# Print version
golem version

# Start agent (auto-detects TTY → opens Bubble Tea TUI; pipe/redirect → plain output)
golem agent

# Start agent with an initial message (one-shot, no TUI)
golem agent -m "Hello, what can you do?"

# Force plain interactive mode (no TUI)
golem agent --no-tui

# Start HTTP gateway (port 18790)
golem gateway

# Use a specific model
golem agent -M deepseek/deepseek-chat -m "Hello"

# Resume last session
golem agent -C last

# Resume specific session
golem agent -C <session-id>

# Pipe input from another command
echo "Summarize this" | golem agent

# Start agent with MCP server (loads external tools from Model Context Protocol server)
golem agent --mcp '[{"command": "python", "args": ["path/to/mcp-server.py"]}]'
# Or load MCP config from JSON file
golem agent --mcp ./mcp-config.json

# Start agent with RAG (indexes all text files in ./docs directory)
golem agent --rag ./docs
# Or load RAG document list from JSON file
golem agent --rag ./rag-documents.json

# Start agent with built-in skills enabled
golem agent --skills summarize,code-review

# Combine multiple features: RAG + Skills + MCP
golem agent --rag ./docs --skills summarize,code-review --mcp ./mcp-config.json
```

### Configuration Management

Golem stores config at `~/.golem/config.json`. Manage it via CLI:

```bash
# Set a config value
golem config set default_model openai/gpt-4

# Get a config value
golem config get default_model

# List all config values
golem config list

# Use a custom config file
golem --config /path/to/config.json agent -m "hello"
```

### Status & Health Check

```bash
# Show system status (version, config, model info, gateway health)
golem status
```

### Shell Completion

Generate shell completion scripts for your shell:

```bash
# Bash
golem completion bash > /etc/bash_completion.d/golem

# Zsh
golem completion zsh > "${fpath[1]}/_golem"

# Fish
golem completion fish > ~/.config/fish/completions/golem.fish

# PowerShell
golem completion powershell > golem.ps1
```

### Docker

```bash
# Build image
docker build -f docker/Dockerfile -t golem .

# Run with Docker Compose (gateway mode)
docker compose -f docker/docker-compose.yml --profile gateway up

# Run with monitoring stack (Prometheus + Grafana)
docker compose -f docker/monitoring/docker-compose.monitoring.yml up
```

### Environment Variables

Set API keys via environment variables:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Chinese LLM providers
export DEEPSEEK_API_KEY="sk-..."
export MOONSHOT_API_KEY="sk-..."
export ZHIPU_API_KEY="..."
export MINIMAX_API_KEY="..."
export DASHSCOPE_API_KEY="sk-..."
```

Or use the config file approach:

```bash
cp config/config.example.json ~/.golem/config.json
# Edit with your API keys, or use: golem config set ...
```

## Project Structure

```
Golem/
├── cmd/golem/              # CLI entry point (cobra)
├── core/                           # Core domain logic
│   ├── agent/                      # ReAct loop engine
│   ├── bus/                        # Message bus (pub/sub)
│   ├── config/                     # Configuration system with hot reload
│   ├── providers/                  # LLM provider interface
│   │   ├── openai/                 # OpenAI adapter
│   │   └── anthropic/              # Anthropic adapter
│   ├── session/                    # Session + history management
│   ├── tools/                      # Tool interface + registry
│   │   ├── exec/                   # Command execution tool
│   │   ├── fileops/                # File operations tool
│   │   └── websearch/              # Web search tool
│   └── usage/                      # Token usage tracking & pricing
├── foundation/                     # Infrastructure primitives
│   ├── concurrency/                # Pool, semaphore, rate limiter
│   ├── logger/                     # Structured logging (slog)
│   ├── store/                      # SQLite persistence (pure Go)
│   └── term/                       # Terminal detection
├── feature/                        # Optional feature modules
│   ├── mcp/                        # MCP protocol client (wired, enabled via --mcp flag)
│   ├── memory/                     # Long-term memory with importance decay (in development)
│   ├── rag/                        # RAG pipeline with OpenAI embedder (wired, enabled via --rag flag)
│   ├── routing/                    # Error handling + fallback
│   └── skills/                     # Skills registry + built-in skills (wired, enabled via --skills flag)
│       └── builtins/               # Built-in skills (summarize, code-review)
├── internal/                       # Internal-only packages
│   ├── channels/                   # I/O adapters
│   │   ├── cli/                    # CLI adapter
│   │   ├── tui/                    # Bubble Tea TUI (auto-detected on TTY)
│   │   └── telegram/               # Telegram bot adapter
│   ├── gateway/                    # HTTP gateway server with SSE streaming
│   ├── metrics/                    # Prometheus-compatible metrics
│   └── security/                   # Auth, rate limiting, sandbox
├── openspec/                       # OpenSpec SDD specifications
├── docs/study/                     # Learning guides (Chinese)
├── docker/                         # Dockerfile + Compose
│   └── monitoring/                 # Prometheus + Grafana configs
├── k8s/                            # Kubernetes manifests
├── helm/golem/             # Helm chart
├── .github/workflows/              # CI/CD pipelines
├── scripts/                        # Utility scripts
├── Makefile                        # Build automation
└── .golangci.yaml                  # Linter configuration
```

## Testing

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. -benchmem ./internal/gateway/...
```

**Test coverage: 79.2%** across 28 packages (200+ tests, 9 Example functions for godoc).

## Kubernetes Deployment

```bash
# Apply manifests directly
kubectl apply -f k8s/

# Or use Helm
helm install golem helm/golem/
```

## Learning Resources

The `docs/study/` directory contains Chinese learning guides:

1. **Architecture Overview** — Hexagonal architecture and design patterns
2. **Agent ReAct Loop** — How the Think→Act→Observe cycle works
3. **Tool System** — Building a pluggable tool registry
4. **Provider System** — LLM provider abstraction and adapters
5. **Message Bus** — Async event-driven communication
6. **Streaming & Chinese Providers** — SSE streaming, Chinese LLM integration, session resume
7. **TUI Channel & init Wizard** — Bubble Tea Elm architecture, recursive Cmd streaming, Termux compatibility

## Design Principles

- **Pure Go** — Zero CGO dependencies (`CGO_ENABLED=0`), single static binary
- **Layered Architecture** — 4-layer structure (core/foundation/feature/internal) with clean dependency flow
- **Cloud-Native** — Docker, Kubernetes, Helm, Prometheus metrics
- **Security First** — Auth middleware, rate limiting, command sandboxing
- **Test-Driven** — 79.2% coverage, race-detector clean, benchmark suite

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.25+ |
| CLI | [cobra](https://github.com/spf13/cobra) |
| TUI | [Bubble Tea v1.3.10](https://github.com/charmbracelet/bubbletea) + lipgloss |
| Database | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go) |
| Metrics | Custom Prometheus-compatible (no external deps) |
| Container | Docker multi-stage build |
| Orchestration | Kubernetes + Helm |
| CI/CD | GitHub Actions |
| Monitoring | Prometheus + Grafana |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## License

[MIT License](LICENSE)

## Acknowledgments

- Inspired by [PicoClaw](https://github.com/sipeed/picoclaw) by Sipeed
- Built following [OpenSpec SDD](https://github.com/Fission-AI/OpenSpec) workflow
