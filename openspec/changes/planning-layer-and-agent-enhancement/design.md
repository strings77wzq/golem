# Design: Planning Layer & Agent Enhancement

## Context

Golem's current agent loop is a flat ReAct cycle:

```
user message → [system prompt + history + tools] → LLM → tool call → execute → LLM → ... → response
```

This works for simple tasks ("what's the weather?") but fails for complex tasks ("deploy this service to production") because:

1. The LLM has no structured plan — it improvises each step
2. There's no goal tracking — the agent doesn't know when it's "done"
3. No error recovery — if a tool fails, the agent just tries again or gives up
4. Tool selection is wasteful — all tools are sent to the LLM every turn

## Architecture

### Planning Layer

```
User: "deploy the golem service to production"
  │
  ▼
┌─────────────────────────────────┐
│ PLANNER                         │
│                                 │
│ 1. Decompose into plan:         │
│    Step 1: Build Docker image   │
│    Step 2: Push to registry     │
│    Step 3: Apply K8s manifests  │
│    Step 4: Verify rollout       │
│                                 │
│ 2. For each step:               │
│    - Select relevant tools      │
│    - Execute via ReAct loop     │
│    - Check result               │
│    - Revise plan if needed      │
│                                 │
│ 3. When all steps complete:     │
│    - Compose final response     │
└─────────────────────────────────┘
```

### Data Flow

```
User Input
  │
  ▼
┌──────────────┐
│ Context Mgr  │ → system prompt + token budget
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Planner    │ → plan: []Step
└──────┬───────┘
       │
       ▼ (for each step)
┌──────────────┐
│ Tool Selector│ → subset of tools for this step
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  ReAct Loop  │ → execute tools (max 5 iterations per step)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Reflector   │ → did step succeed? revise plan?
└──────┬───────┘
       │
       ▼ (next step or final response)
```

## Key Interfaces

```go
// core/planner/plan.go

type Plan struct {
    ID        string      `json:"id"`
    Goal      string      `json:"goal"`
    Steps     []Step      `json:"steps"`
    Status    PlanStatus  `json:"status"`
    CreatedAt time.Time   `json:"created_at"`
}

type Step struct {
    ID          string       `json:"id"`
    Description string       `json:"description"`
    ToolHints   []string     `json:"tool_hints,omitempty"`
    ExpectedOut string       `json:"expected_outcome"`
    Status      StepStatus   `json:"status"`
    Result      string       `json:"result,omitempty"`
    Error       string       `json:"error,omitempty"`
}

type PlanStatus string
const (
    PlanPending   PlanStatus = "pending"
    PlanRunning   PlanStatus = "running"
    PlanComplete  PlanStatus = "complete"
    PlanFailed    PlanStatus = "failed"
    PlanRevised   PlanStatus = "revised"
)

type StepStatus string
const (
    StepPending  StepStatus = "pending"
    StepRunning  StepStatus = "running"
    StepDone     StepStatus = "done"
    StepFailed   StepStatus = "failed"
    StepSkipped  StepStatus = "skipped"
)

// core/planner/planner.go

type Planner struct {
    llm       providers.LLMProvider
    budget    *context.TokenBudget
    maxSteps  int
}

// Decompose breaks a user request into a structured plan.
func (p *Planner) Decompose(ctx context.Context, goal string, tools []tools.ToolDefinition) (*Plan, error)

// Revise updates a plan based on step results.
func (p *Planner) Revise(ctx context.Context, plan *Plan, stepResult string) (*Plan, error)

// ShouldContinue decides whether to continue executing or stop.
func (p *Planner) ShouldContinue(plan *Plan) bool

// core/agent/enhanced_loop.go (modifications to existing loop)

// Enhanced agent loop that uses planner:
func (a *Agent) processMessageWithPlan(ctx, msg, streamFinal, onToken, emit) {
    // 1. Build context
    // 2. Create plan via planner.Decompose()
    // 3. For each step:
    //    a. Select tools via toolSelector.Select(step)
    //    b. Run mini ReAct loop (max 5 iterations)
    //    c. Reflector checks result
    //    d. If failed, planner.Revise()
    // 4. Compose final response
}

// core/agent/tool_selector.go

type ToolSelector struct {
    registry *tools.Registry
}

// Select returns the most relevant tools for a given step.
func (ts *ToolSelector) Select(step *planner.Step, allTools []tools.ToolDefinition, maxTools int) []tools.ToolDefinition
```

## Decision Points

### D1. Plan Generation Strategy

**Option A: LLM generates plan** — Ask the LLM to output a JSON plan with steps.
- Pro: Flexible, handles arbitrary tasks
- Con: Adds latency, LLM may generate invalid JSON

**Option B: Rule-based decomposition** — Use templates for common task patterns.
- Pro: Fast, reliable
- Con: Rigid, can't handle novel tasks

**Decision: Option A with fallback.** LLM generates plan first. If JSON parsing fails, fall back to a single-step plan (the whole request as one step). This gives flexibility while maintaining reliability.

### D2. Tool Selection Strategy

**Option A: Embedding similarity** — Embed tool descriptions and step, pick top-K.
- Pro: Semantic matching
- Con: Requires embedding model, adds dependency

**Option B: Keyword matching** — Match step description against tool names/descriptions.
- Pro: Zero dependencies, fast
- Con: Less accurate

**Option C: LLM-based selection** — Ask the LLM which tools are relevant.
- Pro: Most accurate
- Con: Extra LLM call per step

**Decision: Option B for now.** Keyword matching is sufficient for v1. Can upgrade to Option C later.

### D3. Reflection Strategy

**Option A: LLM self-evaluation** — Ask the LLM "did this step achieve its goal?"
- Pro: Nuanced understanding
- Con: Extra LLM call

**Option B: Output comparison** — Compare step output against expected outcome using heuristics.
- Pro: Fast, no extra LLM calls
- Con: May miss subtle failures

**Decision: Option B with escalation.** Use heuristic comparison first. If uncertain, escalate to LLM evaluation.

## Risks

- **Latency**: Planning adds 1-2 extra LLM calls before execution starts. Mitigation: planning is only used for complex tasks (detected by intent analysis).
- **Plan validity**: LLM may generate invalid plans. Mitigation: schema validation + fallback to single-step.
- **Over-engineering**: Simple tasks don't need planning. Mitigation: skip planning for single-intent requests (detected by keyword/complexity analysis).

## Success Criteria

1. Agent can decompose "deploy this service" into [build, push, apply, verify]
2. Agent can recover from step failures by revising the plan
3. Tool selection reduces token usage by 30%+ vs sending all tools
4. All existing tests pass, new tests cover planner and tool selector
5. No new external dependencies
