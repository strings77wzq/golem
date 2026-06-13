## Context

Golem 项目当前处于 v0.5.1 版本，已完成核心 Agent 框架、工具系统、多 LLM 提供商适配、MCP/RAG/技能系统集成。项目采用 Hexagonal Architecture（端口与适配器模式），遵循清晰的分层依赖规则：

```
cmd/         → imports ALL layers (composition root)
internal/    → imports core/ + foundation/ only
core/        → imports foundation/ only
feature/     → imports core/ + foundation/ only
foundation/  → imports stdlib only
```

当前存在的问题：
1. **telegram 包构建失败** — `webhook.go` 与 `types.go` 存在类型重复声明，`webhook.go` 引用了不存在的 `doRequest` 方法
2. **测试覆盖率不均** — `cmd/golem` (23.6%)、`foundation/term` (33.3%) 覆盖率不足
3. **memory 模块未完成** — 仅有占位实现，缺少核心持久化和衰减逻辑
4. **gateway 安全性待增强** — 缺少审计日志和请求验证

## Goals / Non-Goals

**Goals:**
1. 修复 telegram 包构建问题，使其成为可用通道
2. 将测试覆盖率从 79.2% 提升至 85%+
3. 完成 memory 模块的核心功能实现
4. 增强 gateway 安全性，添加审计日志
5. 添加 provider 健康检查和会话导出/导入功能

**Non-Goals:**
1. 不引入新的外部依赖（保持 `CGO_ENABLED=0`）
2. 不修改 `LLMProvider` 或 `StreamingProvider` 接口签名
3. 不重构现有架构或包结构
4. 不实现 WebSocket 支持（继续使用 SSE）

## Decisions

### D1: Telegram 包修复策略

**决策**: 合并 `webhook.go` 中的重复类型定义到 `types.go`，删除 `webhook.go` 中的重复定义，并在 `client.go` 中添加 `doRequest` 方法。

**理由**: 
- `types.go` 已有完整的类型定义，`webhook.go` 的重复定义是开发过程中的冗余
- `doRequest` 是通用方法，应放在 `client.go` 中供 `SetWebhook` 和 `DeleteWebhook` 复用

**替代方案**: 删除 `webhook.go`，仅保留长轮询模式。但 webhook 模式对生产部署更友好，值得保留。

### D2: Memory 模块实现方案

**决策**: 实现 `MemoryStore` 接口，使用 SQLite 作为持久化后端，支持重要性评分和指数衰减。

**数据模型**:
```go
type Memory struct {
    ID          string    // UUID
    Content     string    // 记忆内容
    Importance  float64   // 重要性评分 (0-1)
    AccessCount int       // 访问次数
    CreatedAt   time.Time
    LastAccess  time.Time
    Embedding   []float64 // 可选：向量嵌入
}
```

**衰减公式**: `Score = Importance * e^(-λ * Age) * log(AccessCount + 1)`
- λ = 0.1 (衰减率，可配置)
- Age = time.Since(CreatedAt).Hours() / 24 (天数)

**理由**: 
- SQLite 满足纯 Go 约束，无需额外依赖
- 指数衰减符合人类记忆遗忘曲线
- 访问次数加权确保重要记忆不会被遗忘

### D3: Provider 健康检查设计

**决策**: 在 `core/providers/` 添加 `HealthChecker` 接口，各 Provider 实现轻量级健康检查。

**接口定义**:
```go
type HealthChecker interface {
    HealthCheck(ctx context.Context) (*HealthStatus, error)
}

type HealthStatus struct {
    Provider    string
    Status      string  // "healthy", "degraded", "unhealthy"
    Latency     time.Duration
    Error       string  // 如果 unhealthy
    CheckedAt   time.Time
}
```

**实现方式**: 发送一个最小化的 chat 请求（如 "ping"），测量响应时间和成功率。

**理由**: 
- 统一接口便于扩展新 Provider
- 轻量级检查不会消耗大量 token
- 延迟信息对路由决策有价值

### D4: 会话导出/导入格式

**决策**: 使用 JSON 格式，包含完整会话元数据和消息历史。

**格式**:
```json
{
  "version": "1.0",
  "exported_at": "2026-03-23T10:00:00Z",
  "session": {
    "id": "uuid",
    "created_at": "2026-03-20T10:00:00Z",
    "messages": [
      {"role": "user", "content": "..."},
      {"role": "assistant", "content": "..."}
    ]
  }
}
```

**理由**: 
- JSON 格式通用性好，易于调试和迁移
- 版本字段支持未来格式演进
- 完整元数据便于会话恢复和审计

### D5: Gateway 审计日志

**决策**: 添加 `AuditMiddleware`，记录所有请求到 SQLite 审计表。

**记录字段**:
```go
type AuditLog struct {
    ID           string
    Timestamp    time.Time
    Method       string
    Path         string
    StatusCode   int
    Latency      time.Duration
    ClientIP     string
    UserID       string  // 如果有认证
    RequestSize  int64
    ResponseSize int64
}
```

**理由**: 
- SQLite 满足纯 Go 约束
- 结构化日志便于查询和分析
- 延迟和大小信息对性能分析有价值

## Risks / Trade-offs

### R1: Memory 模块性能风险
**风险**: 大量记忆条目可能导致检索延迟
**缓解**: 
- 实现重要性阈值过滤
- 添加 LRU 缓存层
- 支持批量检索优化

### R2: Provider 健康检查成本
**风险**: 频繁健康检查消耗 API 配额
**缓解**: 
- 默认检查间隔 5 分钟
- 支持手动触发检查
- 失败时指数退避

### R3: 审计日志存储增长
**风险**: 高流量场景下审计日志快速增长
**缓解**: 
- 支持日志轮转配置
- 提供清理命令
- 默认保留 7 天

### R4: Telegram Webhook 安全性
**风险**: Webhook 端点可能被滥用
**缓解**: 
- 强制验证 secret token
- 添加 IP 白名单选项
- 速率限制

## Migration Plan

### Phase 1: 修复构建问题 (Day 1)
1. 修复 telegram 包类型重复
2. 添加 `doRequest` 方法
3. 运行测试验证

### Phase 2: 提升测试覆盖率 (Day 2-3)
1. 为 `cmd/golem` 添加集成测试
2. 为 `foundation/term` 添加单元测试
3. 验证覆盖率达标

### Phase 3: 实现新功能 (Day 4-7)
1. 实现 memory 模块
2. 实现 provider 健康检查
3. 实现会话导出/导入
4. 实现 gateway 审计日志

### Phase 4: 集成测试 (Day 8)
1. 端到端功能测试
2. 性能基准测试
3. 文档更新

### 回滚策略
- 所有新功能通过 feature flag 控制
- 数据库变更使用迁移脚本
- 保留旧 API 兼容性

## Open Questions

1. **Memory 嵌入维度**: 是否需要支持自定义嵌入模型？当前建议使用 OpenAI text-embedding-3-small 作为默认。
2. **健康检查端点**: 是否需要在 Gateway 暴露 `/health/providers` 公开端点，还是仅内部使用？
3. **审计日志导出**: 是否需要支持审计日志导出功能？建议后续版本实现。
