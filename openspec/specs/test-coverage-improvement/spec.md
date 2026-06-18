# Test Coverage Improvement — Spec

## [S1] Problem

当前测试覆盖率不均衡：

| 包 | 当前覆盖率 | 目标 | 差距 |
|-----|-----------|------|------|
| core/database | 32.3% | 80% | -47.7% |
| core/providers/ollama | 34.0% | 80% | -46.0% |
| cmd/golem | 40.0% | 70% | -30.0% |
| core/agent | 49.6% | 80% | -30.4% |

## [S2] 方案

### Phase 1: core/database (32.3% → 80%)

**需要测试的文件：**
- `sqlite.go` — SQLite 驱动核心
- `postgres.go` — PostgreSQL 驱动
- `mysql.go` - MySQL 驱动
- `redis.go` — Redis 驱动
- `registry.go` — 驱动注册表

**测试策略：**
- 使用内存数据库（SQLite in-memory）
- Mock 外部连接（PG/MySQL/Redis）
- 测试 CRUD 操作、错误处理、连接管理

### Phase 2: core/providers/ollama (34.0% → 80%)

**需要测试的文件：**
- `ollama.go` — Ollama 适配器

**测试策略：**
- Mock HTTP 服务器模拟 Ollama API
- 测试 Chat、ChatStream、错误处理
- 测试连接超时、重试逻辑

### Phase 3: cmd/golem (40.0% → 70%)

**需要测试的文件：**
- 各个 CLI 命令的 RunE 函数

**测试策略：**
- 使用 cobra 的 Execute() 测试完整命令
- Mock 外部依赖（数据库、LLM）
- 测试命令参数解析、错误处理

### Phase 4: core/agent (49.6% → 80%)

**需要测试的文件：**
- `loop.go` — ReAct 循环的边界情况
- `compactor.go` — 压缩逻辑

**测试策略：**
- 测试空会话、超长会话
- 测试并发访问
- 测试 hook 执行顺序

## [S3] 验证

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep "total:"
```
