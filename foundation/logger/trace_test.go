package logger

import (
	"context"
	"regexp"
	"sync"
	"testing"
)

func TestNewTraceID(t *testing.T) {
	id := NewTraceID()

	// Format: "trace-" followed by 16 hex characters
	matched, err := regexp.MatchString(`^trace-[0-9a-f]{16}$`, id)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Errorf("trace ID format mismatch: got %q, want trace-{16hex}", id)
	}
}

func TestNewTraceID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := NewTraceID()
			mu.Lock()
			if seen[id] {
				t.Errorf("duplicate trace ID: %s", id)
			}
			seen[id] = true
			mu.Unlock()
		}()
	}
	wg.Wait()
}

func TestWithTraceID_And_TraceIDFromContext(t *testing.T) {
	ctx := context.Background()
	traceID := "trace-abc123def4567890"

	ctx = WithTraceID(ctx, traceID)
	got := TraceIDFromContext(ctx)

	if got != traceID {
		t.Errorf("TraceIDFromContext() = %q, want %q", got, traceID)
	}
}

func TestTraceIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := TraceIDFromContext(ctx)

	if got != "" {
		t.Errorf("TraceIDFromContext() = %q, want empty string", got)
	}
}
