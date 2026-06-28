package rag

import (
	"testing"

	"github.com/strings77wzq/golem/foundation/bm25"
)

func TestBM25_ChineseSearch(t *testing.T) {
	index := bm25.NewBM25Index()

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
