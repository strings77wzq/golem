package planner

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/tools"
)

type mockLLM struct {
	response string
	err      error
}

func (m *mockLLM) Chat(ctx context.Context, messages []providers.Message, toolDefs []tools.ToolDefinition, model string, opts *providers.ChatOptions) (*providers.LLMResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &providers.LLMResponse{Content: m.response}, nil
}

func (m *mockLLM) Name() string { return "mock" }

func TestDecomposeSuccess(t *testing.T) {
	planResp := planResponse{
		Steps: []planStepJSON{
			{Description: "Build Docker image", ToolHints: []string{"exec"}, ExpectedOut: "image built"},
			{Description: "Push to registry", ToolHints: []string{"exec"}, ExpectedOut: "image pushed"},
		},
	}
	respJSON, _ := json.Marshal(planResp)

	llm := &mockLLM{response: string(respJSON)}
	p := NewPlanner(llm, "test-model")

	plan, err := p.Decompose(context.Background(), "deploy service", nil)
	if err != nil {
		t.Fatalf("Decompose failed: %v", err)
	}

	if plan.Goal != "deploy service" {
		t.Errorf("goal = %q, want %q", plan.Goal, "deploy service")
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(plan.Steps))
	}
	if plan.Steps[0].Description != "Build Docker image" {
		t.Errorf("step 1 desc = %q", plan.Steps[0].Description)
	}
	if plan.Steps[1].Description != "Push to registry" {
		t.Errorf("step 2 desc = %q", plan.Steps[1].Description)
	}
}

func TestDecomposeWithMarkdownCodeBlock(t *testing.T) {
	llm := &mockLLM{response: "Here's the plan:\n```json\n{\"steps\":[{\"description\":\"step 1\",\"tool_hints\":[\"exec\"],\"expected_outcome\":\"done\"}]}\n```\n"}
	p := NewPlanner(llm, "test")

	plan, err := p.Decompose(context.Background(), "test goal", nil)
	if err != nil {
		t.Fatalf("Decompose failed: %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(plan.Steps))
	}
}

func TestDecomposeInvalidJSON(t *testing.T) {
	llm := &mockLLM{response: "I can't do that"}
	p := NewPlanner(llm, "test")

	plan, err := p.Decompose(context.Background(), "test goal", nil)
	if err != nil {
		t.Fatalf("Decompose failed: %v", err)
	}
	// Should fallback to single-step plan
	if len(plan.Steps) != 1 {
		t.Fatalf("steps = %d, want 1 (fallback)", len(plan.Steps))
	}
	if plan.Steps[0].Description != "test goal" {
		t.Errorf("fallback step desc = %q, want %q", plan.Steps[0].Description, "test goal")
	}
}

func TestDecomposeLLMError(t *testing.T) {
	llm := &mockLLM{err: context.DeadlineExceeded}
	p := NewPlanner(llm, "test")

	plan, err := p.Decompose(context.Background(), "test goal", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	// Should still return a fallback plan
	if len(plan.Steps) != 1 {
		t.Errorf("steps = %d, want 1 (fallback)", len(plan.Steps))
	}
}

func TestDecomposeMaxSteps(t *testing.T) {
	// Generate 15 steps
	planResp := planResponse{Steps: make([]planStepJSON, 15)}
	for i := range planResp.Steps {
		planResp.Steps[i] = planStepJSON{
			Description: "step",
			ExpectedOut: "done",
		}
	}
	respJSON, _ := json.Marshal(planResp)

	llm := &mockLLM{response: string(respJSON)}
	p := NewPlanner(llm, "test")

	plan, _ := p.Decompose(context.Background(), "test", nil)
	if len(plan.Steps) > 10 {
		t.Errorf("steps = %d, want max 10", len(plan.Steps))
	}
}

func TestDecomposeWithToolDefs(t *testing.T) {
	llm := &mockLLM{response: `{"steps":[{"description":"search","tool_hints":["web_search"],"expected_out":"found"}]}`}
	p := NewPlanner(llm, "test")

	toolDefs := []tools.ToolDefinition{
		{Name: "web_search", Description: "Search the web"},
		{Name: "file_read", Description: "Read files"},
	}

	plan, err := p.Decompose(context.Background(), "find info", toolDefs)
	if err != nil {
		t.Fatalf("Decompose failed: %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(plan.Steps))
	}
}

func TestReviseSuccess(t *testing.T) {
	// First create a plan
	llm := &mockLLM{response: `{"steps":[{"description":"build","expected_out":"done"},{"description":"push","expected_out":"done"}]}`}
	p := NewPlanner(llm, "test")

	plan, _ := p.Decompose(context.Background(), "deploy", nil)
	plan.MarkRunning()
	plan.Steps[0].Status = StepDone
	plan.Steps[0].Result = "built"

	// Now revise - LLM returns revised steps
	llm.response = `{"steps":[{"description":"push to alt registry","expected_out":"pushed"}]}`

	revised, err := p.Revise(context.Background(), plan, "step-2", "permission denied")
	if err != nil {
		t.Fatalf("Revise failed: %v", err)
	}

	if revised.Revision != 1 {
		t.Errorf("revision = %d, want 1", revised.Revision)
	}
	// First step should be preserved
	if revised.Steps[0].Status != StepDone {
		t.Errorf("step 1 status = %q, want %q", revised.Steps[0].Status, StepDone)
	}
}

func TestReviseMaxRevisions(t *testing.T) {
	llm := &mockLLM{response: `{"steps":[{"description":"step","expected_out":"done"}]}`}
	p := NewPlanner(llm, "test")

	plan, _ := p.Decompose(context.Background(), "test", nil)
	plan.MarkRunning()
	plan.Revision = 3 // already at max

	revised, err := p.Revise(context.Background(), plan, "step-1", "failed")
	if err != nil {
		t.Fatalf("Revise failed: %v", err)
	}

	if revised.Status != PlanFailed {
		t.Errorf("status = %q, want %q", revised.Status, PlanFailed)
	}
}

func TestShouldContinue(t *testing.T) {
	p := NewPlanner(&mockLLM{}, "test")

	// Plan with pending steps
	plan := NewPlan("test")
	plan.AddStep("a", "done", nil)
	if !p.ShouldContinue(plan) {
		t.Error("should continue with pending steps")
	}

	// Plan complete
	plan.Steps[0].Status = StepDone
	if p.ShouldContinue(plan) {
		t.Error("should not continue when complete")
	}

	// Plan failed
	plan2 := NewPlan("test")
	plan2.MarkFailed("error")
	if p.ShouldContinue(plan2) {
		t.Error("should not continue when failed")
	}

	// Max revisions reached
	plan3 := NewPlan("test")
	plan3.AddStep("a", "done", nil)
	plan3.Revision = 3
	if p.ShouldContinue(plan3) {
		t.Error("should not continue at max revisions")
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain JSON", `{"steps":[]}`, `{"steps":[]}`},
		{"code block", "```json\n{\"steps\":[]}\n```", `{"steps":[]}`},
		{"mixed", "Here:\n```json\n{\"steps\":[]}\n```\nDone.", `{"steps":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}
