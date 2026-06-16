package rag

import (
	"testing"
)

// Bug: BM25 tokenizer uses whitespace splitting, which doesn't work for Chinese.
// Chinese text like "数据库查询优化" has no spaces, so it's treated as one token.
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

	// Search for Chinese term "数据库" (database)
	results := index.Search("数据库", 3)
	if len(results) == 0 {
		t.Fatal("expected results for Chinese query")
	}

	// doc1 and doc3 should rank higher (contain "数据库")
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

// Chinese text is tokenized character-level (correct behavior for CJK).
func TestBM25_TokenizeChinese(t *testing.T) {
	terms := tokenize("数据库查询")
	// Character-level tokenization: each CJK char is a separate token
	if len(terms) != 5 {
		t.Errorf("expected 5 tokens for Chinese text (char-level), got %d: %v", len(terms), terms)
	}
	// Verify each character is a separate token
	expected := []string{"数", "据", "库", "查", "询"}
	for i, exp := range expected {
		if terms[i] != exp {
			t.Errorf("terms[%d] = %q, want %q", i, terms[i], exp)
		}
	}
}
