## Why

当前日志系统存在三个核心问题，阻碍团队协作和问题定位：

1. **无 trace ID 传播** — 生产环境日志无法关联跨组件请求（gateway → agent → tool → provider）
2. **无组件标识** — 日志输出只有 message + key-value，不知道来自哪个模块
3. **无字段模板** — 每次写日志都要手动拼 key-value，容易遗漏字段，格式不一致

这三个问题导致：
- 生产环境排查问题时，无法追踪一个请求的完整生命周期
- 多组件日志混合在一起，无法按模块过滤
- 不同开发者写的日志格式不统一，增加认知负担

## What Changes

### 核心变更

- **trace_id 迁移**：将 `NewTraceID()` 从 `core/agent/trace.go` 迁移到 `foundation/logger/trace.go`，使所有层都能生成和传播 trace_id
- **组件标识**：在 `foundation/logger/` 新增 `WithComponent(component string) Logger`，自动注入 `component` 字段
- **字段模板**：新增 `LogToolCall`、`LogHTTPRequest`、`LogError` 模板函数，统一日志格式

### 次要变更

- 更新 `core/agent/trace.go` 委托给 `foundation/logger.NewTraceID()`
- 更新 gateway/tui 的请求入口注入 trace_id
- 更新各组件构造时传入 component 参数

## Capabilities

### New Capabilities

- `trace-propagation`: 所有组件生成的 trace_id 自动注入 context，日志可按 trace_id 过滤关联
- `component-tagging`: 日志自动带 `component=xxx` 字段，可按组件过滤
- `field-templates`: 预定义日志字段组合，减少手动拼接，统一格式

### Modified Capabilities

- `logger`: 新增 `WithContext(ctx)`、`WithComponent(component)`、`WithFields(fields...)` 方法
- `agent`: `processMessage` 使用 `foundation/logger.NewTraceID()` 生成 trace_id
- `gateway`: HTTP 请求入口注入 trace_id 到 context

## Impact

- **Code**: `foundation/logger/` (3 个新文件), `core/agent/trace.go` (修改), `internal/gateway/` (修改), 多个组件 (传入 component)
- **APIs**: Logger 接口新增 3 个方法（向后兼容，新增方法不破坏现有调用）
- **Dependencies**: 无新增。纯 Go，`CGO_ENABLED=0` 保留
- **Systems**: 日志输出格式变化（新增 trace_id、component 字段），但不影响运行时行为
- **Risk**: 低。所有变更都是新增能力，不修改现有行为
