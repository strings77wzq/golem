# Critical Bug Fixes — Spec v2

## [S1] 问题分析

4 个 Critical bug，全部在 `feature/rag/` 模块：

| Bug | 文件 | 根因 | 后果 |
|-----|------|------|------|
| 1 | `hybrid.go:27-29` | `AddDocument` 只写 BM25，不写向量存储 | 混合搜索退化为纯 BM25 |
| 2 | `chunker.go:63,75` | `chunk_index` 用 `rune('0'+index)` | index≥10 时元数据损坏 |
| 3 | `chunker.go:79` | overlap 守卫比较 byte offset 与 chunk index | 无限循环风险 |
| 4 | `embedded.go:114-165` | switch-case 复制 register.go 逻辑 | 维护成本翻倍 |

## [S2] 方案选择

### Bug 1: 已修复 ✓
- 修改 `AddDocument` 签名添加 `ctx context.Context` 参数
- 内部调用 `embedder.Embed()` 并写入 `vectorStore`
- 错误返回给调用方

### Bug 2 & 3: chunker.go
- 用 `strconv.Itoa(index)` 替代 `string(rune('0'+index))`
- 用 `prevStart` 追踪前一个 start 位置，替代错误的 `chunks[len(chunks)-1].Index` 比较

### Bug 4: embedded.go
- 删除 50 行 switch-case
- 使用 `providers.GlobalRegistry.Create()` 替代
- 依赖 openai/register.go、anthropic/register.go、ollama/register.go 的 init() 注册

## [S3] TDD 计划

| Bug | 测试 | 预期 |
|-----|------|------|
| 1 | `TestHybridRetriever_AddDocument_PopulatesVectorStore` | AddDocument 后向量存储有数据 |
| 2 | `TestChunker_ChunkIndexFormat` | index≥10 时 chunk_index 正确 |
| 3 | `TestChunker_SingleChunk_Index` | 单 chunk 时 chunk_index="0" |
| 4 | `TestEmbeddedProviderFactory` | 使用 Registry 创建 provider |

## [S4] 验证

```bash
go test -race ./feature/rag/ ./core/agent/
go test ./...
go build ./cmd/golem/
```
