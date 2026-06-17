package rag

import (
	"strconv"
	"testing"
)

// Bug: chunk_index uses rune('0' + index), breaks at index >= 10
func TestChunker_ChunkIndexFormat(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{ChunkSize: 10, ChunkOverlap: 2})

	// Create text that produces 12+ chunks
	text := "abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij abcdefghij"
	chunks := chunker.Split(text, nil)

	if len(chunks) < 11 {
		t.Fatalf("expected at least 11 chunks, got %d", len(chunks))
	}

	// Check that chunk_index is correct numeric string for all chunks
	for i, chunk := range chunks {
		expected := strconv.Itoa(i)
		actual := chunk.Metadata["chunk_index"]
		if actual != expected {
			t.Errorf("chunk %d: expected chunk_index=%q, got %q", i, expected, actual)
		}
	}
}

// Bug: chunk_index for single chunk should be "0"
func TestChunker_SingleChunk_Index(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{ChunkSize: 1000})
	chunks := chunker.Split("short text", nil)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Metadata["chunk_index"] != "0" {
		t.Errorf("expected chunk_index='0', got %q", chunks[0].Metadata["chunk_index"])
	}
}
