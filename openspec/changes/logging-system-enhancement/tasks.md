## 1. trace_id 迁移到 foundation/logger

- [x] 1.1 创建 `foundation/logger/trace.go`：实现 `NewTraceID()` 函数，生成 `trace-{16hex}` 格式
- [x] 1.2 创建 `foundation/logger/trace_test.go`：测试 trace_id 格式正确、唯一性
- [x] 1.3 修改 `core/agent/trace.go`：`NewTraceID()` 委托给 `foundation/logger.NewTraceID()`
- [x] 1.4 运行 `go test -race ./foundation/logger/... ./core/agent/...`，确认测试通过

## 2. Context propagation（trace_id 注入日志）

- [x] 2.1 创建 `foundation/logger/context_handler.go`：自定义 slog.Handler，从 context 提取 trace_id 注入日志
- [x] 2.2 修改 `foundation/logger/logger.go`：`New()` 函数包装 handler 为 contextHandler
- [x] 2.3 修改 `foundation/logger/trace.go`：新增 `TraceIDFromContext(ctx)` 函数（从 core/agent 迁移）
- [x] 2.4 创建 `foundation/logger/context_handler_test.go`：测试 trace_id 注入日志
- [x] 2.5 运行 `go test -race ./foundation/logger/...`，确认测试通过

## 3. 组件标识（WithComponent）

- [x] 3.1 创建 `foundation/logger/component.go`：定义组件常量 + `WithComponent(component string) Logger` 方法
- [x] 3.2 修改 `foundation/logger/logger.go`：`slogLogger` 实现 `WithComponent` 方法
- [x] 3.3 创建 `foundation/logger/component_test.go`：测试组件标识注入日志
- [x] 3.4 运行 `go test -race ./foundation/logger/...`，确认测试通过

## 4. 字段模板（LogToolCall + LogHTTPRequest）

- [x] 4.1 创建 `foundation/logger/templates.go`：实现 `LogToolCall` 和 `LogHTTPRequest` 函数
- [x] 4.2 创建 `foundation/logger/templates_test.go`：测试模板函数输出格式
- [x] 4.3 运行 `go test -race ./foundation/logger/...`，确认测试通过

## 5. 集成到 agent 和 gateway

- [x] 5.1 修改 `core/agent/loop.go`：`processMessage` 使用 `logger.NewTraceID()` 生成 trace_id
- [x] 5.2 修改 `internal/gateway/routes.go`：`handleChatStream` 注入 trace_id 到 context
- [x] 5.3 修改 `internal/gateway/openai_compat.go`：`handleOpenAICompatChat` 注入 trace_id 到 context
- [x] 5.4 运行 `go test -race ./core/agent/... ./internal/gateway/...`，确认测试通过

## 6. 组件标识集成

- [x] 6.1 修改 `core/agent/agent.go`：`New` 传入 `logger.ComponentAgent`
- [x] 6.2 修改 `internal/gateway/server.go`：`NewServerWithSecurity` 传入 `logger.ComponentGateway`
- [ ] 6.3 修改 `feature/mcp/server.go`：`NewServer` 传入 `logger.ComponentMCP`（MCP Server 无 logger 参数，跳过）
- [x] 6.4 修改 `feature/health/manager.go`：`New` 传入 `logger.ComponentHealth`
- [x] 6.5 运行 `go test -race ./...`，确认所有测试通过

## 7. 文档同步

- [x] 7.1 更新 `README.md`：新增日志配置说明（级别、格式、trace_id）
- [x] 7.2 更新 `AGENTS.md`：新增日志使用规范（何时用模板、何时用自由格式）
- [x] 7.3 新增 `docs/logging.md`：日志架构文档（trace_id 传播、组件标识、字段模板）

## 8. 验证与门禁

- [x] 8.1 `go build ./...` 和 `go vet ./...` 无错误
- [x] 8.2 `go test -race ./...` 全部通过
- [x] 8.3 `golangci-lint run` 无新增警告
- [x] 8.4 确认所有现有日志调用点仍正常工作（grep 确认无遗漏）
