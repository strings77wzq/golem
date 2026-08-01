## Why

`feature/mcp` 的自研 JSON-RPC 协议层（`client.go`/`server.go`/`transport.go`）是项目最大的维护税：MCP spec 在 2025-11 → 2026-07 间发布了 5 个版本（含 2026-07-28 移除 initialize handshake 的重大改写），自研协议层被迫持续跟进。官方 SDK `github.com/modelcontextprotocol/go-sdk` v1.7.0（2026-08-01 已 `go get` 验证为最新稳定版，Google 联合维护）覆盖 golem 的全部用法（server 模式 + client 模式），内置 5 个历史 spec 的版本协商，且依赖全纯 Go（segmentio/encoding、x/tools、oauth2 等，无 CGO）。

## What Changes

- `feature/mcp/` 协议层替换为官方 SDK：
  - server 模式（`mcp-server` 子命令）：`mcp.NewServer` + 遍历注册表 `AddTool`（`In=any` 特例适配动态工具）+ `StdioTransport`
  - client 模式（`--mcp` flag）：`mcp.NewClient` + `CommandTransport`（每 server 一个）+ `session.CallTool`
- 保留：`manager.go` 注册/命名逻辑（`mcp_<server>_<tool>` 前缀）、cmd 接线契约（`--mcp` flag、`mcp-server` 子命令、JSON config string/文件路径解析）、懒启动语义、metrics 包装
- 删除：自研 `transport.go`（逐行 scanner + 自管理生命周期）
- go.mod：新增 `github.com/modelcontextprotocol/go-sdk v1.7.0`（+9 直接/3 间接依赖，全部纯 Go，符合 `CGO_ENABLED=0`）
- **BREAKING**（内部）：`feature/mcp` 的 `transport.go` 相关导出符号删除（无外部消费者，仅 `mcp-runtime-hardening` spec 的测试要求引用）

## Capabilities

### New Capabilities
- `mcp-protocol-layer`: 定义 MCP 协议层的行为要求（server/client/transport 语义），作为替换后的验收基准

### Modified Capabilities
- `mcp-runtime-hardening`: 其 requirement 锚定自研 `transport.go` 的生命周期/错误语义——替换后改为验证 go-sdk 适配层的等价行为（工具发现、调用路由、代理执行、优雅关闭保持不变）

## Impact

- go.mod/go.sum：新增 go-sdk v1.7.0 依赖图
- `feature/mcp/`：文件结构变化（协议层文件替换，manager/命名保留）
- `cmd/golem/`：3 个接线点（`mcp_adapter.go`/`mcp_server.go`/`adapter.go`）仅可能微调签名，外部 CLI 契约不变
- `openspec/specs/mcp-runtime-hardening/spec.md`：delta 更新
- 测试：`feature/mcp` 5 个现有测试按新协议层重写/补充；全量回归（42 包 82.5% 覆盖基线）
- 体积：新增依赖全部纯 Go；`segmentio/encoding` 为高性能 JSON（与自研 json 手写解析相比净收益）；无 CGO
