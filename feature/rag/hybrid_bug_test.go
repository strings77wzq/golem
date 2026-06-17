package rag

import (
	"context"
	"testing"
)

// Bug: AddDocument only writes to BM25, not vector store
// After AddDocument, vector search should find the document
func TestHybridRetriever_AddDocument_PopulatesVectorStore(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	embedder := NewMockEmbedder(3)
	hybrid := NewHybridRetriever(embedder, store, 10)

	// Add document
	err := hybrid.AddDocument(ctx, "doc1", "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify BM25 has it
	bm25Results := hybrid.bm25.Search("hello", 10)
	if len(bm25Results) == 0 {
		t.Error("BM25 should find doc1")
	}

	// Verify vector store has it
	vecResults, err := store.Search(ctx, []float64{0.1, 0.2, 0.3}, 10)
	if err != nil {
		t.Fatalf("vector search error: %v", err)
	}
	if len(vecResults) == 0 {
		t.Error("vector store should have doc1 after AddDocument")
	}
}

// Bug: AddDocument with nil embedder should still work (BM25 only)
func TestHybridRetriever_AddDocument_NilEmbedder(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	hybrid := NewHybridRetriever(nil, store, 10)

	err := hybrid.AddDocument(ctx, "doc1", "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// BM25 should work
	results := hybrid.bm25.Search("hello", 10)
	if len(results) == 0 {
		t.Error("BM25 should find doc1")
	}
}
