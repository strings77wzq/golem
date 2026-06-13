package infra

import (
	"context"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/tools"
)

type mockExecutor struct {
	output string
	err    error
}

func (m *mockExecutor) Execute(ctx context.Context, name string, args ...string) (string, error) {
	return m.output, m.err
}

func TestDockerToolName(t *testing.T) {
	tool := NewDockerTool()
	if tool.Name() != "docker" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "docker")
	}
}

func TestDockerToolNoAction(t *testing.T) {
	tool := NewDockerTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing action")
	}
}

func TestDockerToolUnknownAction(t *testing.T) {
	tool := NewDockerTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{"action": "unknown"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for unknown action")
	}
}

func TestDockerToolPS(t *testing.T) {
	tool := NewDockerTool()
	tool.executor = &mockExecutor{output: "CONTAINER ID  IMAGE  STATUS  NAMES\nabc123  golem:latest  Up 2h  golem-1"}

	result, err := tool.Execute(context.Background(), map[string]interface{}{"action": "ps"})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "golem-1") {
		t.Error("expected container name in output")
	}
}

func TestDockerToolBuild(t *testing.T) {
	tool := NewDockerTool()
	tool.executor = &mockExecutor{output: "Successfully built golem:latest"}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "build",
		"tag":    "golem:latest",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
}

func TestDockerToolBuildNoTag(t *testing.T) {
	tool := NewDockerTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{"action": "build"})
	if err != nil {
		t.Fatal(err)
	}
	// Should still work (tag is optional)
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
}

func TestKubectlToolName(t *testing.T) {
	tool := NewKubectlTool()
	if tool.Name() != "kubectl" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "kubectl")
	}
}

func TestKubectlToolNoAction(t *testing.T) {
	tool := NewKubectlTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing action")
	}
}

func TestKubectlToolGet(t *testing.T) {
	tool := NewKubectlTool()
	tool.executor = &mockExecutor{output: "NAME  READY  STATUS\ngolem  1/1  Running"}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"action":   "get",
		"resource": "pods",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
}

func TestKubectlToolGetNoResource(t *testing.T) {
	tool := NewKubectlTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{"action": "get"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing resource")
	}
}

func TestHelmToolName(t *testing.T) {
	tool := NewHelmTool()
	if tool.Name() != "helm" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "helm")
	}
}

func TestHelmToolNoAction(t *testing.T) {
	tool := NewHelmTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing action")
	}
}

func TestHelmToolList(t *testing.T) {
	tool := NewHelmTool()
	tool.executor = &mockExecutor{output: "NAME  REVISION  STATUS\ngolem  1  deployed"}

	result, err := tool.Execute(context.Background(), map[string]interface{}{"action": "list"})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
}

func TestToolParameterCounts(t *testing.T) {
	tests := []struct {
		name   string
		tool   tools.Tool
		params int
	}{
		{"docker", NewDockerTool(), 10},
		{"kubectl", NewKubectlTool(), 10},
		{"helm", NewHelmTool(), 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.tool.Parameters()
			if len(params) != tt.params {
				t.Errorf("Parameters() = %d, want %d", len(params), tt.params)
			}
		})
	}
}
