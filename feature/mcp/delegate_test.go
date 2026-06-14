package mcp

import (
	"context"
	"testing"
)

func TestDelegateToolName(t *testing.T) {
	tool := NewDelegateTool()
	if tool.Name() != "delegate" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "delegate")
	}
}

func TestDelegateToolDescription(t *testing.T) {
	tool := NewDelegateTool()
	if tool.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestDelegateToolParameters(t *testing.T) {
	tool := NewDelegateTool()
	params := tool.Parameters()
	if len(params) != 5 {
		t.Errorf("expected 5 parameters, got %d", len(params))
	}
}

func TestDelegateToolMissingCommand(t *testing.T) {
	tool := NewDelegateTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing command")
	}
}

func TestDelegateToolMissingTool(t *testing.T) {
	tool := NewDelegateTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing tool")
	}
}
