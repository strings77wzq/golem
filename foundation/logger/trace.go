package logger

import (
	"context"
	"crypto/rand"
	"fmt"
)

type traceIDKey struct{}

// NewTraceID generates a unique trace ID in the format "trace-{16hex}".
func NewTraceID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("trace-%x", b)
}

// WithTraceID attaches a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// TraceIDFromContext extracts the trace ID from context.
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey{}).(string); ok {
		return v
	}
	return ""
}
