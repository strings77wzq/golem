## Context

Golem 当前日志系统基于 `log/slog`（stdlib），提供 4 个级别（Debug/Info/Warn/Error）和 JSON/Text 格式。日志通过依赖注入传递（构造函数传入），不使用全局单例。

现有问题：
- `core/agent/trace.go` 生成 trace_id 但只在 agent 包内使用，gateway/tui/mcp 无法生成和传播
- 日志输出无组件标识，无法按模块过滤
- 每次写日志都要手动拼 key-value，格式不一致

约束（AGENTS.md §4）：`CGO_ENABLED=0`，纯 Go，零外部依赖，层依赖 `foundation → core → feature → internal → cmd`。

## Goals / Non-Goals

**Goals:**
- G1: trace_id 在所有组件间传播，日志可按 trace_id 过滤关联
- G2: 日志自动带 `component=xxx` 字段，可按组件过滤
- G3: 预定义字段模板（LogToolCall, LogHTTPRequest, LogError），统一日志格式
- G4: 保持向后兼容，不破坏现有 58 个 logger 调用点

**Non-Goals:**
- 日志聚合平台集成（Loki/Prometheus）— 过度设计
- 日志 rotation — 需要外部工具（loki/promtail）
- 替换 slog 为 zerolog — 违反零外部依赖约束
- 全量迁移现有日志调用到模板 — 逐步迁移

## Decisions

### D1 — trace_id 迁移到 foundation/logger

**决策**：将 `NewTraceID()` 从 `core/agent/trace.go` 迁移到 `foundation/logger/trace.go`，`core/agent/trace.go` 委托给 foundation。

**理由**：
1. trace_id 生成是基础能力，应该在 foundation 层
2. 所有层（foundation/core/feature/internal/cmd）都能 import foundation/logger
3. gateway 的 HTTP 请求可以生成 trace_id，实现全链路追踪
4. `WithTraceID`/`TraceIDFromContext` 保留 in core/agent（它们是 context 操作，属于 core 层职责）

**替代方案**：重新生成（B）— 否决，两套 trace_id 生成器格式可能不一致。

### D2 — 组件名用常量 + 自由字符串

**决策**：定义核心组件常量（agent, gateway, mcp, rag, routing, health, security, tui, cli, telegram），同时允许自由字符串扩展。

**理由**：
1. 常量提供编译时安全、IDE 自动补全、易于 grep
2. 自由字符串允许 feature 层新模块扩展，不需要改 foundation 代码
3. 常量作为"推荐命名"，自由字符串作为"扩展性"

**实现**：
```go
// foundation/logger/component.go
const (
    ComponentAgent    = "agent"
    ComponentGateway  = "gateway"
    // ... 其他常量
)

func (l *slogLogger) WithComponent(component string) Logger {
    return &slogLogger{inner: l.inner.With("component", component)}
}
```

### D3 — 字段模板先实现 P0（LogToolCall + LogHTTPRequest）

**决策**：先实现 LogToolCall 和 LogHTTPRequest，再实现 LogError。

**理由**：
1. LogToolCall 使用频率最高（每次工具调用），诊断价值最大
2. LogHTTPRequest 是入口点，每个 HTTP 请求都需要
3. LogError 使用频率中等，但诊断价值极高（P1 优先级）

**实现**：
```go
// LogToolCall — 工具调用日志
func LogToolCall(log Logger, toolName string, duration time.Duration, err error) {
    args := []any{"tool", toolName, "duration_ms", duration.Milliseconds()}
    if err != nil {
        log.Error("tool execution failed", append(args, "error", err)...)
    } else {
        log.Info("tool executed", args...)
    }
}

// LogHTTPRequest — HTTP 请求日志
func LogHTTPRequest(log Logger, method, path string, status int, duration time.Duration) {
    level := slog.LevelInfo
    if status >= 400 { level = slog.LevelWarn }
    if status >= 500 { level = slog.LevelError }
    log.Log(level, "http request",
        "method", method, "path", path,
        "status", status, "duration_ms", duration.Milliseconds(),
    )
}
```

### D4 — Context propagation 通过 slog.Handler 实现

**决策**：在 `slog.HandlerOptions` 中自定义 `ReplaceAttr`，从 context 提取 trace_id 并注入日志。

**理由**：
1. slog 原生支持 context propagation（`slog.Handler.Handle(ctx, ...)`）
2. 不需要修改 Logger 接口，只需要自定义 Handler
3. 所有通过 `Logger.WithContext(ctx)` 的调用自动注入 trace_id

**实现**：
```go
// foundation/logger/context_handler.go
type contextHandler struct {
    inner slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
    if traceID := TraceIDFromContext(ctx); traceID != "" {
        r.Add("trace_id", traceID)
    }
    return h.inner.Handle(ctx, r)
}
```

## Risks / Trade-offs

- [trace_id 迁移] `core/agent/trace.go` 的 `NewTraceID()` 被外部调用？ → 验证：grep 确认只有 `loop.go:331` 调用，无外部调用者。
- [Logger 接口变更] 新增方法是否破坏现有调用？ → 不会，Go 接口是鸭子类型，新增方法不破坏现有实现。
- [性能影响] context propagation 增加开销？ → 极小，`TraceIDFromContext` 是 O(1) map 查找，可忽略。
- [组件常量维护] 新增组件需要改常量定义？ → 允许自由字符串扩展，常量只是推荐命名。
