package rag

import (
	"context"
	"testing"
)

func TestBM25_IndexAndSearch(t *testing.T) {
	index := NewBM25Index()

	docs := []struct {
		id   string
		text string
	}{
		{"doc1", "the quick brown fox jumps over the lazy dog"},
		{"doc2", "the lazy dog sleeps in the sun"},
		{"doc3", "the quick brown fox is very fast"},
	}

	for _, d := range docs {
		index.Add(d.id, d.text)
	}

	results := index.Search("quick fox", 3)
	if len(results) == 0 {
		t.Fatal("expected results")
	}

	if results[0].ID != "doc1" && results[0].ID != "doc3" {
		t.Errorf("expected doc1 or doc3 as top result, got %q", results[0].ID)
	}
}

func TestBM25_EmptyQuery(t *testing.T) {
	index := NewBM25Index()
	index.Add("doc1", "hello world")

	results := index.Search("", 10)
	if len(results) != 0 {
		t.Errorf("expected empty results for empty query, got %d", len(results))
	}
}

func TestBM25_EmptyIndex(t *testing.T) {
	index := NewBM25Index()

	results := index.Search("hello", 10)
	if len(results) != 0 {
		t.Errorf("expected empty results for empty index, got %d", len(results))
	}
}

func TestBM25_ScoringConsistency(t *testing.T) {
	index := NewBM25Index()
	index.Add("doc1", "the cat sat on the mat")
	index.Add("doc2", "the dog played in the park")

	results1 := index.Search("cat", 2)
	results2 := index.Search("cat", 2)

	if len(results1) != len(results2) {
		t.Fatal("inconsistent result count")
	}

	for i := range results1 {
		if results1[i].ID != results2[i].ID {
			t.Errorf("inconsistent ordering: %v vs %v", results1[i].ID, results2[i].ID)
		}
	}
}

func TestBM25_TopKLimit(t *testing.T) {
	index := NewBM25Index()
	for i := 0; i < 10; i++ {
		index.Add("doc"+string(rune('0'+i)), "document about topic "+string(rune('a'+i)))
	}

	results := index.Search("document", 3)
	if len(results) > 3 {
		t.Errorf("expected at most 3 results, got %d", len(results))
	}
}

func TestRRF_SingleRankList(t *testing.T) {
	docs := []ScoredDoc{
		{ID: "doc1", Score: 0.9},
		{ID: "doc2", Score: 0.7},
		{ID: "doc3", Score: 0.5},
	}

	results := ReciprocalRankFusion([][]ScoredDoc{docs}, 60)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].ID != "doc1" {
		t.Errorf("expected doc1 first, got %q", results[0].ID)
	}
}

func TestRRF_MergeTwoLists(t *testing.T) {
	list1 := []ScoredDoc{
		{ID: "doc1", Score: 0.9},
		{ID: "doc2", Score: 0.7},
		{ID: "doc3", Score: 0.5},
	}

	list2 := []ScoredDoc{
		{ID: "doc2", Score: 0.8},
		{ID: "doc4", Score: 0.6},
		{ID: "doc1", Score: 0.4},
	}

	results := ReciprocalRankFusion([][]ScoredDoc{list1, list2}, 60)
	if len(results) != 4 {
		t.Fatalf("expected 4 unique results, got %d", len(results))
	}

	foundDoc1 := false
	foundDoc2 := false
	for _, r := range results {
		if r.ID == "doc1" {
			foundDoc1 = true
		}
		if r.ID == "doc2" {
			foundDoc2 = true
		}
	}
	if !foundDoc1 || !foundDoc2 {
		t.Errorf("doc1 and doc2 should be in results: %v", results)
	}
}

func TestRRF_EmptyInput(t *testing.T) {
	results := ReciprocalRankFusion([][]ScoredDoc{}, 60)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestRRF_EmptyLists(t *testing.T) {
	results := ReciprocalRankFusion([][]ScoredDoc{{}, {}}, 60)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestHybridRetriever(t *testing.T) {
	store := NewMemoryVectorStore()
	ctx := context.Background()

	store.Add(ctx, []Document{
		{ID: "doc1", Content: "the quick brown fox", Vector: []float64{0.1, 0.2, 0.3}},
		{ID: "doc2", Content: "the lazy dog", Vector: []float64{0.4, 0.5, 0.6}},
		{ID: "doc3", Content: "quick fox jumps", Vector: []float64{0.7, 0.8, 0.9}},
	})

	embedder := NewMockEmbedder(3)

	hybrid := NewHybridRetriever(embedder, store, 10, nil)

	hybrid.AddDocument(ctx, "doc1", "the quick brown fox")
	hybrid.AddDocument(ctx, "doc2", "the lazy dog")
	hybrid.AddDocument(ctx, "doc3", "quick fox jumps")

	results, err := hybrid.Search(ctx, "quick fox", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected results from hybrid search")
	}
}
