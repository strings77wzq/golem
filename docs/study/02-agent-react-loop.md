# 02 - Agent ReAct 推理循环

本文档深入讲解 Golem 的核心模块——Agent ReAct 推理循环。理解这一核心工作流是掌握整个系统运行机制的关键。

## 目录

- [什么是 ReAct 模式](#什么是-react-模式)
- [ReAct 循环工作流程](#react-循环工作流程)
- [核心代码解析](#核心代码解析)
- [关键设计决策](#关键设计决策)
- [错误处理和安全机制](#错误处理和安全机制)
- [ReAct 循环流程图](#react-循环流程图)

## 什么是 ReAct 模式

**ReAct** = **Rea**son（推理）+ A**ct**（行动），是当前主流的 Agent 工作模式。
与传统大模型一次性生成答案的方式不同，ReAct 模式让 AI 可以模拟人类解决问题的过程，交替进行推理和行动：
1. **问题分析（Reason）**：理解用户需求，决策下一步需要执行的操作
2. **工具执行（Act）**：调用外部工具获取信息或执行具体操作
3. **结果观察（Observe）**：接收并处理工具返回的执行结果
4. **迭代推理**：基于工具结果继续思考，判断是否需要调用更多工具
5. **答案生成**：综合所有信息，生成最终回答反馈给用户

### 示例对话流程

```
用户：北京今天天气如何？

[第 1 轮]
Agent 推理：我需要获取北京的天气信息
Agent 行动：调用 get_weather(city="北京")
工具结果：晴天，温度 15°C，湿度 40%

[第 2 轮]
Agent 推理：我已经有了天气信息，可以回答用户
Agent 回答：北京今天天气晴朗，温度 15°C，湿度 40%，适合外出活动。
```

### ReAct 模式 vs 传统思维链（Chain-of-Thought）
| 对比维度 | 传统思维链（Chain-of-Thought） | ReAct 模式 |
|----------|--------------------------------|------------|
| **推理方式** | 纯文本内部推理 | 推理与工具调用交替进行 |
| **信息来源** | 仅依赖模型训练时的知识库 | 模型知识 + 外部工具实时信息 |
| **信息时效性** | 受限于训练数据截止时间 | 可获取最新的实时信息 |
| **结果可靠性** | 容易产生幻觉（虚构信息） | 基于工具真实返回结果，可靠性更高 |
| **任务适应性** | 难以完成需要外部信息的复杂任务 | 支持多步骤复杂任务的自动拆解与执行 |

## ReAct 循环工作流程
Golem 的 Agent 模块实现了完整的 ReAct 推理循环，核心工作流步骤如下：

### 步骤1：接收入站消息
Agent 启动后会订阅消息总线的 `inbound` 主题，持续监听并接收用户输入消息。

```go
// core/agent/agent.go 第 81-103 行
func (a *Agent) Start(ctx context.Context) {
    // 订阅 inbound 主题
    ch := a.bus.Subscribe(TopicInbound)
    defer a.bus.Unsubscribe(TopicInbound, ch)
    
    for {
        select {
        case <-ctx.Done():
            return
        case raw := <-ch:
            msg, ok := raw.(bus.InboundMessage)
            if !ok {
                a.logger.Error("invalid inbound message type", nil)
                continue
            }
            a.handleMessage(ctx, msg)
        }
    }
}
```

**InboundMessage 结构**（参见 `core/bus/message.go` 第 14-18 行）：
```go
type InboundMessage struct {
    SessionID string  // 会话 ID
    Content   string  // 用户消息内容
    Role      Role    // 消息角色（通常是 "user"）
}
```

### 步骤2：获取或创建会话
每个对话都对应一个独立的会话（Session）对象，用于存储完整的消息历史和会话状态。

```go
func (a *Agent) handleMessage(ctx context.Context, msg bus.InboundMessage) {
    // 从 SessionStore 获取或创建会话
    sess := a.sessionStore.Get(msg.SessionID)
    if sess == nil {
        sess = session.NewSession(msg.SessionID)
        a.sessionStore.Set(sess)
    }
    
    // 添加系统提示词（如果是新会话）
    if sess.MessageCount() == 0 {
        sess.AddMessage(providers.Message{
            Role:    providers.RoleSystem,
            Content: a.systemPrompt,
        })
    }
    
    // 添加用户消息到历史
    sess.AddMessage(providers.Message{
        Role:    providers.RoleUser,
        Content: msg.Content,
    })
}
```

### 步骤3：准备工具定义
从工具注册表获取所有已注册的工具，转换为 LLM 能够识别的标准化格式。

```go
// 获取所有工具定义（按字母顺序）
toolDefs := a.toolRegistry.ListDefinitions()
```

工具定义示例：
```json
{
  "name": "get_weather",
  "description": "获取指定城市的天气信息",
  "parameters": [
    {
      "name": "city",
      "type": "string",
      "description": "城市名称",
      "required": true
    }
  ]
}
```

### 步骤4：调用大模型（LLM）
将会话历史和可用工具定义发送给 LLM 进行推理。

```go
// 获取 Provider
provider, modelName, err := a.providerFactory.GetProviderForModel(a.config.Model)
if err != nil {
    a.logger.Error("failed to get provider", err)
    return
}

// 调用 LLM
response, err := provider.Chat(
    ctx,
    sess.GetMessages(),  // 会话历史
    toolDefs,            // 可用工具
    modelName,           // 模型名称
    nil,                 // 选项（temperature 等）
)
```

### 步骤5：处理 LLM 响应
LLM 的响应通常分为两种类型：

#### 情况A：直接文本响应（无工具调用）

```go
if len(response.ToolCalls) == 0 {
    // 将 AI 回答添加到会话
    sess.AddMessage(providers.Message{
        Role:    providers.RoleAssistant,
        Content: response.Content,
    })
    
    // 发布出站消息
    a.bus.Publish(TopicOutbound, bus.OutboundMessage{
        SessionID: msg.SessionID,
        Content:   response.Content,
        Role:      bus.RoleAssistant,
        Done:      true,
    })
    
    return // 结束循环
}
```

#### 情况B：工具调用请求

```go
if len(response.ToolCalls) > 0 {
    // 将 AI 的工具调用请求添加到会话
    sess.AddMessage(providers.Message{
        Role:      providers.RoleAssistant,
        Content:   response.Content,
        ToolCalls: response.ToolCalls,
    })
    
    // 执行每个工具调用
    for _, toolCall := range response.ToolCalls {
        result := a.executeTool(ctx, toolCall)
        
        // 如果有 ForUser 内容，立即发送给用户
        if result.ForUser != "" && !result.Silent {
            a.bus.Publish(TopicOutbound, bus.OutboundMessage{
                SessionID: msg.SessionID,
                Content:   result.ForUser,
                Role:      bus.RoleTool,
                Done:      false,
            })
        }
        
        // 将工具结果添加到会话（ForLLM 内容）
        sess.AddMessage(providers.Message{
            Role:       providers.RoleTool,
            Content:    result.ForLLM,
            ToolCallID: toolCall.ID,
        })
    }
    
    // 回到步骤 4，再次调用 LLM
}
```

### 步骤6：循环迭代
重复执行步骤4-5，直到满足以下任一终止条件：
- LLM 返回直接文本响应（无工具调用需求）
- 达到最大迭代次数限制（默认25次，防止无限循环）

```go
const MaxToolIterations = 25

for iteration := 0; iteration < MaxToolIterations; iteration++ {
    response, err := provider.Chat(...)
    
    if len(response.ToolCalls) == 0 {
        // 返回最终答案
        break
    }
    
    // 执行工具，继续循环
}

if iteration >= MaxToolIterations {
    a.logger.Warn("reached max tool iterations")
}
```

## 核心代码解析

### Agent 结构体定义
核心结构体定义参见 `core/agent/agent.go` 第20-31行：

```go
type Agent struct {
    bus               bus.Bus                    // 消息总线
    toolRegistry      *tools.Registry            // 工具注册表
    providerFactory   *providers.Factory         // Provider 工厂
    sessionStore      session.SessionStore       // 会话存储
    historyManager    *session.HistoryManager    // 历史管理
    logger            logger.Logger              // 日志记录器
    config            *config.Config             // 配置
    systemPrompt      string                     // 系统提示词
    maxToolIterations int                        // 最大工具迭代次数
}
```

**所有依赖均通过构造函数注入**，Agent 自身不创建任何依赖对象，完全符合依赖注入的最佳实践。

### 工具执行逻辑实现

```go
func (a *Agent) executeTool(ctx context.Context, toolCall providers.ToolCall) *tools.ToolResult {
    // 从注册表获取工具
    tool, ok := a.toolRegistry.Get(toolCall.Name)
    if !ok {
        return &tools.ToolResult{
            ForLLM:  fmt.Sprintf("Error: tool %q not found", toolCall.Name),
            ForUser: "",
            IsError: true,
        }
    }
    
    // 执行工具
    result, err := tool.Execute(ctx, toolCall.Arguments)
    if err != nil {
        return &tools.ToolResult{
            ForLLM:  fmt.Sprintf("Error executing tool: %v", err),
            ForUser: "",
            IsError: true,
        }
    }
    
    return result
}
```

**核心设计亮点**：工具返回的 `ToolResult` 采用双通道设计：
- `ForLLM`：内容总是发送给 LLM 作为推理上下文
- `ForUser`：可选内容，会立即展示给终端用户

### 消息历史管理

会话（Session）维护完整的消息历史：

```
[系统提示词]
User: 北京天气如何？
Assistant: [ToolCalls: get_weather(city="北京")]
Tool: 晴天，15°C
Assistant: 北京今天天气晴朗，温度 15°C...
```

这个历史在每次调用 LLM 时都会发送，让 LLM 能够：
- 理解对话上下文
- 知道之前调用了哪些工具
- 基于工具结果进行推理

## 核心设计决策解析

### 决策1：为什么采用消息总线架构？

**核心问题**：CLI 与 Agent 之间如何实现通信？

**方案A：直接调用模式**
```go
cli.OnUserInput(func(input string) {
    response := agent.Process(input)
    cli.Print(response)
})
```

**方案B：消息总线模式（最终采用）**
```go
// CLI
cli.OnUserInput(func(input string) {
    bus.Publish("inbound", InboundMessage{Content: input})
})
bus.Subscribe("outbound").OnMessage(func(msg OutboundMessage) {
    cli.Print(msg.Content)
})

// Agent
bus.Subscribe("inbound").OnMessage(func(msg InboundMessage) {
    response := agent.Process(msg)
    bus.Publish("outbound", response)
})
```

**为什么最终选择方案B？**
1. **完全解耦**：CLI 和 Agent 互不依赖，可独立进行单元测试
2. **高可扩展**：可以无缝添加 HTTP、WebSocket 等其他输入输出通道
3. **异步处理**：Agent 可以异步处理消息，不会阻塞 CLI 交互
4. **多订阅者支持**：多个组件可以同时监听同一主题（如日志记录、监控系统等）

### 决策2：为什么会话管理与 Agent 分离？

**Session** 是独立的包（`core/session/`），不是 Agent 的内部状态。

**核心优势**：
1. **单一职责原则**：Agent 专注于推理循环逻辑，Session 专注于会话状态管理
2. **高可测试性**：可以独立测试 Session 模块的线程安全性
3. **高可替换性**：可以无缝替换 SessionStore 实现（内存 → Redis → 数据库）
4. **高可复用性**：其他组件也可以直接使用 Session 模块（如历史查询、数据分析等）

### 决策3：工具定义为什么要按字母顺序排序？

参见 `core/tools/registry.go` 第 47-50 行注释：

```go
// CRITICAL: Alphabetical ordering is required for LLM KV cache optimization.
// When tools are always presented in the same order, the LLM can reuse its KV cache.
```

**原理**：
- LLM 使用 KV 缓存来加速推理
- 如果工具列表顺序改变，KV 缓存失效
- 按字母顺序排序，保证每次顺序一致
- LLM 可以重用缓存，显著提升性能

### 决策4：为什么要设置最大迭代次数限制？

```go
const DefaultMaxToolIterations = 25
```

**原因**：
1. **防止无限循环**：AI 可能陷入工具调用死循环
2. **成本控制**：每次 LLM 调用都有成本（API 费用）
3. **用户体验**：避免用户长时间等待

**实际案例**：
- 正常对话：1-3 次迭代
- 复杂任务：5-10 次迭代
- 异常情况：如果达到 25 次，说明出现了问题

## 错误处理与安全机制

### 1. 工具执行错误处理

```go
result, err := tool.Execute(ctx, args)
if err != nil {
    // 将错误信息返回给 LLM
    return &tools.ToolResult{
        ForLLM:  fmt.Sprintf("Error: %v", err),
        IsError: true,
    }
}
```

**设计**：不中断循环，将错误返回给 LLM，让 LLM 决定如何处理。

### 2. LLM Provider 错误处理

```go
response, err := provider.Chat(ctx, ...)
if err != nil {
    a.logger.Error("LLM call failed", err)
    a.bus.Publish(TopicOutbound, bus.OutboundMessage{
        Content: "抱歉，AI 服务暂时不可用，请稍后重试。",
        Done:    true,
    })
    return
}
```

**设计**：记录日志，向用户返回友好的错误消息。

### 3. 上下文取消与优雅关闭

```go
func (a *Agent) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            a.logger.Info("agent stopped by context")
            return
        case msg := <-ch:
            a.handleMessage(ctx, msg)
        }
    }
}
```

**设计**：支持优雅关闭，通过 `context.Context` 传递取消信号。

### 4. 并发线程安全保障

- **Session**：使用 `sync.RWMutex` 保护消息历史
- **ToolRegistry**：使用 `sync.RWMutex` 保护工具映射
- **Bus**：使用 `sync.RWMutex` 保护订阅者列表

## ReAct 循环可视化流程图

```mermaid
flowchart TD
    Start([开始]) --> Subscribe[订阅 inbound 主题]
    Subscribe --> Wait[等待消息]
    Wait --> Receive{收到消息?}
    Receive -->|Context 取消| Stop([停止])
    Receive -->|收到 InboundMessage| GetSession[获取/创建 Session]
    
    GetSession --> CheckSystem{是否新会话?}
    CheckSystem -->|是| AddSystem[添加系统提示词]
    CheckSystem -->|否| AddUser[添加用户消息]
    AddSystem --> AddUser
    
    AddUser --> InitIter[迭代次数 = 0]
    InitIter --> GetTools[获取工具定义]
    GetTools --> GetProvider[获取 Provider]
    GetProvider --> CallLLM[调用 LLM]
    
    CallLLM --> CheckError{调用成功?}
    CheckError -->|失败| LogError[记录错误]
    LogError --> SendError[发送错误消息]
    SendError --> Wait
    
    CheckError -->|成功| CheckToolCalls{有工具调用?}
    
    CheckToolCalls -->|无| AddAssistant[添加 AI 消息到历史]
    AddAssistant --> PublishFinal[发布最终响应]
    PublishFinal --> Wait
    
    CheckToolCalls -->|有| AddToolRequest[添加工具请求到历史]
    AddToolRequest --> LoopTools[遍历每个工具调用]
    
    LoopTools --> GetTool{工具存在?}
    GetTool -->|不存在| ToolNotFound[返回错误结果]
    GetTool -->|存在| ExecuteTool[执行工具]
    
    ToolNotFound --> AddToolResult[添加工具结果到历史]
    ExecuteTool --> CheckForUser{有 ForUser 内容?}
    CheckForUser -->|有且非 Silent| PublishToolOutput[立即发布工具输出]
    CheckForUser -->|无或 Silent| AddToolResult
    PublishToolOutput --> AddToolResult
    
    AddToolResult --> MoreTools{还有更多工具?}
    MoreTools -->|是| LoopTools
    MoreTools -->|否| IncIter[迭代次数 + 1]
    
    IncIter --> CheckMaxIter{达到最大迭代次数?}
    CheckMaxIter -->|是| WarnMaxIter[警告：达到最大迭代次数]
    WarnMaxIter --> Wait
    CheckMaxIter -->|否| CallLLM
    
    style Start fill:#90EE90
    style Stop fill:#FFB6C1
    style CallLLM fill:#FFD700
    style ExecuteTool fill:#87CEEB
    style CheckToolCalls fill:#FFA500
    style PublishFinal fill:#90EE90
```

### 流程图说明
- **绿色节点**：开始/结束/成功状态
- **黄色节点**：核心操作（LLM 调用）
- **蓝色节点**：工具执行操作
- **橙色节点**：关键决策分支点

### 典型执行路径示例

**简单问答**（无工具调用）：
```
Wait → Receive → GetSession → AddUser → CallLLM → PublishFinal → Wait
```

**工具调用**（1 次迭代）：
```
Wait → Receive → GetSession → AddUser → CallLLM 
→ ExecuteTool → AddToolResult → CallLLM → PublishFinal → Wait
```

**多轮工具调用**（2 次迭代）：
```
Wait → Receive → GetSession → AddUser → CallLLM
→ ExecuteTool → CallLLM
→ ExecuteTool → CallLLM → PublishFinal → Wait
```

## 实战示例

### 示例 1：获取天气信息

```
[用户输入]
User: 北京今天天气怎么样？

[第 1 轮 LLM 调用]
LLM 推理：用户想知道天气，我需要调用 get_weather 工具
LLM 响应：ToolCalls: [get_weather(city="北京")]

[工具执行]
执行 get_weather(city="北京")
工具返回：
  ForLLM: "北京：晴天，温度 15°C，湿度 40%，风力 2 级"
  ForUser: "🌤️ 正在查询北京天气..."

[第 2 轮 LLM 调用]
LLM 推理：我已经有了天气数据，可以给出友好的回答
LLM 响应：Content: "北京今天天气晴朗，温度 15°C，适合外出活动。"

[最终响应]
Assistant: 北京今天天气晴朗，温度 15°C，适合外出活动。
```

### 示例 2：多步骤任务

```
[用户输入]
User: 帮我查一下北京和上海今天哪个城市更适合户外活动

[第 1 轮 LLM 调用]
LLM 推理：需要获取两个城市的天气信息
LLM 响应：ToolCalls: [
  get_weather(city="北京"),
  get_weather(city="上海")
]

[工具执行]
工具 1 结果：北京：晴天，15°C
工具 2 结果：上海：小雨，18°C

[第 2 轮 LLM 调用]
LLM 推理：北京晴天，上海下雨，北京更适合户外活动
LLM 响应：Content: "北京今天天气晴朗，温度 15°C；上海有小雨，温度 18°C。
          北京更适合户外活动。"

[最终响应]
Assistant: 北京今天天气晴朗，温度 15°C；上海有小雨，温度 18°C。
          北京更适合户外活动。
```

## 小结

Agent 的 ReAct 循环是 Golem 的核心，它实现了：

1. **智能推理**：LLM 能够理解复杂任务，分解为多个步骤
2. **工具调用**：通过工具获取实时信息或执行操作
3. **循环优化**：多轮迭代，逐步逼近最终答案
4. **安全机制**：最大迭代次数、错误处理、优雅关闭

**关键要点**：
- 消息总线解耦组件
- 会话管理分离关注点
- 工具顺序优化性能
- 双通道结果（ForLLM + ForUser）

下一步，我们将深入学习**工具系统**，了解如何实现和注册自定义工具。

👉 [下一章：工具系统](./03-tool-system.md)
