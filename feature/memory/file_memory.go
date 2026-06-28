package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/strings77wzq/golem/foundation/bm25"
)

type FileMemory struct {
	filePath string
	entries  map[string]*Entry
	bm25     *bm25.BM25Index
	mu       sync.RWMutex
}

func NewFileMemory(filePath string) (*FileMemory, error) {
	fm := &FileMemory{
		filePath: filePath,
		entries:  make(map[string]*Entry),
		bm25:     bm25.NewBM25Index(),
	}

	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read memory file: %w", err)
		}

		var entries []*Entry
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, fmt.Errorf("failed to parse memory file: %w", err)
		}

		for _, entry := range entries {
			fm.entries[entry.ID] = entry
			fm.bm25.Add(entry.ID, buildIndexText(entry))
		}
	}

	return fm, nil
}

func (fm *FileMemory) Store(ctx context.Context, entry *Entry) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if existing, ok := fm.entries[entry.ID]; ok {
		entry.CreatedAt = existing.CreatedAt
	}
	entry.UpdatedAt = time.Now()

	fm.entries[entry.ID] = entry
	fm.bm25.Add(entry.ID, buildIndexText(entry))
	return fm.persist()
}

func (fm *FileMemory) Recall(ctx context.Context, query string, limit int) ([]*Entry, error) {
	if query == "" {
		return nil, nil
	}

	fm.mu.RLock()
	bm25Results := fm.bm25.Search(query, limit)

	type scoredEntry struct {
		entry *Entry
		score float64
	}

	var scored []scoredEntry
	now := time.Now()
	for _, r := range bm25Results {
		if entry, ok := fm.entries[r.ID]; ok {
			decayed := entry.DecayedImportance(now, DefaultDecayLambda)
			finalScore := r.Score * (1.0 + decayed)
			scored = append(scored, scoredEntry{entry: entry, score: finalScore})
		}
	}
	fm.mu.RUnlock()

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	result := make([]*Entry, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].entry)
	}

	fm.mu.Lock()
	for _, entry := range result {
		entry.AccessedAt = now
	}
	fm.mu.Unlock()

	return result, nil
}

func (fm *FileMemory) Forget(ctx context.Context, id string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.bm25.Remove(id)
	delete(fm.entries, id)
	return fm.persist()
}

func (fm *FileMemory) List(ctx context.Context) ([]*Entry, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	result := make([]*Entry, 0, len(fm.entries))
	for _, entry := range fm.entries {
		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	return result, nil
}

func (fm *FileMemory) persist() error {
	entries := make([]*Entry, 0, len(fm.entries))
	for _, entry := range fm.entries {
		entries = append(entries, entry)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}

	dir := filepath.Dir(fm.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmpFile := fm.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpFile, fm.filePath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
