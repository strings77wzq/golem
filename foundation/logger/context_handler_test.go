package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestContextHandler_TraceID(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0, // Debug
		Format: FormatJSON,
		Output: &buf,
	})

	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-abc123def4567890")

	log.WithContext(ctx).Info("test message")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	traceID, ok := entry["trace_id"].(string)
	if !ok {
		t.Fatal("trace_id not found in log entry")
	}
	if traceID != "trace-abc123def4567890" {
		t.Errorf("trace_id = %q, want %q", traceID, "trace-abc123def4567890")
	}
}

func TestContextHandler_NoTraceID(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0,
		Format: FormatJSON,
		Output: &buf,
	})

	log.Info("test message")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if _, ok := entry["trace_id"]; ok {
		t.Error("trace_id should not be present when not in context")
	}
}

func TestContextHandler_WithFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0,
		Format: FormatJSON,
		Output: &buf,
	})

	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-test123")

	log.WithContext(ctx).With("component", "agent").Info("agent started")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["trace_id"] != "trace-test123" {
		t.Errorf("trace_id = %v, want trace-test123", entry["trace_id"])
	}
	if entry["component"] != "agent" {
		t.Errorf("component = %v, want agent", entry["component"])
	}
}

func TestContextHandler_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0,
		Format: FormatText,
		Output: &buf,
	})

	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-text123")

	log.WithContext(ctx).Info("text format test")

	output := buf.String()
	if !strings.Contains(output, "trace_id=trace-text123") {
		t.Errorf("trace_id not found in text output: %s", output)
	}
}
