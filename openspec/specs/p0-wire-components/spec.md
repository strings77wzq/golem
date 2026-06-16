# P0: Wire Existing Components — SDD

## [S1] Problem

Golem has 3 implemented but unwired components:
1. **RetryProvider** — exponential backoff retry, not connected to provider factory
2. **Compactor** — LLM-driven session compression, not connected to agent loop (old string truncation still used)
3. **HybridRetriever** — BM25+RRF hybrid search, not connected to RAG pipeline (vector-only search still used)

This means the codebase has dead code that was implemented but never activated.

## [S2] Solution

### Task 1: Wire RetryProvider into Factory

**File:** `internal/wiring/providers.go`

Change: Wrap each registered provider with `RetryProvider` using default config.

```go
// Before
factory.Register(vendor, openai.New(entry.APIKey, opts...))

// After
factory.Register(vendor, providers.NewRetryProvider(
    openai.New(entry.APIKey, opts...),
    providers.RetryConfig{},
))
```

**Test:** `internal/wiring/providers_test.go` — verify providers are wrapped.

### Task 2: Wire Compactor into Agent Loop

**File:** `core/agent/loop.go`

Change: Replace old `HandleCompact` (string truncation) with new `Compactor`.

1. Add `compactor *Compactor` field to `Agent` struct
2. Add `WithCompactor(compactor *Compactor) Option`
3. In `HandleCompact`, use `compactor.Compact()` instead of old truncation
4. Auto-trigger compaction when token count exceeds 80% of budget

**File:** `cmd/golem/run.go`

Change: Create Compactor and pass to Agent via `WithCompactor()`.

**Test:** `core/agent/compactor_integration_test.go` — verify compaction works end-to-end.

### Task 3: Wire HybridRetriever into RAG Pipeline

**File:** `feature/rag/retriever.go`

Change: Add `HybridQuery()` method that uses BM25+vector+RRF.

```go
func (r *Retriever) HybridQuery(ctx context.Context, query string) ([]SearchResult, error) {
    // 1. BM25 search
    bm25Results := r.hybrid.bm25.Search(query, r.topK*2)
    
    // 2. Vector search
    queryVec, err := r.embedder.Embed(ctx, query)
    vectorResults := r.store.Search(ctx, queryVec, r.topK*2)
    
    // 3. RRF fusion
    fused := ReciprocalRankFusion([][]ScoredDoc{bm25Results, vectorResults}, 60)
    
    return fused, nil
}
```

**File:** `cmd/golem/adapter.go`

Change: Use `HybridQuery()` when RAG is enabled.

**Test:** `feature/rag/hybrid_integration_test.go` — verify hybrid search works.

## [S3] Verification

```bash
go test -race ./internal/wiring/ ./core/agent/ ./feature/rag/
go vet ./...
go build ./cmd/golem/
```

## [S4] Risk

| Risk | Mitigation |
|------|------------|
| RetryProvider wrapping changes error behavior | Configurable: default on, can disable |
| Compactor auto-trigger might summarize too aggressively | 80% threshold, configurable |
| HybridRetriever BM25 without embedder | Graceful fallback to vector-only |
