package rag

import (
	"testing"
)

// Test: chunker should not infinite loop on edge cases
func TestChunker_NoInfiniteLoop(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{ChunkSize: 10, ChunkOverlap: 2})

	// Text that previously caused infinite loop
	text := "abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij"
	
	chunks := chunker.Split(text, nil)
	
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	
	// Should complete without timeout
	t.Logf("produced %d chunks", len(chunks))
}

// Test: chunker with no spaces at end
func TestChunker_NoSpacesAtEnd(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{ChunkSize: 10, ChunkOverlap: 2})
	text := "abcdefghijabcdefghijabcdefghij"
	
	chunks := chunker.Split(text, nil)
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
}
