## Context

`BuildDBTools(dbPath, auditFn, secHandler)`（internal/wiring/tools.go:35）已支持 auditFn 注入，`SetAuditFunc`（sql_query.go:48）已实现——只差 `cmd/golem/main.go:166` 传 nil。`AuditEntry`（gates.go:200-209）8 字段无 trace_id。DelegateTool 无白名单。LoadMCPTools（mcp_adapter.go:16）返回的 manager 被丢弃。

## Goals / Non-Goals

**Goals:**
- 审计落盘：denied+success 的 AuditEntry 以结构化日志输出，携带 trace_id
- delegate fail-closed 白名单
- 外部 MCP 子进程随主进程清理

**Non-Goals:**
- 不引入审计存储/查询（日志即存储，聚合交给日志系统）
- 不改变 delegate 的 Tools.Tool 接口
- 不改变 LoadMCPTools 签名（生命周期内部注册）

## Decisions

- **D1 TraceID 来源**：`AuditEntry.TraceID` 新增字段；sql_query 三处审计点从 `ctx` 经 `logger.TraceIDFromContext` 提取（审计发生在 `Execute(ctx)` 内，ctx 可用）
- **D2 审计输出**：`cmd/golem/main.go` 构造 auditFn → `foundation/logger` 的 `LogError`/Info 级别结构化输出（`msg="db audit"`, component=audit, 字段=AuditEntry json tags）；日志级别 INFO；SQL 原文进日志（文档标注）
- **D3 delegate 白名单**：`NewDelegateTool()` = 空 allowlist（fail-closed）；`NewDelegateToolWithAllowlist(commands ...string)`；Execute 校验 `command` 精确匹配 allowlist 成员，未命中返回 `IsError`（错误信息含"command not in allowlist"）；不改变接口形状
- **D4 manager 生命周期**：`LoadMCPTools` 内注册 `go func() { <-ctx.Done(); _ = manager.Close() }()`——前提（A4）：调用方 ctx 有 cancel 路径（spec 阶段验证 main.go 的 agent ctx）

## Risks / Trade-offs

- **R1 日志敏感**：SQL 原文与 rollback SQL 进日志——文档标注为敏感字段；日志级别可配置
- **R2 trace_id 缺失**：CLI 非 gateway 路径无注入 trace 时为空串——AuditEntry.TraceID 允许空（兼容旧路径）
- **R3 ctx 生命周期**：若 cmd 接线 ctx 无 cancel（main 常驻），清理不触发——验证后若如此，改为显式 defer 注册到 signal 处理
