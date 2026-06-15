package think

import (
	"context"
	"testing"
)

func TestThinkTool_Name(t *testing.T) {
	tool := New()
	if tool.Name() != "think" {
		t.Errorf("expected 'think', got %q", tool.Name())
	}
}

func TestThinkTool_Description(t *testing.T) {
	tool := New()
	desc := tool.Description()
	if desc == "" {
		t.Error("description should not be empty")
	}
}

func TestThinkTool_Parameters(t *testing.T) {
	tool := New()
	params := tool.Parameters()
	if len(params) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(params))
	}
	if params[0].Name != "thought" {
		t.Errorf("expected 'thought', got %q", params[0].Name)
	}
	if !params[0].Required {
		t.Error("thought parameter should be required")
	}
}

func TestThinkTool_Execute(t *testing.T) {
	tool := New()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"thought": "Let me analyze this step by step...",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("should not be an error")
	}
	if result.Silent != true {
		t.Error("think tool should be silent (no user output)")
	}
	if result.ForLLM == "" {
		t.Error("ForLLM should not be empty")
	}
}

func TestThinkTool_Execute_EmptyThought(t *testing.T) {
	tool := New()
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("empty thought should return error")
	}
}
