# mcp-runtime-hardening Specification — Delta

## Purpose
协议层替换为官方 go-sdk v1.7.0 后，原锚定自研 `transport.go` 生命周期/错误语义的 requirement 改写为对 go-sdk 适配层的等价验证。工具发现、调用路由、代理执行、优雅关闭等编排要求不变。

## ADDED Requirements

### Requirement: MCP 适配层生命周期 SHALL 通过 go-sdk 内存 transport 验证
系统 SHALL 提供自动化测试，使用 go-sdk `NewInMemoryTransports` 验证 server↔client 完整会话：启动、tools/list、tools/call、关闭。原自研 `transport.go` 的逐行 scanner 生命周期测试随文件删除。

#### Scenario: 内存 transport 端到端会话
- **WHEN** 测试创建 in-memory transport 对，server 注册注册表工具后 Connect
- **THEN** client 可 list 到工具、调用工具并收到结果，`Close` 后双方会话干净退出

#### Scenario: 会话关闭幂等
- **WHEN** 对已关闭的 session 再次调用 `Close` 或对已关闭连接发起 `CallTool`
- **THEN** 返回确定性错误而非挂起或 panic

### Requirement: MCP client 错误路径 SHALL 覆盖 go-sdk 会话语义
系统 SHALL 提供自动化测试覆盖：`CallTool` 协议错误、ctx 取消、transport 失败、代理工具软错误返回。

#### Scenario: 调用失败软返回
- **WHEN** 外部 server 调用返回错误响应或 ctx 被取消
- **THEN** 代理工具返回 `ToolResult{IsError: true}`（不终止 agent 循环），且连接状态可恢复或显式失败
