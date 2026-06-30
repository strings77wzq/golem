package agent

import (
	"context"

	"github.com/strings77wzq/golem/foundation/logger"
)

// WithTraceID attaches a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return logger.WithTraceID(ctx, traceID)
}

// TraceIDFromContext extracts the trace ID from context.
func TraceIDFromContext(ctx context.Context) string {
	return logger.TraceIDFromContext(ctx)
}

// NewTraceID generates a unique trace ID.
// Deprecated: Use foundation/logger.NewTraceID() directly for new code.
func NewTraceID() string {
	return logger.NewTraceID()
}
