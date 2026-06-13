# Spec: Planner

## [S1] Plan Data Model

The `Plan` struct represents a decomposed task with ordered steps.

**Fields:**
- `ID` (string, UUID) — unique plan identifier
- `Goal` (string) — the original user request
- `Steps` ([]Step) — ordered list of execution steps
- `Status` (PlanStatus) — pending/running/complete/failed/revised
- `CreatedAt` (time.Time)

**Step fields:**
- `ID` (string) — step identifier (e.g., "step-1")
- `Description` (string) — human-readable step description
- `ToolHints` ([]string) — suggested tools for this step
- `ExpectedOut` (string) — what success looks like
- `Status` (StepStatus) — pending/running/done/failed/skipped
- `Result` (string) — actual output after execution
- `Error` (string) — error message if failed

**Constraints:**
- Steps must be ordered (sequential execution)
- Each step has exactly one of: Result (success) or Error (failure)
- Plan status transitions: pending → running → complete/failed/revised

## [S2] Plan Decomposition

The `Planner.Decompose` method generates a plan from a user goal.

**Input:** goal string, available tools
**Output:** *Plan, error

**Algorithm:**
1. Send prompt to LLM asking for JSON plan:
   ```
   Given the goal: "{goal}"
   And available tools: {tool names}
   Break this into 2-5 sequential steps.
   Output JSON: {"steps": [{"description": "...", "tool_hints": ["..."], "expected_outcome": "..."}]}
   ```
2. Parse LLM response as JSON
3. Validate: at least 1 step, at most 10 steps
4. If JSON parse fails: create single-step plan with entire goal as one step
5. Assign step IDs (step-1, step-2, ...)

**Error handling:**
- LLM returns invalid JSON → fallback to single-step plan
- LLM returns 0 steps → fallback to single-step plan
- LLM returns >10 steps → truncate to first 10

## [S3] Plan Revision

The `Planner.Revise` method updates a plan based on step results.

**Input:** current plan, step result string
**Output:** *Plan, error

**Algorithm:**
1. Send prompt to LLM:
   ```
   Original goal: {plan.Goal}
   Current plan: {plan.Steps}
   Step {stepID} result: {result}
   Revise the remaining steps if needed. Output JSON.
   ```
2. Parse revised plan
3. Keep completed steps unchanged
4. Update pending steps based on LLM's revision

## [S4] Plan Termination

The `Planner.ShouldContinue` method decides whether to keep executing.

**Returns false when:**
- All steps are done (status=done or skipped)
- Plan status is complete or failed
- Maximum steps reached (safety limit)

## [S5] Serialization

Plans must be serializable to JSON for:
- Session persistence (store plan in session)
- Debugging (log plan at each step)
- API exposure (gateway can return plan status)

JSON format:
```json
{
  "id": "plan-uuid",
  "goal": "deploy golem to production",
  "steps": [
    {"id": "step-1", "description": "Build Docker image", "status": "done", "result": "built golem:v0.6.0"},
    {"id": "step-2", "description": "Push to registry", "status": "pending"}
  ],
  "status": "running"
}
```
