## Context

`internal/security/ratelimit.go`：自研 `rateLimitStore`（map[ip]*bucket + cleanup 定时清理，`allow(ip) (bool, time.Duration)` :67）。`core/providers/backoff.go`：自研 `RetryProvider` 装饰器（`calculateDelay(attempt)` 指数退避 + jitter，:90）。两者均有测试。

## Goals / Non-Goals

**Goals:**
- 限流内部实现换 `golang.org/x/time/rate`（官方令牌桶），外部 API 与默认参数不变
- RetryProvider 退避计算层换 `cenkalti/backoff/v4`，重试判定逻辑保留

**Non-Goals:**
- 不改变限流中间件签名与默认值（RPS 100/burst 200）
- 不改变 RetryConfig 形状与重试语义（哪些错误重试、429 处理）
- 不改变 LLMProvider 接口（C3）

## Decisions

- **D1 限流**：`rateLimitStore` 内 map[ip] 的值从自研 bucket 换 `*rate.Limiter`（`rate.NewLimiter(rate.Limit(cfg.Rate), cfg.Burst)`）；`allow(ip)` 调 `limiter.Reserve()` 计算延迟（语义对齐旧返回 `(ok, waitDuration)`）；cleanup 定时清理机制保留（每 IP limiter 不再访问即删除）
- **D2 退避**：`calculateDelay(attempt)` 内部用 `backoff.ExponentialBackOff`（`InitialInterval`/`Multiplier`/`RandomizationFactor`/`MaxInterval`/`MaxElapsedTime` 映射自 RetryConfig 或默认）；`Reset()` 时机与旧实现对齐；jitter 由 backoff 的 RandomizationFactor 提供
- **D3 依赖**：`golang.org/x/time` 已是 go-sdk 间接依赖（提升为直接）；`cenkalti/backoff/v4` 新增

## Risks / Trade-offs

- **R1 行为对齐**：旧 bucket 的 burst 语义 vs rate.Limiter 的 burst——测试逐项断言（允许/拒绝/恢复窗口）
- **R2 backoff 默认参数**：RetryConfig 字段若与 backoff 默认不同，映射显式化（不依赖隐式默认）
- **R3 cleanup 精度**：x/time/rate 的 Limiter 无自带 TTL——保留原 cleanup goroutine 的删除逻辑
