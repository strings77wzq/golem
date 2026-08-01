# db-audit-trail Specification

## Purpose
定义数据库操作审计落盘、delegate 命令白名单、外部 MCP 子进程生命周期的行为要求。

## ADDED Requirements

### Requirement: 数据库审计 SHALL 落盘并携带 trace_id
CLI 运行时 SHALL 将 `AuditEntry`（denied 与 success）输出为结构化日志（component=audit），携带 `trace_id`（从执行 ctx 提取，缺失时允许为空）。

#### Scenario: 权限拒绝被记录
- **WHEN** sql_query 执行被权限门拒绝（`sql_query.go:82`）
- **THEN** 输出审计日志，status=denied，operation/table/sql 字段完整，trace_id 与当前请求一致

#### Scenario: 成功写操作被记录
- **WHEN** 带权限的 UPDATE/DELETE 成功执行（`sql_query.go:225`）
- **THEN** 输出审计日志，status=success，含 affected_rows 与 rollback_sql

### Requirement: delegate 命令白名单 SHALL 默认拒绝
`DelegateTool` SHALL 校验命令在构造时注入的 allowlist 中；`NewDelegateTool()` 的 allowlist 为空（拒绝一切）。

#### Scenario: 白名单外命令被拒绝
- **WHEN** `Execute` 收到不在 allowlist 的 command
- **THEN** 返回 `ToolResult{IsError: true}`，错误信息含 "command not in allowlist"，不启动任何进程

#### Scenario: 白名单内命令放行
- **WHEN** command 精确匹配 allowlist 成员
- **THEN** 正常执行 MCP 委托流程

### Requirement: 外部 MCP server 子进程 SHALL 随主进程清理
`LoadMCPTools` SHALL 在传入 ctx 取消时关闭已建立的 manager（终止外部 server 子进程）。

#### Scenario: ctx 取消触发清理
- **WHEN** LoadMCPTools 的 ctx 被 cancel
- **THEN** manager.Close 被调用，外部子进程终止，Close 幂等无 panic
