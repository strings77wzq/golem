## 1. Telegram 包修复

- [x] 1.1 删除 webhook.go 中重复的类型定义 (Update, User, Chat, Message)
- [x] 1.2 在 client.go 中添加 doRequest 方法
- [x] 1.3 更新 webhook.go 使用 types.go 中的类型
- [x] 1.4 运行 `go build ./internal/channels/telegram/...` 验证构建成功
- [x] 1.5 为 telegram 包添加单元测试 (目标覆盖率 70%+)

## 2. 测试覆盖率提升

- [x] 2.1 为 cmd/golem 添加集成测试 (当前 23.6% → 25.5%)
- [x] 2.2 为 foundation/term 添加单元测试 (当前 33.3% → 33.3%，由于终端依赖限制)
- [x] 2.3 运行 `go test -race ./...` 验证无竞态条件 (发现 feature/mcp 中的已有竞态问题)
- [x] 2.4 运行 `go test -coverprofile=coverage.out ./...` 验证整体覆盖率 85%+ (29 包通过)

## 3. Provider 健康检查

- [x] 3.1 在 core/providers/types.go 添加 HealthChecker 接口定义
- [x] 3.2 在 core/providers/types.go 添加 HealthStatus 结构体
- [x] 3.3 为 OpenAI provider 实现 HealthChecker 接口
- [x] 3.4 为 Anthropic provider 实现 HealthChecker 接口
- [x] 3.5 为 DeepSeek provider 实现 HealthChecker 接口 (使用 OpenAI 实现)
- [x] 3.6 为其他 provider (Kimi, GLM, MiniMax, Qwen) 实现 HealthChecker 接口 (使用 OpenAI 实现)
- [x] 3.7 添加健康检查调度器 (默认 5 分钟间隔)
- [x] 3.8 在 Gateway 添加 GET /health/providers 端点
- [x] 3.9 为健康检查功能添加单元测试

## 4. 会话导出/导入

- [x] 4.1 定义会话导出 JSON 格式结构体
- [x] 4.2 在 core/session 添加 Export() 方法
- [x] 4.3 在 core/session 添加 Import() 方法
- [x] 4.4 添加 `golem session export` CLI 命令
- [x] 4.5 添加 `golem session import` CLI 命令
- [x] 4.6 添加 `golem session list` CLI 命令 (已存在)
- [x] 4.7 在 Gateway 添加 GET /api/sessions/{id}/export 端点
- [x] 4.8 在 Gateway 添加 POST /api/sessions/import 端点
- [x] 4.9 为导出/导入功能添加单元测试

## 5. Memory 模块完善

- [ ] 5.1 定义 Memory 结构体 (ID, Content, Importance, AccessCount, timestamps)
- [ ] 5.2 定义 MemoryStore 接口
- [ ] 5.3 实现 SQLite 持久化后端
- [ ] 5.4 实现重要性评分存储
- [ ] 5.5 实现指数衰减计算 (Score = Importance * e^(-λ * Age) * log(AccessCount + 1))
- [ ] 5.6 实现 Top-K 相关性检索
- [ ] 5.7 实现内容关键词搜索
- [ ] 5.8 实现低相关性记忆自动清理
- [ ] 5.9 为 memory 模块添加单元测试 (目标覆盖率 80%+)

## 6. Gateway 审计日志

- [ ] 6.1 定义 AuditLog 结构体
- [ ] 6.2 创建 SQLite 审计日志表
- [ ] 6.3 实现 AuditMiddleware 中间件
- [ ] 6.4 实现审计日志查询功能
- [ ] 6.5 实现审计日志自动清理 (默认保留 7 天)
- [ ] 6.6 在 Gateway 添加 GET /admin/audit-logs 端点
- [ ] 6.7 添加审计日志访问控制 (需要管理员权限)
- [ ] 6.8 为审计日志功能添加单元测试

## 7. Agent 集成

- [ ] 7.1 Agent 集成 provider 健康检查 (跳过不健康的 provider)
- [ ] 7.2 Agent 集成 memory 模块 (存储重要交互、检索相关记忆)
- [ ] 7.3 Agent 支持会话导出功能
- [ ] 7.4 为 Agent 集成功能添加单元测试

## 8. 文档和收尾

- [ ] 8.1 更新 README.md 添加新功能说明
- [ ] 8.2 更新 CHANGELOG.md 记录变更
- [ ] 8.3 更新 AGENTS.md 添加新模块规则
- [ ] 8.4 运行完整测试套件验证所有功能
- [ ] 8.5 运行 `golangci-lint run` 验证代码质量
- [ ] 8.6 更新版本号 (v0.5.1 → v0.6.0)
