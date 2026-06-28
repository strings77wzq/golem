package rag

import (
	"testing"
)

func TestBM25_AddDuplicateID_ShouldDeduplicate(t *testing.T) {
	index := NewBM25Index()

	index.Add("doc1", "hello world")
	index.Add("doc1", "hello world again")

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
