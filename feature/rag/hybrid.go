package rag

import (
	"context"
	"fmt"

	"github.com/strings77wzq/golem/foundation/bm25"
	"github.com/strings77wzq/golem/foundation/logger"
)

// HybridRetriever combines BM25 keyword search with vector similarity search.
type HybridRetriever struct {
	embedder    Embedder
	vectorStore VectorStore
	bm25        *bm25.BM25Index
	topK        int
	log         logger.Logger
}

// NewHybridRetriever creates a new hybrid retriever.
func NewHybridRetriever(embedder Embedder, store VectorStore, topK int, log logger.Logger) *HybridRetriever {
	return &HybridRetriever{
		embedder:    embedder,
		vectorStore: store,
		bm25:        bm25.NewBM25Index(),
		topK:        topK,
		log:         log,
	}
}

// AddDocument adds a document to both BM25 and vector indexes.
func (h *HybridRetriever) AddDocument(ctx context.Context, id, content string) error {
	h.bm25.Add(id, content)

	if h.embedder != nil {
		vec, err := h.embedder.Embed(ctx, content)
		if err != nil {
			return fmt.Errorf("embedding document %s: %w", id, err)
		}
		h.vectorStore.Add(ctx, []Document{{ //nolint:errcheck
			ID:      id,
			Content: content,
			Vector:  vec,
		}})
	}
	return nil
}

// Search performs hybrid search combining BM25 and vector similarity.
func (h *HybridRetriever) Search(ctx context.Context, query string, topK int) ([]bm25.ScoredDoc, error) {
	if topK == 0 {
		topK = h.topK
	}

	bm25Results := h.bm25.Search(query, topK*2)

	var vectorResults []bm25.ScoredDoc
	if h.embedder != nil {
		queryVec, err := h.embedder.Embed(ctx, query)
		if err != nil {
			if h.log != nil {
				h.log.Warn("hybrid search: embedding failed, falling back to BM25 only", "error", err)
			}
		} else {
			searchResults, err := h.vectorStore.Search(ctx, queryVec, topK*2)
			if err != nil {
				if h.log != nil {
					h.log.Warn("hybrid search: vector store search failed, falling back to BM25 only", "error", err)
				}
			} else {
				for _, sr := range searchResults {
					vectorResults = append(vectorResults, bm25.ScoredDoc{
						ID:    sr.Document.ID,
						Score: sr.Score,
					})
				}
			}
		}
	}

	allLists := [][]bm25.ScoredDoc{bm25Results}
	if len(vectorResults) > 0 {
		allLists = append(allLists, vectorResults)
	}

	fused := bm25.ReciprocalRankFusion(allLists, 60)

	if topK > len(fused) {
		topK = len(fused)
	}
	return fused[:topK], nil
}
