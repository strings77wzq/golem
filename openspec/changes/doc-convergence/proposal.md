## Why

文档与实现漂移：① `feature/rag/pipeline.go:2` 注释声称 TF-IDF，实际关键字检索是 BM25（`feature/rag/hybrid.go:11`、`foundation/bm25/`）——误导维护者与读者；② 项目 CLAUDE.md providers 段描述 `registerProviders()` 为"case 分支"注册，实际是 registry 驱动的动态注册（各 vendor 包 `init()` 自注册，`internal/wiring/providers.go:16-55` 无 switch）。

## What Changes

- `feature/rag/pipeline.go:2` 注释修正：TF-IDF → BM25（与 `hybrid.go:11` 一致）
- 项目 `CLAUDE.md` providers 段：`registerProviders()` 描述修正为 registry 动态注册（vendors 由各包 `init()` 注册）

## Capabilities

### New Capabilities
（无——文档修正，无行为变化）

### Modified Capabilities
（无）

## Impact

- 2 个文件、零行为变化；注释与代码一致性恢复
