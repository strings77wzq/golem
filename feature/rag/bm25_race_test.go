package rag

import (
	"sync"
	"testing"
)

// Bug: BM25Index is not thread-safe. Concurrent Add + Search causes data race.
func TestBM25Race_ConcurrentAddAndSearch(t *testing.T) {
	index := NewBM25Index()

	var wg sync.WaitGroup

	// Writer goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				index.Add("doc"+string(rune('A'+n))+string(rune('0'+j)), "document about topic "+string(rune('a'+j)))
			}
		}(i)
	}

	// Reader goroutines
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

// Bug: Search returns results even when query terms don't match any document.
func TestBM25Search_NoMatchReturnsEmpty(t *testing.T) {
	index := NewBM25Index()
	index.Add("doc1", "hello world")

	results := index.Search("xyz123", 10)
	if len(results) != 0 {
		t.Errorf("expected empty results for non-matching query, got %d", len(results))
	}
}
