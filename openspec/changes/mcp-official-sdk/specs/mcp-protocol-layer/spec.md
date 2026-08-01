# mcp-protocol-layer Specification

## Purpose
定义 MCP 协议层替换为官方 go-sdk v1.7.0 后的行为要求（server 模式、client 模式、懒启动）。

## ADDED Requirements

### Requirement: MCP server 模式 SHALL 通过官方 go-sdk 暴露注册表工具
`mcp-server` 子命令 SHALL 使用 `modelcontextprotocol/go-sdk` v1.7.0 的 `Server`/`StdioTransport` 暴露当前全局工具注册表；每个工具的 `InputSchema` SHALL 由 `ToolParameter` 列表生成（type: object + properties + required）。

#### Scenario: tools/list 返回注册表工具
- **WHEN** 客户端经 stdio 发送 `tools/list`
- **THEN** 响应包含注册表全部工具，名称/描述/参数 schema 与注册表一致

### Requirement: MCP server 模式 SHALL 将 tool call 路由到 golem 工具执行
`tools/call` SHALL 将 `Arguments`（map[string]any）传递给对应工具的 `Execute`，`ToolResult.ForUser` SHALL 以 `TextContent` 返回，`IsError` SHALL 映射为 `CallToolResult.IsError`。

#### Scenario: 外部调用工具成功与失败
- **WHEN** 客户端调用已注册工具，参数合法
- **THEN** 返回 `TextContent`（ForUser 文本），IsError=false
- **WHEN** 工具返回 `IsError=true` 或 Execute 出错
- **THEN** 返回 `CallToolResult{IsError: true}` 且内容含错误描述，连接保持可用

### Requirement: MCP client 模式 SHALL 经 go-sdk 连接外部 server 并注册 mcp_ 前缀工具
`--mcp` 加载的外部 server SHALL 使用 `Client`/`CommandTransport` 连接，其工具 SHALL 以 `mcp_<server>_<tool>` 注册进全局注册表，调用 SHALL 经 `session.CallTool` 转发。

#### Scenario: 外部工具发现与调用
- **WHEN** `--mcp` 指向含 N 个工具的 server
- **THEN** 注册 N 个 `mcp_` 前缀工具，注册表按字母序（C4 不变式）
- **WHEN** 调用其中一个工具
- **THEN** 参数与结果经 `CallTool` 往返，错误软返回（`ToolResult{IsError: true}`，I3 不变式）

### Requirement: 协议层替换 SHALL 保持懒启动语义
未启用 `--mcp`/`mcp-server` 时 SHALL 不创建任何 client/server/transport 连接。

#### Scenario: flag 关闭时零初始化
- **WHEN** agent 模式不带 `--mcp` 启动
- **THEN** feature/mcp 无连接建立、无 goroutine 泄漏、启动路径零开销
