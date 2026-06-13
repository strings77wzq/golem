package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/tools"
)

// Planner decomposes user goals into structured plans and manages
// their execution lifecycle.
type Planner struct {
	llm          providers.LLMProvider
	model        string
	maxSteps     int
	maxRevisions int
}

// NewPlanner creates a planner with the given LLM provider.
func NewPlanner(llm providers.LLMProvider, model string) *Planner {
	return &Planner{
		llm:          llm,
		model:        model,
		maxSteps:     10,
		maxRevisions: 3,
	}
}

// Decompose breaks a user goal into a structured plan.
// Falls back to a single-step plan if LLM output is invalid.
func (p *Planner) Decompose(ctx context.Context, goal string, toolDefs []tools.ToolDefinition) (*Plan, error) {
	// Build tool list for prompt
	toolNames := make([]string, 0, len(toolDefs))
	for _, td := range toolDefs {
		toolNames = append(toolNames, td.Name)
	}

	prompt := fmt.Sprintf(`Given the user goal: "%s"

Available tools: %s

Break this goal into 2-5 sequential steps. Each step should:
- Have a clear description of what to do
- List which tools might be needed (tool_hints)
- Describe what success looks like (expected_outcome)

Output ONLY a JSON object with this exact format:
{
  "steps": [
    {
      "description": "what to do",
      "tool_hints": ["tool1", "tool2"],
      "expected_outcome": "what success looks like"
    }
  ]
}

Do not include any other text, just the JSON.`, goal, strings.Join(toolNames, ", "))

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: prompt},
	}

	resp, err := p.llm.Chat(ctx, messages, nil, p.model, nil)
	if err != nil {
		return p.fallbackPlan(goal), fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse the plan from LLM response
	plan, err := parsePlanResponse(resp.Content, goal)
	if err != nil {
		// Fallback to single-step plan
		return p.fallbackPlan(goal), nil
	}

	// Enforce limits
	if len(plan.Steps) > p.maxSteps {
		plan.Steps = plan.Steps[:p.maxSteps]
	}

	return plan, nil
}

// Revise updates a plan based on a failed step's result.
func (p *Planner) Revise(ctx context.Context, plan *Plan, failedStepID string, stepResult string) (*Plan, error) {
	if plan.Revision >= p.maxRevisions {
		plan.MarkFailed(fmt.Sprintf("max revisions (%d) reached", p.maxRevisions))
		return plan, nil
	}

	// Build context for revision
	var stepsDesc strings.Builder
	for _, step := range plan.Steps {
		status := string(step.Status)
		if step.ID == failedStepID {
			status = "FAILED"
		}
		stepsDesc.WriteString(fmt.Sprintf("- [%s] %s (expected: %s)\n", status, step.Description, step.ExpectedOut))
	}

	prompt := fmt.Sprintf(`Original goal: "%s"

Current plan status:
%s

Step "%s" failed with result: "%s"

Revise the remaining (pending) steps. Keep completed steps as-is.
Output ONLY a JSON object with this format:
{
  "steps": [
    {"description": "...", "tool_hints": [...], "expected_outcome": "..."}
  ]
}

Only include PENDING steps in your response. Do not include completed or failed steps.`, plan.Goal, stepsDesc.String(), failedStepID, stepResult)

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: prompt},
	}

	resp, err := p.llm.Chat(ctx, messages, nil, p.model, nil)
	if err != nil {
		plan.IncrementRevision()
		return plan, fmt.Errorf("LLM revision call failed: %w", err)
	}

	// Parse revised steps
	revisedPlan, err := parsePlanResponse(resp.Content, plan.Goal)
	if err != nil {
		plan.IncrementRevision()
		return plan, nil
	}

	// Keep completed/failed steps, replace pending with revised
	newSteps := make([]Step, 0)
	for _, step := range plan.Steps {
		if step.Status == StepDone || step.Status == StepSkipped || step.ID == failedStepID {
			newSteps = append(newSteps, step)
		}
	}
	// Add revised steps
	for _, rs := range revisedPlan.Steps {
		rs.ID = fmt.Sprintf("step-%d", len(newSteps)+1)
		rs.Status = StepPending
		newSteps = append(newSteps, rs)
	}

	plan.Steps = newSteps
	plan.IncrementRevision()

	return plan, nil
}

// ShouldContinue decides whether to keep executing the plan.
func (p *Planner) ShouldContinue(plan *Plan) bool {
	if plan.Status == PlanFailed {
		return false
	}
	if plan.Status == PlanComplete {
		return false
	}
	if plan.Revision >= p.maxRevisions {
		return false
	}
	if plan.IsComplete() {
		return false
	}
	return true
}

// fallbackPlan creates a single-step plan when decomposition fails.
func (p *Planner) fallbackPlan(goal string) *Plan {
	plan := NewPlan(goal)
	plan.AddStep(goal, "task completed", nil)
	return plan
}

// planResponse is the JSON structure returned by the LLM.
type planResponse struct {
	Steps []planStepJSON `json:"steps"`
}

type planStepJSON struct {
	Description string   `json:"description"`
	ToolHints   []string `json:"tool_hints"`
	ExpectedOut string   `json:"expected_outcome"`
}

// parsePlanResponse parses the LLM's JSON response into a Plan.
func parsePlanResponse(content, goal string) (*Plan, error) {
	// Extract JSON from response (may be wrapped in markdown code blocks)
	jsonStr := extractJSON(content)

	var resp planResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if len(resp.Steps) == 0 {
		return nil, fmt.Errorf("no steps in response")
	}

	plan := NewPlan(goal)
	for _, s := range resp.Steps {
		plan.AddStep(s.Description, s.ExpectedOut, s.ToolHints)
	}

	return plan, nil
}

// extractJSON extracts JSON from a string that may contain markdown code blocks.
func extractJSON(s string) string {
	// Try to find JSON in code block
	if idx := strings.Index(s, "```json"); idx != -1 {
		start := idx + 7
		if end := strings.Index(s[start:], "```"); end != -1 {
			return strings.TrimSpace(s[start : start+end])
		}
	}
	if idx := strings.Index(s, "```"); idx != -1 {
		start := idx + 3
		if end := strings.Index(s[start:], "```"); end != -1 {
			return strings.TrimSpace(s[start : start+end])
		}
	}

	// Try to find JSON object directly
	if start := strings.Index(s, "{"); start != -1 {
		if end := strings.LastIndex(s, "}"); end != -1 {
			return s[start : end+1]
		}
	}

	return s
}
