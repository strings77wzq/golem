## Why

三笔信任债：① CLAUDE.md/README 声称"每次允许和拒绝的操作都被审计"，但 CLI 接线 `BuildDBTools(dbFlag, nil, nil)`（`cmd/golem/main.go:166`）传 **nil auditFn**——审计只存在于工具内部调用点，从未落盘；② `DelegateTool` 无命令白名单（安全审阅 M2："接入即高危"），命令完全由 LLM 控制；③ `LoadMCPTools` 丢弃 manager 引用，外部 MCP server 子进程在进程退出时成为孤儿。

## What Changes

- **审计接线**：`security.AuditEntry` 增加 `TraceID` 字段（兼容新增，非破坏）；sql_query 三处审计调用点（denied `sql_query.go:82/110` + success `:225`）从 ctx 提取 trace_id 填入；`main.go:166` 的 auditFn 从 nil 改为 foundation/logger 结构化输出（component=audit）
- **delegate 白名单**：`NewDelegateTool()` 默认空 allowlist（fail-closed，拒绝一切）；新增 `NewDelegateToolWithAllowlist(commands ...string)`；`Execute` 校验 command ∈ allowlist
- **manager 生命周期**：`LoadMCPTools` 注册 ctx-cancel → `manager.Close()`（子进程随主进程生命周期清理）

## Capabilities

### New Capabilities
- `db-audit-trail`: 定义数据库操作审计的行为要求（denied/success 都记录、携带 trace_id、可聚合）

### Modified Capabilities
（无——现有 specs 无 audit 相关）

## Impact

- `core/security/gates.go`（AuditEntry +TraceID）、`core/tools/database/sql_query.go`（trace_id 提取）、`cmd/golem/main.go`（auditFn 接线）、`cmd/golem/mcp_adapter.go`（生命周期）、`feature/mcp/delegate.go`（白名单）
- 测试：audit 接线测试（trace_id 贯穿断言）、白名单测试、生命周期测试
- 日志：新增 component=audit 输出（SQL 原文进日志——文档标注敏感信息）
