package logger

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0,
		Format: FormatJSON,
		Output: &buf,
	})

	log.WithComponent(ComponentAgent).Info("agent started")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["component"] != "agent" {
		t.Errorf("component = %v, want agent", entry["component"])
	}
}

func TestWithComponent_FreeString(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0,
		Format: FormatJSON,
		Output: &buf,
	})

	log.WithComponent("custom-module").Info("custom message")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["component"] != "custom-module" {
		t.Errorf("component = %v, want custom-module", entry["component"])
	}
}

func TestWithComponent_Combined(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{
		Level:  0,
		Format: FormatJSON,
		Output: &buf,
	})

	log.WithComponent(ComponentGateway).With("request_id", "req-123").Info("request received")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry["component"] != "gateway" {
		t.Errorf("component = %v, want gateway", entry["component"])
	}
	if entry["request_id"] != "req-123" {
		t.Errorf("request_id = %v, want req-123", entry["request_id"])
	}
}
