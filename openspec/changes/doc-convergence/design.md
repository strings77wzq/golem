## Context

`feature/rag/pipeline.go:2` 注释声称 TF-IDF；实现为 BM25（`feature/rag/hybrid.go:11`、`foundation/bm25/bm25.go`）。项目 `CLAUDE.md` providers 段描述 `registerProviders()` 为 case 分支注册；实际为 registry 动态注册（`internal/wiring/providers.go:16-55`，vendors 由各包 `init()` 自注册）。

## Goals / Non-Goals

**Goals:**
- 注释与代码一致（TF-IDF → BM25）
- CLAUDE.md provider 描述与实际实现一致

**Non-Goals:**
- 不改实现、不改行为、不扩范围（其他文档债另行处理）

## Decisions

- **D1** `feature/rag/pipeline.go:2` 注释改为 BM25（与 hybrid.go:11 措辞一致）
- **D2** 项目 CLAUDE.md "Providers" 段：case 分支描述 → registry 动态注册描述

## Risks / Trade-offs

- 无（纯文档；gofmt 无关）
