## 1. 审计落盘（TDD）

- [ ] 1.1 写测试（红）：`AuditEntry.TraceID` 字段存在；sql_query denied/success 审计点携带 trace_id（mock ctx 注入 trace 断言）
- [ ] 1.2 实现：`gates.go` AuditEntry +TraceID；`sql_query.go` 三处审计点从 ctx 提取 trace_id
- [ ] 1.3 写测试（红）：main.go 接线 auditFn → 结构化日志输出（logger 断言）
- [ ] 1.4 实现：`cmd/golem/main.go:166` auditFn 接线（component=audit）
- [ ] 1.5 验证（绿）：`go test ./core/security/... ./core/tools/database/... ./cmd/golem/... -race`

## 2. delegate 白名单（TDD）

- [ ] 2.1 写测试（红）：空 allowlist 拒绝一切；allowlist 命中放行；未命中拒绝（不启动进程）
- [ ] 2.2 实现：`delegate.go` 新增 allowlist 字段 + `NewDelegateToolWithAllowlist` + Execute 校验
- [ ] 2.3 验证（绿）：`go test ./feature/mcp/... -race`

## 3. manager 生命周期（TDD）

- [ ] 3.1 写测试（红）：ctx cancel → manager.Close 被调用（helper 子进程终止）
- [ ] 3.2 实现：`mcp_adapter.go` LoadMCPTools ctx 清理注册
- [ ] 3.3 验证（绿）：`go test ./cmd/golem/... -race`

## 4. 收尾

- [ ] 4.1 全量：`go test ./... -race` + `-cover` ≥80%；vet；golangci-lint
- [ ] 4.2 输出 task-output.md（evidence pasting）
