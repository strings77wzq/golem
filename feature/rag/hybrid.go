package rag

import (
	"context"
	"fmt"
)

// HybridRetriever combines BM25 keyword search with vector similarity search.
type HybridRetriever struct {
	embedder    Embedder
	vectorStore VectorStore
	bm25        *BM25Index
	topK        int
}

// NewHybridRetriever creates a new hybrid retriever.
func NewHybridRetriever(embedder Embedder, store VectorStore, topK int) *HybridRetriever {
	return &HybridRetriever{
		embedder:    embedder,
		vectorStore: store,
		bm25:        NewBM25Index(),
		topK:        topK,
	}
}

// AddDocument adds a document to both BM25 and vector indexes.
func (h *HybridRetriever) AddDocument(id, content string) {
	h.bm25.Add(id, content)
}

// Search performs hybrid search combining BM25 and vector similarity.
func (h *HybridRetriever) Search(ctx context.Context, query string, topK int) ([]ScoredDoc, error) {
	if topK == 0 {
		topK = h.topK
	}

	// BM25 search
	bm25Results := h.bm25.Search(query, topK*2)

	// Vector search (if embedder is available)
	var vectorResults []ScoredDoc
	if h.embedder != nil {
		queryVec, err := h.embedder.Embed(ctx, query)
		if err == nil {
			searchResults, err := h.vectorStore.Search(ctx, queryVec, topK*2)
			if err == nil {
				for _, sr := range searchResults {
					vectorResults = append(vectorResults, ScoredDoc{
						ID:    sr.Document.ID,
						Score: sr.Score,
					})
				}
			}
		}
	}

	// Fuse results using RRF
	allLists := [][]ScoredDoc{bm25Results}
	if len(vectorResults) > 0 {
		allLists = append(allLists, vectorResults)
	}

	fused := ReciprocalRankFusion(allLists, 60)

	if topK > len(fused) {
		topK = len(fused)
	}
	return fused[:topK], nil
}

// String returns a debug representation.
func (d ScoredDoc) String() string {
	return fmt.Sprintf("%s (score=%.4f)", d.ID, d.Score)
}
