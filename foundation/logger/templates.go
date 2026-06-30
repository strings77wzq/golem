package logger

import (
	"time"
)

// LogToolCall logs a tool execution with structured fields.
func LogToolCall(log Logger, toolName string, duration time.Duration, err error) {
	args := []any{
		"tool", toolName,
		"duration_ms", duration.Milliseconds(),
	}
	if err != nil {
		log.Error("tool execution failed", append(args, "error", err)...)
	} else {
		log.Info("tool executed", args...)
	}
}

// LogHTTPRequest logs an HTTP request with structured fields.
func LogHTTPRequest(log Logger, method, path string, status int, duration time.Duration) {
	args := []any{
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", duration.Milliseconds(),
	}
	switch {
	case status >= 500:
		log.Error("http request", args...)
	case status >= 400:
		log.Warn("http request", args...)
	default:
		log.Info("http request", args...)
	}
}

// LogError logs an error with optional context fields.
func LogError(log Logger, err error, msg string, args ...any) {
	allArgs := append([]any{"error", err}, args...)
	log.Error(msg, allArgs...)
}
