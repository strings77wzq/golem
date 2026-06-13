package agent

import (
	"context"
	"crypto/rand"
	"fmt"
)

type contextKey struct{}

// WithTraceID attaches a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, contextKey{}, traceID)
}

// TraceIDFromContext extracts the trace ID from context.
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKey{}).(string); ok {
		return v
	}
	return ""
}

// NewTraceID generates a unique trace ID.
func NewTraceID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("trace-%x", b)
}
