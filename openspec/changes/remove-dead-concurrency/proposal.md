## Why

`foundation/concurrency`（`semaphore.go`、`pool.go`、`rate_limiter.go` 及其 3 个测试）是零生产消费者的死代码：全仓 grep 非测试代码无任何引用（2026-08-01 验证）。其令牌桶 `RateLimiter` 与 `internal/security/ratelimit.go`（按 IP 令牌桶，RPS 100/burst 200）语义重复。死代码的维护成本 > 0（测试要维护、读者要排雷、与活代码的重复实现会漂移）。

## What Changes

- 删除 `foundation/concurrency/` 全部 3 个源文件 + 3 个测试文件
- `foundation/` 目录结构收敛（剩余 `bm25`、`logger`、`metrics`、`store`、`term` 均为活跃代码）
- go.mod 不变（该包无依赖）
- 不引入替代实现——如需限流/信号量原语，后续按"依赖进消费层"原则引官方 `golang.org/x/time/rate` / `x/sync/semaphore`

## Capabilities

### New Capabilities
（无——纯删除变更）

### Modified Capabilities
（无——foundation/concurrency 无对应 spec）

## Impact

- **删除文件**（破坏性变更协议已走：引用图 grep 全仓验证零消费者；用户已确认删除）
- 测试计数 -3（`foundation/concurrency` 3 个测试文件）
- 构建/CI：无影响（无引用路径）
- 文档：CLAUDE.md/AGENTS.md 中无该包描述，无需更新
