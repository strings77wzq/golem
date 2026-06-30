package memory

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/strings77wzq/golem/foundation/bm25"
)

type InMemoryStore struct {
	entries map[string]*Entry
	bm25    *bm25.BM25Index
	mu      sync.RWMutex
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		entries: make(map[string]*Entry),
		bm25:    bm25.NewBM25Index(),
	}
}

func buildIndexText(entry *Entry) string {
	var sb strings.Builder
	sb.WriteString(entry.Content)
	for _, tag := range entry.Tags {
		sb.WriteString(" ")
		sb.WriteString(tag)
	}
	return sb.String()
}

func (s *InMemoryStore) Store(ctx context.Context, entry *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.entries[entry.ID]; ok {
		entry.CreatedAt = existing.CreatedAt
	}
	entry.UpdatedAt = time.Now()

	s.entries[entry.ID] = entry
	s.bm25.Add(entry.ID, buildIndexText(entry))
	return nil
}

func (s *InMemoryStore) Recall(ctx context.Context, query string, limit int) ([]*Entry, error) {
	if query == "" {
		return nil, nil
	}

	s.mu.RLock()
	bm25Results := s.bm25.Search(query, limit)

	type scoredEntry struct {
		entry *Entry
		score float64
	}

	var scored []scoredEntry
	now := time.Now()
	for _, r := range bm25Results {
		if entry, ok := s.entries[r.ID]; ok {
			decayed := entry.DecayedImportance(now, DefaultDecayLambda)
			finalScore := r.Score * (1.0 + decayed)
			scored = append(scored, scoredEntry{entry: entry, score: finalScore})
		}
	}
	s.mu.RUnlock()

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	result := make([]*Entry, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].entry)
	}

	s.mu.Lock()
	for _, entry := range result {
		entry.AccessedAt = now
	}
	s.mu.Unlock()

	return result, nil
}

func (s *InMemoryStore) Forget(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bm25.Remove(id)
	delete(s.entries, id)
	return nil
}

func (s *InMemoryStore) List(ctx context.Context) ([]*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	return result, nil
}

// GetTopByRelevance returns the top k entries by relevance score.
func (s *InMemoryStore) GetTopByRelevance(_ context.Context, k int) ([]*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	type scoredEntry struct {
		entry *Entry
		score float64
	}

	scored := make([]scoredEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		decayed := entry.DecayedImportance(now, DefaultDecayLambda)
		score := decayed * math.Log(float64(len(entry.Tags)+1))
		scored = append(scored, scoredEntry{entry: entry, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	result := make([]*Entry, 0, k)
	for i := 0; i < len(scored) && i < k; i++ {
		result = append(result, scored[i].entry)
	}

	return result, nil
}

// Cleanup removes entries with decayed importance below the threshold.
// Important entries (importance >= 0.9) are never deleted.
func (s *InMemoryStore) Cleanup(_ context.Context, threshold float64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	deleted := 0

	for id, entry := range s.entries {
		if entry.Importance >= 0.9 {
			continue
		}

		decayed := entry.DecayedImportance(now, DefaultDecayLambda)
		if decayed < threshold {
			s.bm25.Remove(id)
			delete(s.entries, id)
			deleted++
		}
	}

	return deleted, nil
}
