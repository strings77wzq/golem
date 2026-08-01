## Context

`foundation/concurrency`（3 源文件 + 3 测试）经全仓 grep（2026-08-01）验证：非测试代码零引用。功能上被 `internal/security/ratelimit.go` 的按 IP 令牌桶覆盖（语义重复）。

## Goals / Non-Goals

**Goals:**
- 删除 `foundation/concurrency` 全部 6 个文件
- 消除重复限流概念，收敛 foundation 目录

**Non-Goals:**
- 不引入替代实现（`x/time/rate` 等留待真实消费方出现，按"依赖进消费层"原则）

## Decisions

- **D1 删除范围**：`foundation/concurrency/{semaphore.go, pool.go, rate_limiter.go}` + `{semaphore,pool,rate_limiter}_test.go`（文件名以实际 ls 为准）
- **D2 回归**：删除后 `go build ./...` + `go test ./... -race` 全量（预期 0 影响）

## Risks / Trade-offs

- 风险：无（零消费者已验证）
- Trade-off：若未来内部确实需要通用并发原语，将引官方库而非复活本包——这是有意决策
