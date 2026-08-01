## Why

自研轮子治理收尾：① `internal/security/ratelimit.go` 的按 IP 令牌桶（~45 行自研）与 P0 已删的 `foundation/concurrency` 限流器语义重复——官方 `golang.org/x/time/rate`（已在依赖图，go-sdk 间接引入）是 Go 团队维护的标准实现；② `core/providers/backoff.go` 的指数退避计算（`calculateDelay`）边界正确性（jitter 公式、MaxElapsedTime 终止）可交给业界标准 `cenkalti/backoff/v4`（16.7k 依赖包，纯 Go）。

## What Changes

- `internal/security/ratelimit.go`：`rateLimitStore` 内部实现换 `golang.org/x/time/rate` 的 `rate.Limiter`（每 IP 一个），外部 API（`allow(ip) (bool, time.Duration)`、默认 RPS 100/burst 200）与中间件签名不变
- `core/providers/backoff.go`：仅替换退避计算层——`calculateDelay` 内部用 `backoff.ExponentialBackOff`（参数映射自现有 `RetryConfig`）；重试判定逻辑（哪些错误重试、429 处理）保留——`RetryProvider` 是 LLMProvider 装饰器，`backoff.Retry` 闭包模型不匹配

## Capabilities

### New Capabilities
（无——内部实现替换，外部行为不变）

### Modified Capabilities
（无）

## Impact

- `internal/security/ratelimit.go` + 测试；`core/providers/backoff.go` + 测试
- go.mod：新增 `golang.org/x/time`（直接）、`cenkalti/backoff/v4`（直接）
- 外部行为：限流语义、RetryConfig 形状、装饰器接口不变
