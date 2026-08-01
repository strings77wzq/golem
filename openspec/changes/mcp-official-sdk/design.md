## Context

`feature/mcp` 现状（2026-08-01 取证）：`client.go`/`server.go`/`transport.go` 为自研 JSON-RPC；`manager.go` 负责注册与 `mcp_<server>_<tool>` 命名（manager.go:189）；cmd 侧 3 个接线点（`mcp_adapter.go`/`mcp_server.go`/`adapter.go`）。外部契约：`--mcp` flag（JSON string/文件路径）、`mcp-server` 子命令、懒启动、metrics 包装。

官方 SDK 已验证（临时模块探针，2026-08-01）：`github.com/modelcontextprotocol/go-sdk@v1.7.0`（最新稳定版）；API 形态 `NewServer`+`AddTool[In,Out]`+`StdioTransport`、`NewClient`+`CommandTransport`+`ClientSession.CallTool`；`AddTool` 对 `In=any` 是官方特例（显式 `Tool.InputSchema` 时不做强制类型推断）；`Tool` 的 `InputSchema` 接受 JSON Schema（含 `json.RawMessage`）；依赖 9 直接 + 3 间接全纯 Go。

## Goals / Non-Goals

**Goals:**
- 协议层替换为 go-sdk v1.7.0（server + client 双模式）
- 外部契约零变化（flag/前缀/子命令/config/懒启动/metrics）
- 现有 `mcp-runtime-hardening` spec 的测试意图（生命周期、错误路径、编排）以新协议层等价实现
- 全量回归通过（42 包基线，`-race`）

**Non-Goals:**
- 不新增 HTTP transport 模式（STDIO 现状保持，AGENTS.md §13）
- 不改变 `mcp_` 前缀命名规则
- 不引入 OAuth/streamable HTTP 能力（SDK 有，本项目暂不用）

## Decisions

- **D1 版本**：`go-sdk v1.7.0`（2026-08-01 `go get @latest` 验证为稳定版；内置 5 个历史 spec 协商降级，兼容 2025-11-25 客户端）
- **D2 Server 模式映射**（`mcp-server` 子命令）：
  - `mcp.NewServer`；对注册表每个工具构造 `mcp.Tool{Name, Description, InputSchema}`，`InputSchema` 由 `ToolParameter` 列表生成 2020-12 draft JSON Schema（type: object + properties + required）
  - handler 用 `AddTool[any, any]`：`In=any` 特例 → SDK 不做类型推断，由显式 InputSchema 承载；handler 内 `req.Params.Arguments`（map[string]any）→ `tool.Execute(ctx, args)` → `ToolResult.ForUser` → `CallToolResult{Content: []Content{&TextContent{}}, IsError: ...}`，`ForLLM` 语义经注释保留
  - 若 jsonschema-go 校验拒绝 lenient schema（风险 R1），降级用 `Server.AddTool`（非泛型，自验证）
- **D3 Client 模式映射**（`--mcp`）：`mcp.NewClient` + `CommandTransport{Command: exec.Command(...)}` per server；`tools/list` → 包装为 golem `tools.Tool`（命名逻辑留在 manager 层不变）；调用经 `session.CallTool`；`mcp_` 前缀与 metrics 包装（cmd 层 `newMetricsTool`）不动
- **D4 生命周期**：manager 持有 `*ClientSession`；懒启动保持（flag 解析时才 `Connect(ctx)`）；`Close()` 收敛 session + transport
- **D5 测试策略**：自研 transport 生命周期测试 → 改用 go-sdk `NewInMemoryTransports`（进程外 fixture 需求归零）；client 错误路径覆盖 `CallTool` 错误、ctx 取消、Close 幂等
- **D6 文件布局**：删除 `transport.go`；`client.go`/`server.go` 重写为 SDK 适配层（保留文件名与包结构）；`manager.go`/`types.go` 保留

## Risks / Trade-offs

- **R1 schema 兼容**：golem 动态 `ToolParameter` → 2020-12 draft schema 的映射需测试（type 枚举映射、required 合并）；失败路径 → `Server.AddTool` 非泛型自验证（决策 D2 尾）
- **R2 体积**：`x/tools`（jsonrpc2）+ `segmentio/encoding` 增量估计 <2 MB，纯 Go 无 CGO（符合 C1）；构建后实测
- **R3 协议协商**：v1.7.0 server 暴露给旧客户端（Claude Code 若仍 2025-11-25）→ SDK 自动降级；E2E 验证
- **R4 行为差异**：自研 transport 的超时/错误文本与 SDK 不同 → `mcp-runtime-hardening` spec 中锚定自研行为的具体断言需改写（delta spec）
- Trade-off：SDK 错误信息不再由我们控制（可读性损失）换协议演进免费（收益）
