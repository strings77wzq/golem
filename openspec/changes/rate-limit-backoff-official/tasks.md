## 1. 限流换 x/time/rate（TDD）

- [ ] 1.1 `go get golang.org/x/time@latest`；基线 `go test ./internal/security/... -race`
- [ ] 1.2 写测试（红→绿迁移）：新实现保持现有测试断言（允许/拒绝/burst/恢复窗口/cleanup）——先跑旧测试确认覆盖，再替换实现
- [ ] 1.3 实现：`ratelimit.go` `rateLimitStore` 内部换 `rate.Limiter`（每 IP）；`allow(ip)` 用 `Reserve()` 对齐 `(bool, time.Duration)`；cleanup 保留
- [ ] 1.4 验证（绿）：`go test ./internal/security/... -race`

## 2. RetryProvider 退避层换 backoff/v4（TDD）

- [ ] 2.1 `go get github.com/cenkalti/backoff/v4`；基线测试
- [ ] 2.2 写测试：`calculateDelay` 单调性 + jitter 边界 + MaxElapsed 终止（对应现有测试语义）
- [ ] 2.3 实现：`backoff.go` `calculateDelay` 内部用 `backoff.ExponentialBackOff`（RetryConfig 映射）；重试判定逻辑不动
- [ ] 2.4 验证（绿）：`go test ./core/providers/... -race`

## 3. 收尾

- [ ] 3.1 全量：`go test ./... -race` + `-cover` ≥80%；vet；golangci-lint
- [ ] 3.2 输出 task-output.md
