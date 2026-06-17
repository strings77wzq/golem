# Chunker Infinite Loop Fix — Spec

## [S1] 问题

`chunker.go:79-83` 的 overlap 守卫在某些边界条件下无法阻止无限循环。

**复现条件：**
- ChunkSize=10, ChunkOverlap=2
- 文本末尾没有空格
- findWordBoundary 返回的位置与当前 start 相同

**根因：** 当 `actualEnd == prevStart` 时，guard `start <= prevStart` 为 true，将 start 重置为 `actualEnd`。但下一次迭代中 `actualEnd` 仍然相同，导致无限循环。

## [S2] 方案选择

选择方案 B：检测 `actualEnd <= prevStart` 时直接 break。

**理由：**
1. 无限循环是最严重的 bug，优先防止
2. 丢失最后几个字符是可接受的降级
3. 改动最小，风险最低

## [S3] 实现

```go
prevStart := start
start = actualEnd - c.config.ChunkOverlap
if start <= prevStart {
    break // 防止无限循环，丢失剩余内容
}
```

## [S4] 测试

- `TestChunker_NoInfiniteLoop`：验证不超时
- `TestChunker_ChunkIndexFormat`：验证 index≥10 时正确
- 现有 `TestChunker` 测试必须通过

## [S5] 验证

```bash
go test -race ./feature/rag/ -timeout 30s
```
