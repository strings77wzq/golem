package memory

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreAndRecall(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	// Store some memories
	mem.Store(ctx, NewEntry("Golem is a Go AI agent", "golang", "agent"))
	mem.Store(ctx, NewEntry("Golem uses SQLite for storage", "sqlite", "database"))
	mem.Store(ctx, NewEntry("Python is popular for AI", "python", "ai"))

	// Recall by keyword
	results, err := mem.Recall(ctx, "golang", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'golang', got %d", len(results))
	}
	if results[0].Content != "Golem is a Go AI agent" {
		t.Errorf("unexpected result: %s", results[0].Content)
	}
}

func TestMemoryRecallByTag(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	mem.Store(ctx, NewEntry("Golem uses SQLite", "sqlite"))
	mem.Store(ctx, NewEntry("Golem uses PostgreSQL", "postgres"))
	mem.Store(ctx, NewEntry("Golem uses Redis", "redis"))

	results, err := mem.Recall(ctx, "sqlite", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'sqlite', got %d", len(results))
	}
}

func TestMemoryRecallRelevance(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	mem.Store(ctx, NewEntry("Golem is great", "golang"))
	mem.Store(ctx, NewEntry("Golem Golem Golem is amazing", "golang"))

	results, err := mem.Recall(ctx, "golang", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Entry with more "golang" mentions should rank higher
	if results[0].Content != "Golem Golem Golem is amazing" {
		t.Errorf("expected higher-relevance entry first, got: %s", results[0].Content)
	}
}

func TestMemoryImportanceDecay(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	// Store entries with different importance levels
	entry1 := NewEntry("Low importance", "low")
	entry1.Importance = 0.1
	entry1.AccessedAt = time.Now().Add(-1000 * time.Hour) // Very old

	entry2 := NewEntry("High importance", "high")
	entry2.Importance = 1.0

	mem.Store(ctx, entry1)
	mem.Store(ctx, entry2)

	// Cleanup with threshold above the decayed value of entry1
	// entry1: 0.1 * exp(-0.001 * 1000) ≈ 0.1 * 0.368 ≈ 0.037
	// threshold 0.05 should delete entry1
	deleted, err := mem.Cleanup(ctx, 0.05)
	if err != nil {
		t.Fatal(err)
	}

	// Old entry with low importance should be deleted
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	// New entry should remain
	results, _ := mem.List(ctx)
	if len(results) != 1 {
		t.Errorf("expected 1 remaining entry, got %d", len(results))
	}
}

func TestMemoryCleanupPreservesHighImportance(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	// Entry with high importance should never be deleted
	entry := NewEntry("Critical memory", "critical")
	entry.Importance = 0.9
	entry.AccessedAt = time.Now().Add(-1000 * time.Hour)

	mem.Store(ctx, entry)

	deleted, err := mem.Cleanup(ctx, 0.0) // threshold 0 = delete everything below
	if err != nil {
		t.Fatal(err)
	}

	if deleted != 0 {
		t.Errorf("expected 0 deleted (importance >= 0.9 preserved), got %d", deleted)
	}
}

func TestMemoryForget(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	entry := NewEntry("Temporary memory", "temp")
	mem.Store(ctx, entry)

	// Verify stored
	results, _ := mem.Recall(ctx, "temporary", 10)
	if len(results) != 1 {
		t.Fatal("expected memory to be stored")
	}

	// Forget
	if err := mem.Forget(ctx, entry.ID); err != nil {
		t.Fatal(err)
	}

	// Verify forgotten
	results, _ = mem.Recall(ctx, "temporary", 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results after forget, got %d", len(results))
	}
}

func TestMemoryList(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	mem.Store(ctx, NewEntry("First", "tag1"))
	mem.Store(ctx, NewEntry("Second", "tag2"))
	mem.Store(ctx, NewEntry("Third", "tag3"))

	results, err := mem.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 entries, got %d", len(results))
	}
}

func TestMemoryGetTopByRelevance(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	low := NewEntry("Low relevance", "tag1")
	low.Importance = 0.1

	high := NewEntry("High relevance", "tag1", "tag2", "tag3")
	high.Importance = 1.0

	mem.Store(ctx, low)
	mem.Store(ctx, high)

	results, err := mem.GetTopByRelevance(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// High importance + more tags should rank higher
	if results[0].Content != "High relevance" {
		t.Errorf("expected high-relevance entry first, got: %s", results[0].Content)
	}
}

func TestMemoryConcurrentAccess(t *testing.T) {
	mem := NewInMemoryStore()
	ctx := context.Background()

	// Concurrent store
	for i := 0; i < 10; i++ {
		go func(n int) {
			mem.Store(ctx, NewEntry("concurrent", "test"))
		}(i)
	}

	// Concurrent recall
	for i := 0; i < 10; i++ {
		go func() {
			mem.Recall(ctx, "concurrent", 5)
		}()
	}

	// Concurrent list
	for i := 0; i < 10; i++ {
		go func() {
			mem.List(ctx)
		}()
	}

	// Wait a bit for goroutines
	time.Sleep(10 * time.Millisecond)

	// Verify no panic occurred
	results, _ := mem.List(ctx)
	if len(results) == 0 {
		t.Error("expected at least one entry after concurrent operations")
	}
}
