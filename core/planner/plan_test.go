package planner

import (
	"encoding/json"
	"testing"
)

func TestNewPlan(t *testing.T) {
	p := NewPlan("deploy service")
	if p.Goal != "deploy service" {
		t.Errorf("goal = %q, want %q", p.Goal, "deploy service")
	}
	if p.Status != PlanPending {
		t.Errorf("status = %q, want %q", p.Status, PlanPending)
	}
	if len(p.Steps) != 0 {
		t.Errorf("steps = %d, want 0", len(p.Steps))
	}
}

func TestAddStep(t *testing.T) {
	p := NewPlan("test")
	step := p.AddStep("build image", "image built", []string{"exec"})
	if step.ID != "step-1" {
		t.Errorf("step ID = %q, want %q", step.ID, "step-1")
	}
	if step.Description != "build image" {
		t.Errorf("description = %q, want %q", step.Description, "build image")
	}
	if step.Status != StepPending {
		t.Errorf("status = %q, want %q", step.Status, StepPending)
	}
	if len(p.Steps) != 1 {
		t.Errorf("steps = %d, want 1", len(p.Steps))
	}
}

func TestAddMultipleSteps(t *testing.T) {
	p := NewPlan("test")
	p.AddStep("step 1", "done 1", nil)
	p.AddStep("step 2", "done 2", nil)
	p.AddStep("step 3", "done 3", nil)

	if len(p.Steps) != 3 {
		t.Errorf("steps = %d, want 3", len(p.Steps))
	}
	if p.Steps[0].ID != "step-1" || p.Steps[1].ID != "step-2" || p.Steps[2].ID != "step-3" {
		t.Error("step IDs not sequential")
	}
}

func TestGetStep(t *testing.T) {
	p := NewPlan("test")
	p.AddStep("first", "done", nil)
	p.AddStep("second", "done", nil)

	step := p.GetStep("step-2")
	if step == nil {
		t.Fatal("expected step-2 to exist")
	}
	if step.Description != "second" {
		t.Errorf("description = %q, want %q", step.Description, "second")
	}

	if p.GetStep("step-99") != nil {
		t.Error("expected nil for non-existent step")
	}
}

func TestNextPendingStep(t *testing.T) {
	p := NewPlan("test")
	p.AddStep("first", "done", nil)
	p.AddStep("second", "done", nil)

	step := p.NextPendingStep()
	if step == nil {
		t.Fatal("expected pending step")
	}
	if step.ID != "step-1" {
		t.Errorf("expected step-1, got %q", step.ID)
	}

	// Mark first step as done
	p.Steps[0].Status = StepDone
	step = p.NextPendingStep()
	if step == nil || step.ID != "step-2" {
		t.Error("expected step-2 as next pending")
	}

	// All done
	p.Steps[1].Status = StepDone
	if p.NextPendingStep() != nil {
		t.Error("expected nil when all steps done")
	}
}

func TestIsComplete(t *testing.T) {
	p := NewPlan("test")
	p.AddStep("a", "done", nil)
	p.AddStep("b", "done", nil)

	if p.IsComplete() {
		t.Error("expected not complete with pending steps")
	}

	p.Steps[0].Status = StepDone
	if p.IsComplete() {
		t.Error("expected not complete with one pending step")
	}

	p.Steps[1].Status = StepDone
	if !p.IsComplete() {
		t.Error("expected complete when all steps done")
	}
}

func TestIsCompleteWithSkipped(t *testing.T) {
	p := NewPlan("test")
	p.AddStep("a", "done", nil)
	p.AddStep("b", "done", nil)

	p.Steps[0].Status = StepDone
	p.Steps[1].Status = StepSkipped
	if !p.IsComplete() {
		t.Error("expected complete with done + skipped")
	}
}

func TestHasFailed(t *testing.T) {
	p := NewPlan("test")
	p.AddStep("a", "done", nil)

	if p.HasFailed() {
		t.Error("expected no failure initially")
	}

	p.Steps[0].Status = StepFailed
	if !p.HasFailed() {
		t.Error("expected failure after step failed")
	}
}

func TestMarkRunning(t *testing.T) {
	p := NewPlan("test")
	p.MarkRunning()
	if p.Status != PlanRunning {
		t.Errorf("status = %q, want %q", p.Status, PlanRunning)
	}
}

func TestMarkComplete(t *testing.T) {
	p := NewPlan("test")
	p.MarkComplete()
	if p.Status != PlanComplete {
		t.Errorf("status = %q, want %q", p.Status, PlanComplete)
	}
}

func TestMarkFailed(t *testing.T) {
	p := NewPlan("test")
	p.MarkFailed("error occurred")
	if p.Status != PlanFailed {
		t.Errorf("status = %q, want %q", p.Status, PlanFailed)
	}
}

func TestIncrementRevision(t *testing.T) {
	p := NewPlan("test")
	p.IncrementRevision()
	if p.Revision != 1 {
		t.Errorf("revision = %d, want 1", p.Revision)
	}
	if p.Status != PlanRevised {
		t.Errorf("status = %q, want %q", p.Status, PlanRevised)
	}

	p.IncrementRevision()
	if p.Revision != 2 {
		t.Errorf("revision = %d, want 2", p.Revision)
	}
}

func TestProgress(t *testing.T) {
	p := NewPlan("test")
	p.AddStep("a", "done", nil)
	p.AddStep("b", "done", nil)
	p.AddStep("c", "done", nil)

	if p.Progress() != "0/3 steps done" {
		t.Errorf("progress = %q, want %q", p.Progress(), "0/3 steps done")
	}

	p.Steps[0].Status = StepDone
	if p.Progress() != "1/3 steps done" {
		t.Errorf("progress = %q, want %q", p.Progress(), "1/3 steps done")
	}
}

func TestJSONSerialization(t *testing.T) {
	p := NewPlan("deploy service")
	p.AddStep("build", "image built", []string{"exec"})
	p.Steps[0].Status = StepDone
	p.Steps[0].Result = "built golem:v0.6.0"

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var p2 Plan
	if err := json.Unmarshal(data, &p2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if p2.Goal != p.Goal {
		t.Errorf("goal mismatch: %q vs %q", p2.Goal, p.Goal)
	}
	if len(p2.Steps) != 1 {
		t.Errorf("steps = %d, want 1", len(p2.Steps))
	}
	if p2.Steps[0].Result != "built golem:v0.6.0" {
		t.Errorf("result = %q, want %q", p2.Steps[0].Result, "built golem:v0.6.0")
	}
}
