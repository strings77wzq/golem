package rag

import (
	"context"
	"testing"
)

// Bug: HybridRetriever returns duplicate documents when same doc appears
// in both BM25 and vector results. RRF should deduplicate by ID.
func TestHybridRetriever_DeduplicatesResults(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	// Add same documents to vector store
	store.Add(ctx, []Document{
		{ID: "doc1", Content: "the quick brown fox", Vector: []float64{0.1, 0.2, 0.3}},
		{ID: "doc2", Content: "the lazy dog", Vector: []float64{0.4, 0.5, 0.6}},
	})

	embedder := NewMockEmbedder(3)
	hybrid := NewHybridRetriever(embedder, store, 10)

	// Add same documents to BM25 index
	hybrid.AddDocument("doc1", "the quick brown fox")
	hybrid.AddDocument("doc2", "the lazy dog")

	results, err := hybrid.Search(ctx, "quick fox", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, r := range results {
		if seen[r.ID] {
			t.Errorf("duplicate document in results: %s", r.ID)
		}
		seen[r.ID] = true
	}

	// Should have at most 2 unique results
	if len(results) > 2 {
		t.Errorf("expected at most 2 unique results, got %d", len(results))
	}
}
