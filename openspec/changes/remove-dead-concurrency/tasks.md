## 1. 删除 foundation/concurrency

- [ ] 1.1 `git rm` 删除 `foundation/concurrency/{semaphore.go, pool.go, rate_limiter.go}` 及 3 个 `_test.go`（以 `ls foundation/concurrency/` 实际清单为准）
- [ ] 1.2 确认无残留引用：`grep -rn 'foundation/concurrency' --include='*.go' .`（期望仅此 change 的 proposal/design 命中）
- [ ] 1.3 验证：`go build ./...` 通过；`go test ./... -race` 全量通过（42 包基线无回归）

## 2. 收尾

- [ ] 2.1 确认 `foundation/` 目录剩余包均为活跃代码（bm25/logger/metrics/store/term）
- [ ] 2.2 更新 task 状态 + 输出 `task-output.md`（evidence pasting：build/test 实际输出）
