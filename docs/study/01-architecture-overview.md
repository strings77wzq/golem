# 01 - 架构总览

本文档系统介绍 Golem 的整体架构设计，帮助您建立全局认知，清晰理解各组件的职责划分与协作关系。

## 目录

- [六边形架构原理](#六边形架构原理)
- [项目目录结构](#项目目录结构)
- [核心包职责](#核心包职责)
- [依赖注入模式](#依赖注入模式)
- [与 PicoClaw 的架构对比](#与-picoclaw-的架构对比)
- [系统架构图](#系统架构图)

## 六边形架构原理

Golem 采用**六边形架构**（Hexagonal Architecture，又称端口-适配器架构），这是一种实现业务逻辑与外部依赖完全解耦的先进架构模式。

### 核心设计思想
六边形架构的核心是实现"内外分离"，核心业务逻辑完全不依赖外部系统，所有与外部的交互都通过抽象接口完成：
```
外部系统 → 适配器（实现接口） → 端口（抽象接口） → 核心业务逻辑
```
各层职责定义：
- **核心层（Core）**：包含纯业务逻辑，不依赖任何外部系统或技术实现
- **端口层（Ports）**：定义核心与外部交互的抽象接口契约，是内外交互的唯一入口
- **适配层（Adapters）**：实现端口接口，负责对接具体的外部系统（数据库、API、终端等）

### 架构核心优势
1. **高可测试性**：核心逻辑可以通过 Mock 适配器进行完全独立的单元测试，无需依赖外部环境
2. **高可替换性**：可以无缝替换外部系统实现（如切换 LLM 提供商、更换数据库等），无需修改核心逻辑
3. **关注点分离**：业务逻辑与技术实现细节完全解耦，开发者可以专注于业务逻辑本身
4. **灵活的技术选型**：可以在不影响核心逻辑的前提下推迟技术决策，甚至后期更换技术栈

### 在 Golem 中的落地实现
| 架构层次 | 对应组件 | 功能说明 |
|----------|----------|----------|
| **核心层** | `core/agent/` | ReAct 推理循环核心业务逻辑 |
| **核心层** | `core/session/` | 会话状态与对话历史管理 |
| **端口层** | `bus.Bus` 接口 | 消息传递的抽象契约 |
| **端口层** | `tools.Tool` 接口 | 工具能力的抽象契约 |
| **端口层** | `providers.LLMProvider` 接口 | LLM 调用的抽象契约 |
| **适配层** | `core/bus/memBus` | 内存消息总线的具体实现 |
| **适配层** | `internal/channels/cli/` | 命令行输入输出的适配器实现 |
| **适配层** | `core/providers/openai/` | OpenAI API 的适配器实现 |

## 项目目录结构
Golem 采用严格的四层分层架构，各层职责清晰，依赖关系单向：
```
Golem/
├── cmd/
│   └── golem/        # 组合根（Composition Root）
│       └── main.go           # 应用程序唯一入口，负责组装所有依赖
│
├── core/                     # 核心领域逻辑层（不依赖其他业务层）
│   ├── agent/                # Agent ReAct 推理循环核心实现
│   │   └── agent.go          # Agent 主逻辑入口
│   │
│   ├── bus/                  # 消息总线模块
│   │   ├── bus.go            # Bus 抽象接口与内存实现
│   │   └── message.go        # 入站/出站消息结构定义
│   │
│   ├── config/               # 配置管理模块
│   │   └── config.go         # 配置结构定义与加载逻辑
│   │
│   ├── tools/                # 工具系统模块
│   │   ├── tool.go           # Tool 抽象接口与 ToolResult 定义
│   │   ├── registry.go       # 线程安全的工具注册表
│   │   └── mock.go           # Mock 工具实现（用于单元测试）
│   │
│   ├── providers/            # LLM 提供商抽象层
│   │   ├── types.go          # LLMProvider 接口、消息、工具调用定义
│   │   ├── factory.go        # Provider 工厂（根据模型名路由到具体实现）
│   │   └── mock.go           # Mock Provider 实现（用于单元测试）
│   │
│   ├── session/              # 会话管理模块
│   │   ├── session.go        # Session 结构定义（包含消息历史）
│   │   ├── store.go          # SessionStore 抽象接口与内存实现
│   │   └── history.go        # HistoryManager 实现（会话持久化）
│   │
│   └── usage/                # Token 用量追踪与定价模块
│
├── foundation/               # 基础设施原语层（仅依赖标准库）
│   ├── concurrency/          # 并发原语（协程池、信号量、限流器）
│   ├── logger/               # 结构化日志模块
│   │   └── logger.go         # Logger 抽象接口与实现
│   ├── store/                # SQLite 持久化层（纯 Go 实现，无 CGO 依赖）
│   └── term/                 # 终端环境检测模块
│
├── feature/                  # 可选功能模块层（仅依赖 core 和 foundation）
│   ├── mcp/                  # MCP 协议客户端实现
│   ├── memory/               # 长期记忆模块（支持重要性衰减机制）
│   ├── rag/                  # RAG 检索增强生成管线
│   ├── routing/              # 错误处理与降级路由模块
│   └── skills/               # 技能注册表与内置技能实现
│
├── internal/                 # 内部适配层（仅依赖 core 和 foundation，不对外暴露）
│   ├── channels/             # I/O 适配器（输入输出通道）
│   │   ├── cli/              # 命令行接口适配器实现
│   │   └── telegram/         # Telegram 机器人适配器实现
│   ├── gateway/              # HTTP 网关实现（支持 SSE 流式输出）
│   ├── metrics/              # Prometheus 兼容指标暴露模块
│   └── security/             # 安全模块（认证、限流、沙箱）
│
├── config/                   # 配置文件目录
│   ├── config.example.json   # 配置文件模板
│   └── secrets/              # 敏感配置目录（不纳入 Git 版本控制）
│
├── docs/
│   └── study/                # 学习文档目录（即本指南）
│
├── scripts/                  # 构建与运维工具脚本
├── docker/                   # Docker 镜像配置
├── k8s/                      # Kubernetes 部署配置
├── build/                    # 构建输出目录（自动生成）
├── go.mod                    # Go 模块定义文件
└── Makefile                  # 构建任务配置
```

## 核心包职责

### 1. `cmd/golem/` - 组合根

**职责**：应用程序入口点，负责组装所有依赖并启动系统。

```go
// main.go 的典型结构
func main() {
    // 1. 加载配置
    cfg := config.Load()
    
    // 2. 创建基础设施
    logger := logger.New()
    bus := bus.New()
    
    // 3. 创建工具注册表
    registry := tools.NewRegistry()
    registry.Register(...)
    
    // 4. 创建 Provider 工厂
    factory := providers.NewFactory()
    factory.Register("openai", openaiProvider)
    
    // 5. 创建会话管理
    store := session.NewMemoryStore()
    history := session.NewHistoryManager()
    
    // 6. 创建 Agent（注入所有依赖）
    agent := agent.New(bus, registry, factory, store, history, logger, cfg)
    
    // 7. 创建 CLI 通道
    cli := cli.New(bus, logger)
    
    // 8. 启动所有组件
    go agent.Start(ctx)
    go cli.Start(ctx)
    
    // 9. 等待退出信号
    <-ctx.Done()
}
```

**设计原则**：所有依赖在这里创建和注入，其他包不直接创建依赖。

### 2. `core/agent/` - ReAct 循环核心

**职责**：实现 AI 的推理-行动循环（Reason-Act Loop）。

**核心流程**：
1. 监听 `inbound` 主题的消息（用户输入）
2. 获取或创建会话
3. 将消息添加到会话历史
4. 调用 LLM（附带工具定义）
5. 如果 LLM 返回文本 → 发布到 `outbound` 主题，结束
6. 如果 LLM 返回工具调用 → 执行工具 → 将结果添加到历史 → 回到步骤 4
7. 最大迭代次数保护（防止无限循环）

**关键代码**（参见 `core/agent/agent.go`）：

```go
type Agent struct {
    bus               bus.Bus
    toolRegistry      *tools.Registry
    providerFactory   *providers.Factory
    sessionStore      session.SessionStore
    historyManager    *session.HistoryManager
    logger            logger.Logger
    config            *config.Config
    systemPrompt      string
    maxToolIterations int
}
```

### 3. `core/bus/` - 消息总线

**职责**：基于 Pub/Sub 模式的事件总线，解耦组件间通信。

**接口定义**（参见 `core/bus/bus.go` 第 6-11 行）：

```go
type Bus interface {
    Publish(topic string, msg interface{})
    Subscribe(topic string) <-chan interface{}
    Unsubscribe(topic string, ch <-chan interface{})
    Close()
}
```

**消息类型**（参见 `core/bus/message.go`）：
- `InboundMessage`：进入系统的消息（用户输入）
- `OutboundMessage`：离开系统的消息（AI 响应）

**实现细节**：
- 线程安全（使用 `sync.RWMutex`）
- 非阻塞发布（使用 `select`）
- 缓冲通道（容量 100）

### 4. `core/tools/` - 工具系统

**职责**：定义工具接口、工具注册表、工具执行结果。

**核心接口**（参见 `core/tools/tool.go` 第 6-11 行）：

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() []ToolParameter
    Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}
```

**双通道结果**（参见 `core/tools/tool.go` 第 24-29 行）：

```go
type ToolResult struct {
    ForLLM  string // 总是发送给 LLM 作为上下文
    ForUser string // 立即显示给用户（可为空）
    IsError bool   // 指示工具执行失败
    Silent  bool   // 如果为 true，即使 ForUser 非空也不显示
}
```

**工具注册表**：
- 线程安全的工具管理（`sync.RWMutex`）
- **按字母顺序返回工具**（参见 `core/tools/registry.go` 第 47-66 行）
- 原因：保持工具顺序一致，LLM 可以重用 KV 缓存，提升性能

### 5. `core/providers/` - LLM 提供商抽象

**职责**：定义 LLM 调用接口，支持多种提供商（OpenAI、Anthropic 等）。

**核心接口**（参见 `core/providers/types.go` 第 58-67 行）：

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition, 
         model string, opts *ChatOptions) (*LLMResponse, error)
    Name() string
}
```

**工厂模式**（参见 `core/providers/factory.go`）：
- 根据模型名称路由到具体 Provider
- 例如：`"openai/gpt-4"` → 提取 `"openai"` → 返回 OpenAI Provider
- 支持动态注册新的 Provider

### 6. `core/session/` - 会话管理

**职责**：管理对话历史和会话状态。

**Session 结构**（参见 `core/session/session.go` 第 11-17 行）：

```go
type Session struct {
    ID        string
    Messages  []providers.Message
    CreatedAt time.Time
    UpdatedAt time.Time
    mu        sync.RWMutex
}
```

**线程安全操作**：
- `AddMessage()`：添加消息到历史
- `GetMessages()`：返回消息副本（防止并发修改）
- `MessageCount()`：获取消息数量

### 7. `internal/channels/` - I/O 适配器

**职责**：连接外部 I/O 系统（CLI、HTTP、WebSocket 等）。

**当前实现**：
- `channels/cli/`：命令行接口
  - 读取用户输入
  - 发布 `InboundMessage` 到总线
  - 订阅 `OutboundMessage`，打印到终端

**未来扩展**：
- `channels/http/`：HTTP API 服务器
- `channels/websocket/`：WebSocket 实时通信
- `channels/slack/`：Slack 集成

### 8. `core/config/` - 配置管理

**职责**：加载和管理应用配置。

**配置内容**：
- LLM 模型选择
- API 密钥（从环境变量或配置文件）
- 日志级别
- Agent 系统提示词
- 最大工具迭代次数

### 9. `foundation/logger/` - 结构化日志

**职责**：提供统一的日志接口。

**特性**：
- 结构化日志（JSON 格式）
- 日志级别（Debug、Info、Warn、Error）
- 上下文字段（requestID、sessionID 等）

## 依赖注入模式

Golem 使用**构造函数注入**模式，所有依赖在创建对象时传入。

### 示例：Agent 的创建

```go
// core/agent/agent.go
func New(
    b bus.Bus,
    registry *tools.Registry,
    factory *providers.Factory,
    store session.SessionStore,
    history *session.HistoryManager,
    log logger.Logger,
    cfg *config.Config,
    opts ...Option,
) *Agent {
    // 所有依赖都从外部注入，Agent 不创建任何依赖
    return &Agent{
        bus:             b,
        toolRegistry:    registry,
        providerFactory: factory,
        sessionStore:    store,
        historyManager:  history,
        logger:          log,
        config:          cfg,
    }
}
```

### 优势

1. **可测试性**：可以注入 Mock 对象进行单元测试
2. **灵活性**：可以在运行时选择不同的实现
3. **显式依赖**：从函数签名就能看出所有依赖
4. **单一职责**：每个包只负责自己的逻辑，不负责创建依赖

### 测试示例

```go
func TestAgent(t *testing.T) {
    // 创建 Mock 依赖
    mockBus := &MockBus{}
    mockRegistry := tools.NewRegistry()
    mockFactory := &MockProviderFactory{}
    mockStore := session.NewMemoryStore()
    mockLogger := &MockLogger{}
    mockConfig := &config.Config{}
    
    // 注入 Mock 依赖
    agent := agent.New(
        mockBus,
        mockRegistry,
        mockFactory,
        mockStore,
        nil, // history
        mockLogger,
        mockConfig,
    )
    
    // 测试 Agent 逻辑，无需真实的 LLM 或数据库
}
```

## 与 PicoClaw 的架构对比

| 方面 | PicoClaw (Python) | Golem (Go) |
|------|-------------------|---------------------|
| **语言** | Python | Go |
| **架构风格** | 模块化 | 六边形架构 |
| **组件通信** | 直接调用 | 消息总线（解耦） |
| **并发模型** | asyncio | goroutine + channel |
| **类型系统** | 动态类型 | 静态类型 + 接口 |
| **依赖管理** | 部分依赖注入 | 完全依赖注入 |
| **测试策略** | Mock 工具 | Mock 所有接口 |
| **配置** | YAML/JSON | JSON + 环境变量 |
| **可扩展性** | 插件系统 | 接口 + 注册表 |

### 设计改进

1. **消息总线解耦**：
   - PicoClaw：CLI → Agent（直接调用）
   - Golem：CLI → Bus → Agent（通过消息）
   - 优势：可以轻松添加新的输入输出通道（HTTP、WebSocket）

2. **工具顺序优化**：
   - Golem 按字母顺序返回工具，利用 LLM 的 KV 缓存
   - 参见 `core/tools/registry.go` 第 47-50 行的注释

3. **Provider 工厂**：
   - 支持运行时动态注册 Provider
   - 模型名称包含 vendor 前缀（`vendor/model`）

4. **线程安全**：
   - 所有共享状态都使用 `sync.RWMutex` 保护
   - Go 的并发模型更适合高性能场景

## 系统架构图

```mermaid
graph TB
    subgraph "外部世界（Adapters）"
        CLI[CLI Channel]
        HTTP[HTTP Channel<br/>未来]
        OpenAI[OpenAI Provider]
        Anthropic[Anthropic Provider<br/>未来]
    end
    
    subgraph "端口层（Ports - Interfaces）"
        BusInterface[Bus Interface]
        ToolInterface[Tool Interface]
        ProviderInterface[LLMProvider Interface]
    end
    
    subgraph "核心业务逻辑（Core）"
        Agent[Agent<br/>ReAct Loop]
        Session[Session Manager<br/>对话状态]
        ToolRegistry[Tool Registry<br/>工具管理]
        ProviderFactory[Provider Factory<br/>路由]
    end
    
    subgraph "基础设施（Infrastructure）"
        Bus[Message Bus<br/>memBus]
        Config[Config Loader]
        Logger[Logger]
    end
    
    CLI -->|Publish/Subscribe| BusInterface
    HTTP -.->|未来| BusInterface
    BusInterface --> Bus
    
    Bus <-->|InboundMessage<br/>OutboundMessage| Agent
    
    Agent -->|Register/Get| ToolRegistry
    Agent -->|GetProvider| ProviderFactory
    Agent -->|Get/Update| Session
    
    ToolRegistry --> ToolInterface
    ProviderFactory --> ProviderInterface
    
    ProviderInterface -.->|实现| OpenAI
    ProviderInterface -.->|实现| Anthropic
    
    Agent --> Logger
    Agent --> Config
    
    style Agent fill:#ff9999
    style Session fill:#ff9999
    style CLI fill:#99ccff
    style OpenAI fill:#99ccff
    style BusInterface fill:#99ff99
    style ToolInterface fill:#99ff99
    style ProviderInterface fill:#99ff99
```

### 架构图说明

- **红色**：核心业务逻辑（Core）
- **蓝色**：外部适配器（Adapters）
- **绿色**：端口接口（Ports）
- **实线**：已实现
- **虚线**：未来计划

### 数据流示例

1. **用户发送消息**：
   ```
   用户输入 → CLI.ReadInput() 
   → Bus.Publish("inbound", InboundMessage) 
   → Agent 收到消息
   ```

2. **Agent 处理**：
   ```
   Agent 获取 Session
   → Agent 调用 LLMProvider.Chat()
   → LLM 返回工具调用
   → Agent 从 ToolRegistry 获取工具
   → 执行 Tool.Execute()
   → 将结果添加到 Session
   → 再次调用 LLM
   → LLM 返回最终答案
   ```

3. **返回响应**：
   ```
   Agent → Bus.Publish("outbound", OutboundMessage)
   → CLI 收到消息
   → CLI 打印到终端
   ```

## 小结

Golem 的架构设计遵循以下原则：

1. **依赖倒置**：核心逻辑依赖抽象（接口），不依赖具体实现
2. **关注点分离**：每个包职责单一，边界清晰
3. **可测试性**：所有依赖可注入，方便 Mock 测试
4. **可扩展性**：通过接口和注册表模式，易于添加新功能
5. **并发安全**：所有共享状态都有适当的同步机制

下一步，我们将深入学习 **Agent 的 ReAct 循环**，理解 AI 如何进行推理和工具调用。

👉 [下一章：Agent ReAct 循环](./02-agent-react-loop.md)
