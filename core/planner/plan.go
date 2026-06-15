// Package planner implements task decomposition and structured execution
// planning for the AI agent. It breaks complex user requests into ordered
// steps, executes each step via the ReAct loop, and evaluates results
// to decide whether to continue, revise, or stop.
package planner

import (
	"fmt"
	"time"
)

// PlanStatus represents the overall state of a plan.
type PlanStatus string

const (
	PlanPending  PlanStatus = "pending"
	PlanRunning  PlanStatus = "running"
	PlanComplete PlanStatus = "complete"
	PlanFailed   PlanStatus = "failed"
	PlanRevised  PlanStatus = "revised"
)

// StepStatus represents the state of a single step.
type StepStatus string

const (
	StepPending StepStatus = "pending"
	StepRunning StepStatus = "running"
	StepDone    StepStatus = "done"
	StepFailed  StepStatus = "failed"
	StepSkipped StepStatus = "skipped"
)

// Plan represents a decomposed task with ordered execution steps.
type Plan struct {
	ID        string     `json:"id"`
	Goal      string     `json:"goal"`
	Steps     []Step     `json:"steps"`
	Status    PlanStatus `json:"status"`
	Revision  int        `json:"revision"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Step represents a single execution step within a plan.
type Step struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	ToolHints   []string   `json:"tool_hints,omitempty"`
	ExpectedOut string     `json:"expected_outcome"`
	Status      StepStatus `json:"status"`
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// NewPlan creates a new plan with the given goal.
func NewPlan(goal string) *Plan {
	now := time.Now()
	return &Plan{
		ID:        generateID(),
		Goal:      goal,
		Steps:     []Step{},
		Status:    PlanPending,
		Revision:  0,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddStep adds a step to the plan and returns it.
func (p *Plan) AddStep(description, expectedOut string, toolHints []string) *Step {
	step := Step{
		ID:          fmt.Sprintf("step-%d", len(p.Steps)+1),
		Description: description,
		ToolHints:   toolHints,
		ExpectedOut: expectedOut,
		Status:      StepPending,
	}
	p.Steps = append(p.Steps, step)
	p.UpdatedAt = time.Now()
	return &p.Steps[len(p.Steps)-1]
}

// GetStep returns a step by ID. Returns nil if not found.
func (p *Plan) GetStep(id string) *Step {
	for i := range p.Steps {
		if p.Steps[i].ID == id {
			return &p.Steps[i]
		}
	}
	return nil
}

// NextPendingStep returns the next step with status pending, or nil if none.
func (p *Plan) NextPendingStep() *Step {
	for i := range p.Steps {
		if p.Steps[i].Status == StepPending {
			return &p.Steps[i]
		}
	}
	return nil
}

// IsComplete returns true if all steps are done or skipped.
func (p *Plan) IsComplete() bool {
	for _, step := range p.Steps {
		if step.Status != StepDone && step.Status != StepSkipped {
			return false
		}
	}
	return true
}

// HasFailed returns true if any step has failed.
func (p *Plan) HasFailed() bool {
	for _, step := range p.Steps {
		if step.Status == StepFailed {
			return true
		}
	}
	return false
}

// MarkRunning sets the plan status to running.
func (p *Plan) MarkRunning() {
	p.Status = PlanRunning
	p.UpdatedAt = time.Now()
}

// MarkComplete sets the plan status to complete.
func (p *Plan) MarkComplete() {
	p.Status = PlanComplete
	p.UpdatedAt = time.Now()
}

// MarkFailed sets the plan status to failed.
func (p *Plan) MarkFailed(reason string) {
	p.Status = PlanFailed
	p.UpdatedAt = time.Now()
}

// IncrementRevision increments the revision counter.
func (p *Plan) IncrementRevision() {
	p.Revision++
	p.Status = PlanRevised
	p.UpdatedAt = time.Now()
}

// DoneSteps returns the count of completed steps.
func (p *Plan) DoneSteps() int {
	count := 0
	for _, step := range p.Steps {
		if step.Status == StepDone || step.Status == StepSkipped {
			count++
		}
	}
	return count
}

// TotalSteps returns the total number of steps.
func (p *Plan) TotalSteps() int {
	return len(p.Steps)
}

// Progress returns a human-readable progress string.
func (p *Plan) Progress() string {
	return fmt.Sprintf("%d/%d steps done", p.DoneSteps(), p.TotalSteps())
}

func generateID() string {
	return fmt.Sprintf("plan-%d", time.Now().UnixNano())
}
