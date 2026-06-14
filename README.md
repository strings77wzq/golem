# Golem

[![CI](https://github.com/strings77wzq/golem/actions/workflows/ci.yml/badge.svg)](https://github.com/strings77wzq/golem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/strings77wzq/golem)](https://goreportcard.com/report/github.com/strings77wzq/golem)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/strings77wzq/golem)](https://github.com/strings77wzq/golem/releases)

**一个二进制文件，连上你的数据库，其他 AI agent 就能查询你的数据。**

Golem 是一个 Go 原生的数据库 AI agent。它连接你的 SQLite 数据库，理解你的表结构，然后通过 MCP 协议把查询能力暴露给 Claude Code、Cursor 或任何支持 MCP 的 agent。

不需要 Python。不需要 pip。下载一个二进制文件，指定数据库路径，开始使用。

---

## 为什么需要 Golem？

如果你用 AI agent（Claude Code、Cursor、OpenClaw），你会发现一个共同的问题：**它们看不到你的数据。**

你可以让 AI 写代码、读文件、执行命令，但它无法直接查询你的 SQLite 数据库来回答 "上个月有多少用户注册？" 这样的问题。

Golem 解决这个问题：

```
你的 SQLite 数据库
        │
        ▼
┌─────────────────┐
│  Golem Agent    │  ← 连接数据库，理解 schema
│  (单二进制)      │
└────────┬────────┘
         │ MCP 协议 (stdio)
         ▼
┌─────────────────┐
│  Claude Code    │  ← 通过 MCP 调用 sql_query 工具
│  / Cursor       │
│  / OpenClaw     │
└─────────────────┘
```

---

## 5 分钟上手

```bash
# 1. 安装
go install github.com/strings77wzq/golem/cmd/golem@latest

# 2. 创建演示数据库
golem demo-db

# 3. 让 agent 分析你的数据
golem agent --db .golem-demo.db -m "这个数据库有哪些表？有什么性能问题？"
```

---

## 核心功能

### 数据库智能

```bash
# 连接数据库，用自然语言查询
golem agent --db ./myapp.db -m "查询最近 7 天注册的用户"

# 分析表结构
golem agent --db ./myapp.db -m "分析 orders 表的索引是否合理"

# 发现性能问题
golem agent --db ./myapp.db -m "这个数据库有什么需要优化的地方？"
```

**安全机制：**
- 默认只读（SELECT），写操作需要显式授权
- UPDATE/DELETE 必须带 WHERE 子句，否则拒绝执行
- 自动为 DELETE/UPDATE 生成回滚 SQL
- 所有操作记录审计日志

### MCP Server — 让其他 agent 调用你的数据库

```bash
# 启动 MCP server，暴露数据库工具
golem mcp-server --db ./myapp.db
```

其他 agent 可以通过 MCP 协议调用这些工具：

| 工具 | 功能 |
|------|------|
| `sql_query` | 执行 SQL SELECT 查询 |
| `sql_schema` | 获取数据库 schema |
| `sql_analyze` | 分析表数据分布 |
| `exec` | 执行 shell 命令（可禁用） |
| `file_read` / `file_write` | 文件操作 |

在 Claude Code 的 MCP 配置中添加：

```json
{
  "golem": {
    "command": "golem",
    "args": ["mcp-server", "--db", "./myapp.db"]
  }
}
```

### HTTP Gateway — OpenAI 兼容 API

```bash
golem gateway    # 启动在 :18790
```

```bash
# 兼容 OpenAI API 格式
curl http://localhost:18790/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"查询所有管理员"}]}'
```

---

## 安全设计

Golem 对数据库操作有严格的安全控制：

| 操作 | 默认行为 | 授权后 |
|------|----------|--------|
| SELECT 查询 | ✅ 允许 | ✅ |
| INSERT/UPDATE | ❌ 拒绝 | ✅ (必须带 WHERE) |
| DELETE | ❌ 拒绝 | ✅ (必须带 WHERE) |
| Shell 命令 | ⚠️ 白名单 | ✅ |

写操作会自动生成回滚 SQL，方便恢复数据。

---

## 架构

```
┌─────────────────────────────────────────────────┐
│                    cmd/golem/                     │
│         CLI 入口，组合所有层（Composition Root）    │
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
│         并发原语 │ Logger │ Store                  │
└─────────────────────────────────────────────────┘

依赖方向：cmd → internal → core → foundation
          feature → core + foundation
          foundation → 仅 stdlib
```

---

## 项目结构

```
golem/
├── cmd/golem/         # CLI 入口和组合根
├── core/              # 领域逻辑
│   ├── agent/         # ReAct agent 循环
│   ├── tools/         # 工具实现 (exec, fileops, database, websearch)
│   ├── database/      # 数据库驱动 (SQLite, PostgreSQL, MySQL, Redis, Qdrant)
│   ├── providers/     # LLM 提供商 (OpenAI, Anthropic, Ollama 等)
│   ├── security/      # 权限检查、质量门控
│   └── session/       # 会话管理 (SQLite + 内存)
├── feature/           # 可选功能（通过 CLI flag 启用）
│   ├── mcp/           # MCP client + server
│   ├── rag/           # RAG 管道 (TF-IDF)
│   ├── skills/        # 技能注册表
│   └── memory/        # 长期记忆（重要性衰减）
├── internal/          # 内部适配器
│   ├── channels/      # CLI, TUI, Telegram
│   ├── gateway/       # HTTP gateway
│   └── metrics/       # Prometheus 指标
└── foundation/        # 基础设施原语
```

---

## API 参考

### Gateway 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/metrics` | Prometheus 指标 |
| POST | `/api/chat` | 同步对话 |
| POST | `/api/chat/stream` | SSE 流式对话 |
| POST | `/v1/chat/completions` | OpenAI 兼容 API |
| GET | `/v1/models` | 列出可用模型 |
| GET | `/api/sessions/{id}/export` | 导出会话 |
| POST | `/api/sessions/import` | 导入会话 |

### MCP 工具

| 工具 | 参数 | 说明 |
|------|------|------|
| `sql_query` | `sql`, `args?`, `database?` | 执行 SELECT 查询 |
| `sql_schema` | `table?`, `database?` | 获取 schema |
| `sql_analyze` | `table`, `database?` | 分析表数据分布 |
| `exec` | `command`, `timeout?` | 执行 shell 命令 |
| `file_read` | `path` | 读取文件 |
| `file_write` | `path`, `content` | 写入文件 |
| `web_search` | `query`, `max_results?` | 搜索网页 |

---

## 支持的 LLM 提供商

| 提供商 | 本地/云端 | 说明 |
|--------|-----------|------|
| Ollama | 本地 | 完全离线运行 |
| OpenAI | 云端 | GPT-4o, GPT-4 |
| Anthropic | 云端 | Claude 3.5 Sonnet |
| DeepSeek | 云端 | DeepSeek Chat |
| MiMo | 云端 | MiMo |
| Kimi | 云端 | Moonshot |
| GLM | 云端 | 智谱 |
| MiniMax | 云端 | MiniMax |
| Qwen | 云端 | 通义千问 |

也支持任何 OpenAI 兼容的 API（vLLM、OpenRouter、LiteLLM）。

---

## 测试

```bash
go test ./...                    # 38 个包
go test -race ./...              # 竞态检测
go test -coverprofile=out ./...  # 覆盖率
```

---

## 贡献

欢迎贡献！请先阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。

核心原则：
- `CGO_ENABLED=0` 是强制的
- 分层边界必须严格（cmd → internal → core → foundation）
- 每个 PR 需要附带测试
- 提交信息使用 Conventional Commits 格式

---

## License

MIT License

## 致谢

- 受 [PicoClaw](https://github.com/sipeed/picoclaw) 启发
