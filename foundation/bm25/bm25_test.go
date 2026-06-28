package bm25

import (
	"sync"
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

func TestBM25_AddDuplicateID_ShouldDeduplicate(t *testing.T) {
	index := NewBM25Index()

	index.Add("doc1", "hello world")
	index.Add("doc1", "hello world again")

	if index.numDocs != 1 {
		t.Errorf("expected numDocs=1 after duplicate add (dedup), got %d", index.numDocs)
	}

	results := index.Search("again", 10)
	if len(results) == 0 {
		t.Fatal("expected results for 'again' (updated text)")
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestBM25_Search_NegativeTopK(t *testing.T) {
	index := NewBM25Index()
	index.Add("doc1", "hello world")

	results := index.Search("hello", -1)
	if results != nil {
		t.Errorf("expected nil for negative topK, got %d results", len(results))
	}
}

func TestBM25_Remove(t *testing.T) {
	index := NewBM25Index()
	index.Add("doc1", "hello world")
	index.Add("doc2", "foo bar")

	if !index.Remove("doc1") {
		t.Error("expected Remove to return true for existing doc")
	}

	if index.numDocs != 1 {
		t.Errorf("expected numDocs=1 after remove, got %d", index.numDocs)
	}

	results := index.Search("hello", 10)
	if len(results) != 0 {
		t.Errorf("expected no results for removed doc, got %d", len(results))
	}

	if index.Remove("nonexistent") {
		t.Error("expected Remove to return false for nonexistent doc")
	}
}

func TestBM25_ChineseSearch(t *testing.T) {
	index := NewBM25Index()

	docs := []struct {
		id   string
		text string
	}{
		{"doc1", "数据库查询优化技巧"},
		{"doc2", "Python编程入门教程"},
		{"doc3", "数据库索引设计原则"},
	}

	for _, d := range docs {
		index.Add(d.id, d.text)
	}

	results := index.Search("数据库", 3)
	if len(results) == 0 {
		t.Fatal("expected results for Chinese query")
	}

	found := make(map[string]bool)
	for _, r := range results {
		found[r.ID] = true
	}

	if !found["doc1"] {
		t.Error("doc1 should be in results (contains '数据库')")
	}
	if !found["doc3"] {
		t.Error("doc3 should be in results (contains '数据库')")
	}
}

func TestBM25_TokenizeChinese(t *testing.T) {
	terms := tokenize("数据库查询")
	if len(terms) != 5 {
		t.Errorf("expected 5 tokens for Chinese text (char-level), got %d: %v", len(terms), terms)
	}
	expected := []string{"数", "据", "库", "查", "询"}
	for i, exp := range expected {
		if terms[i] != exp {
			t.Errorf("terms[%d] = %q, want %q", i, terms[i], exp)
		}
	}
}

func TestBM25Race_ConcurrentAddAndSearch(t *testing.T) {
	index := NewBM25Index()

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				index.Add("doc"+string(rune('A'+n))+string(rune('0'+j)), "document about topic "+string(rune('a'+j)))
			}
		}(i)
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				results := index.Search("document", 3)
				_ = results
			}
		}()
	}

	wg.Wait()
}

func TestBM25Search_NoMatchReturnsEmpty(t *testing.T) {
	index := NewBM25Index()
	index.Add("doc1", "hello world")

	results := index.Search("xyz123", 10)
	if len(results) != 0 {
		t.Errorf("expected empty results for non-matching query, got %d", len(results))
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
