package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestLogToolCall_Success(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0,
		Format: FormatJSON,
		Output: &buf,
	})

	LogToolCall(log, "sql_query", 123*time.Millisecond, nil)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["tool"] != "sql_query" {
		t.Errorf("tool = %v, want sql_query", entry["tool"])
	}
	if entry["duration_ms"] != float64(123) {
		t.Errorf("duration_ms = %v, want 123", entry["duration_ms"])
	}
	if entry["msg"] != "tool executed" {
		t.Errorf("msg = %v, want 'tool executed'", entry["msg"])
	}
}

func TestLogToolCall_Error(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0,
		Format: FormatJSON,
		Output: &buf,
	})

	LogToolCall(log, "sql_query", 50*time.Millisecond, errors.New("table not found"))

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["msg"] != "tool execution failed" {
		t.Errorf("msg = %v, want 'tool execution failed'", entry["msg"])
	}
	if entry["error"] != "table not found" {
		t.Errorf("error = %v, want 'table not found'", entry["error"])
	}
}

func TestLogHTTPRequest_Success(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0,
		Format: FormatJSON,
		Output: &buf,
	})

	LogHTTPRequest(log, "GET", "/health", 200, 10*time.Millisecond)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["method"] != "GET" {
		t.Errorf("method = %v, want GET", entry["method"])
	}
	if entry["path"] != "/health" {
		t.Errorf("path = %v, want /health", entry["path"])
	}
	if entry["status"] != float64(200) {
		t.Errorf("status = %v, want 200", entry["status"])
	}
	if entry["msg"] != "http request" {
		t.Errorf("msg = %v, want 'http request'", entry["msg"])
	}
}

func TestLogHTTPRequest_Error(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0,
		Format: FormatJSON,
		Output: &buf,
	})

	LogHTTPRequest(log, "POST", "/api/chat", 500, 100*time.Millisecond)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// JSON output doesn't have level field, but the msg should be present
	if entry["msg"] != "http request" {
		t.Errorf("msg = %v, want 'http request'", entry["msg"])
	}
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0,
		Format: FormatJSON,
		Output: &buf,
	})

	LogError(log, errors.New("connection failed"), "database error", "host", "localhost")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["error"] != "connection failed" {
		t.Errorf("error = %v, want 'connection failed'", entry["error"])
	}
	if entry["host"] != "localhost" {
		t.Errorf("host = %v, want localhost", entry["host"])
	}
}
