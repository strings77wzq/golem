## 1. 依赖与基线

- [ ] 1.1 `go get github.com/modelcontextprotocol/go-sdk@v1.7.0`；确认 `go.mod` 依赖图全纯 Go（无 CGO）
- [ ] 1.2 基线：`go test ./... -race` 全绿（替换前快照）；记录 `feature/mcp` 现有 5 测试清单

## 2. Server 模式适配（TDD：先测试后实现）

- [ ] 2.1 写测试（红）：`mcp.Server` + `NewInMemoryTransports` 端到端——注册表工具可 list、可调用、IsError 映射（对应 specs/mcp-protocol-layer Req 1-2）
- [ ] 2.2 实现：重写 `server.go` 为 SDK 适配层——`mcp.NewServer` + 遍历注册表 `AddTool[any,any]`；`ToolParameter` → 2020-12 JSON Schema 映射函数（含测试）
- [ ] 2.3 验证（绿）：`go test ./feature/mcp/... -race` 通过
- [ ] 2.4 `mcp-server` 子命令（cmd/mcp_server.go）切换为适配层入口；`go build ./...` 通过

## 3. Client 模式适配（TDD）

- [ ] 3.1 写测试（红）：`mcp.NewClient` + `CommandTransport`（用 in-memory 或 fixture 子进程）——tools/list 发现、`mcp_` 前缀注册、CallTool 往返、错误软返回（对应 Req 3）
- [ ] 3.2 实现：重写 `client.go` 连接层；`manager.go` 命名/注册逻辑保留；懒启动保持（flag 解析时才 Connect）
- [ ] 3.3 验证（绿）：`go test ./feature/mcp/... -race` 通过

## 4. 旧 transport 清理与测试迁移

- [ ] 4.1 删除 `transport.go`；原 transport 生命周期测试迁移为 in-memory transport 会话测试（对应 mcp-runtime-hardening delta）
- [ ] 4.2 全 feature/mcp 测试重构完成：`go test ./feature/mcp/... -cover -race` 覆盖 ≥80%
- [ ] 4.3 `go vet ./...` + `golangci-lint run` 零新增告警

## 5. 接线与全量回归

- [ ] 5.1 cmd 侧（mcp_adapter.go/adapter.go）签名微调（如 LoadMCPTools 返回类型变化）；CLI 契约不变
- [ ] 5.2 全量：`go test ./... -race` + `go test -cover ./...`（42 包，无回归）
- [ ] 5.3 体积实测：`go build -o /tmp/golem-p1 && ls -lh /tmp/golem-p1` 与替换前对比（R2 验证，预期增量 <2MB）
- [ ] 5.4 输出 `task-output.md`（evidence pasting：test/build/体积实际输出）
