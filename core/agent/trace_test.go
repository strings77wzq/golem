package agent

import (
	"context"
	"testing"
)

func TestNewTraceID(t *testing.T) {
	id1 := NewTraceID()
	id2 := NewTraceID()

	if id1 == "" {
		t.Error("expected non-empty trace ID")
	}
	if id1 == id2 {
		t.Error("expected different trace IDs")
	}
	if len(id1) < 10 {
		t.Errorf("trace ID too short: %q", id1)
	}
}

func TestTraceIDFromContext(t *testing.T) {
	ctx := context.Background()

	// No trace ID in empty context
	if id := TraceIDFromContext(ctx); id != "" {
		t.Errorf("expected empty trace ID, got %q", id)
	}

	// With trace ID
	ctx = WithTraceID(ctx, "trace-abc123")
	if id := TraceIDFromContext(ctx); id != "trace-abc123" {
		t.Errorf("expected 'trace-abc123', got %q", id)
	}
}

func TestTraceIDPropagation(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-parent")

	// Simulate passing context to child function
	childFunc := func(ctx context.Context) string {
		return TraceIDFromContext(ctx)
	}

	if id := childFunc(ctx); id != "trace-parent" {
		t.Errorf("expected 'trace-parent', got %q", id)
	}
}

func TestTraceIDReuseFromContext(t *testing.T) {
	// When gateway passes trace_id in context, agent should reuse it
	gatewayTraceID := "trace-gateway-abc123"
	ctx := WithTraceID(context.Background(), gatewayTraceID)

	// Simulate what processMessage should do: reuse existing trace_id
	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = NewTraceID()
	}

	if traceID != gatewayTraceID {
		t.Errorf("expected gateway trace_id %q, got %q", gatewayTraceID, traceID)
	}
}

func TestTraceIDGenerateWhenMissing(t *testing.T) {
	// When no trace_id in context, agent should generate new one
	ctx := context.Background()

	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = NewTraceID()
		ctx = WithTraceID(ctx, traceID)
	}

	if traceID == "" {
		t.Error("expected non-empty trace ID")
	}
	if TraceIDFromContext(ctx) != traceID {
		t.Error("trace ID not stored in context")
	}
}
