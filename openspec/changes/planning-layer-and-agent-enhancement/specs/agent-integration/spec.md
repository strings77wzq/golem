# Spec: Agent Integration

## [S1] Enhanced Agent Loop

The agent loop is modified to support planning mode:

**Current flow:**
```
processMessage → historyManager.GetContextWindow → invokeProvider → tool calls → repeat
```

**New flow:**
```
processMessage → 
  if planningEnabled AND isComplexTask:
    planner.Decompose → for each step: toolSelector.Select → mini ReAct → reflector.Evaluate
  else:
    existing ReAct loop (unchanged)
```

**Complexity detection:**
A task is "complex" if ANY of:
- Message contains multiple action verbs (deploy, build, test, check)
- Message length > 100 chars
- Message contains keywords: "plan", "step by step", "first", "then", "finally"
- Message contains "deploy", "migrate", "setup", "configure"

Otherwise, use simple ReAct loop (no planning overhead).

## [S2] Agent Configuration

New fields in `config.Config`:

```go
type AgentConfig struct {
    Defaults AgentDefaults `yaml:"defaults"`
    Planning PlanningConfig `yaml:"planning"`
}

type PlanningConfig struct {
    Enabled     bool `yaml:"enabled"`
    MaxSteps    int  `yaml:"max_steps"`     // default 10
    MaxRevisions int `yaml:"max_revisions"` // default 3
    MaxToolsPerStep int `yaml:"max_tools_per_step"` // default 8
}
```

**Default:** Planning disabled. Users must opt in via config or CLI flag.

## [S3] CLI Flag

New flag on `golem agent`:
```
--plan    Enable planning mode for complex tasks
```

When `--plan` is set, the agent enables planning for all messages.

## [S4] Progress Display

When planning is active, the agent emits progress updates:

```
[Planning] Decomposing task into steps...
[Plan] Step 1/4: Build Docker image
[Step 1] Executing: docker build...
[Step 1] ✓ Done: built golem:v0.6.0
[Plan] Step 2/4: Push to registry
[Step 2] Executing: docker push...
[Step 2] ✗ Failed: permission denied
[Plan] Revising plan...
[Plan] Step 2/4 (revised): Request registry access
...
```

These are emitted as `OutboundMessage` with `Role: bus.RoleTool` and `Done: false`.

## [S5] Session Integration

Plans are stored in the session for resumption:

```go
type Session struct {
    // ... existing fields
    ActivePlan *planner.Plan `json:"active_plan,omitempty"`
}
```

When a session is resumed, the agent can continue executing the plan from where it left off.

## [S6] Backward Compatibility

- Planning is OFF by default
- Existing `golem agent` behavior is unchanged when `--plan` is not set
- All existing tests pass without modification
- The simple ReAct loop remains as fallback for non-planning mode
