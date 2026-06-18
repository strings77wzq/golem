# Active Rollover Tasks

> Carried-forward open work from the two `partially-landed` proposals
> archived in Slice E2. Task numbering is preserved from the source
> proposals so cross-references remain valid; section headings prefix
> the source for clarity.

## A. From `fix-ci-vet-provider-health-types`

<!-- inherited from fix-ci-vet-provider-health-types -->
- [ ] A.3.2 Commit and push the fix branch to trigger GitHub Actions
      (the next push of `feat/e2e-harness-and-spec-cleanup` to remote
      will exercise the same provider-health surface — this task is
      satisfied by capturing that run URL).
- [ ] A.3.3 Confirm the previously failing CI path passes and capture
      the run URL/result in the rollover archive notes.

## B. From `project-review-and-roadmap` — section 5: Memory module completion

<!-- inherited from project-review-and-roadmap -->
- [ ] B.5.1 定义 Memory 结构体 (ID, Content, Importance, AccessCount, timestamps)
- [ ] B.5.2 定义 MemoryStore 接口
- [ ] B.5.3 实现 SQLite 持久化后端
- [ ] B.5.4 实现重要性评分存储
- [ ] B.5.5 实现指数衰减计算 (Score = Importance * e^(-λ * Age) * log(AccessCount + 1))
- [ ] B.5.6 实现 Top-K 相关性检索
- [ ] B.5.7 实现内容关键词搜索
- [ ] B.5.8 实现低相关性记忆自动清理
- [ ] B.5.9 为 memory 模块添加单元测试 (目标覆盖率 80%+)

## C. From `project-review-and-roadmap` — section 6: Gateway audit log

<!-- inherited from project-review-and-roadmap -->
- [ ] C.6.1 定义 AuditLog 结构体
- [ ] C.6.2 创建 SQLite 审计日志表
- [ ] C.6.3 实现 AuditMiddleware 中间件
- [ ] C.6.4 实现审计日志查询功能
- [ ] C.6.5 实现审计日志自动清理 (默认保留 7 天)
- [ ] C.6.6 在 Gateway 添加 GET /admin/audit-logs 端点
- [ ] C.6.7 添加审计日志访问控制 (需要管理员权限)
- [ ] C.6.8 为审计日志功能添加单元测试

## D. From `project-review-and-roadmap` — section 7: Agent integration

<!-- inherited from project-review-and-roadmap -->
- [ ] D.7.1 Agent 集成 provider 健康检查 (跳过不健康的 provider)
- [ ] D.7.2 Agent 集成 memory 模块 (存储重要交互、检索相关记忆)
- [ ] D.7.3 Agent 支持会话导出功能
- [ ] D.7.4 为 Agent 集成功能添加单元测试

## E. From `project-review-and-roadmap` — section 8: Docs and release wrap-up

<!-- inherited from project-review-and-roadmap -->
- [ ] E.8.1 更新 README.md 添加新功能说明
- [ ] E.8.2 更新 CHANGELOG.md 记录变更
- [ ] E.8.3 更新 AGENTS.md 添加新模块规则
- [ ] E.8.4 运行完整测试套件验证所有功能
- [ ] E.8.5 运行 `golangci-lint run` 验证代码质量
- [ ] E.8.6 更新版本号 (v0.5.1 → v0.6.0)

## Notes

- Tasks B.5.* and C.6.* may benefit from being split into a focused
  successor proposal (working title `memory-and-audit-completion`)
  before any code lands, since both areas warrant their own design
  pass. The rollover preserves the original task list verbatim — any
  reorganisation into a new proposal is its own change.
- Task D.7.1 overlaps with `wire-security-gates` task 5.1 (provider
  health runtime wiring); coordinate to avoid duplicate work.
- Task D.7.2 depends on B.5.* memory completion.
