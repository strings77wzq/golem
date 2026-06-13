# Tasks: Planning Layer & Agent Enhancement

## Phase 1: Core Planner (T1-T4)

### T1: Plan data model
- Create `core/planner/plan.go` with Plan, Step, PlanStatus, StepStatus types
- Add JSON serialization tags
- Add Plan.AddStep(), Plan.GetStep(), Plan.IsComplete() methods
- Test: `core/planner/plan_test.go`

### T2: Planner.Decompose
- Create `core/planner/planner.go` with Planner struct
- Implement Decompose: sends prompt to LLM, parses JSON response
- Implement fallback: if JSON parse fails, create single-step plan
- Test: `core/planner/planner_test.go` with mock LLM

### T3: Planner.Revise
- Implement Revise: sends current plan + step result to LLM
- Parse revised plan, keep completed steps unchanged
- Safety limit: max 3 revisions per plan
- Test: revision with mock LLM

### T4: Planner.ShouldContinue
- Implement termination logic
- Check: all steps done, max revisions reached, plan status
- Test: various plan states

## Phase 2: Tool Selector (T5-T6)

### T5: ToolSelector
- Create `core/agent/tool_selector.go`
- Implement keyword-based relevance scoring
- Implement Select() method
- Test: `core/agent/tool_selector_test.go`

### T6: Keyword extraction
- Implement stop word removal and tokenization
- Test: various input strings

## Phase 3: Reflector (T7-T8)

### T7: Reflector
- Create `core/agent/reflector.go`
- Implement heuristic evaluation (keyword matching, error check)
- Test: `core/agent/reflector_test.go`

### T8: LLM escalation
- Implement fallback to LLM evaluation when heuristic is uncertain
- Test: uncertain cases

## Phase 4: Agent Integration (T9-T12)

### T9: Enhanced agent loop
- Modify `core/agent/loop.go` to support planning mode
- Add complexity detection
- Add mini ReAct loop per step
- Test: planning mode with mock provider

### T10: Agent configuration
- Add PlanningConfig to config.go
- Add --plan CLI flag
- Test: config loading

### T11: Progress display
- Emit progress messages during plan execution
- Format: [Plan] Step 1/4: ..., [Step 1] ✓ Done, etc.
- Test: message emission

### T12: Session integration
- Add ActivePlan to Session struct
- Store/retrieve plan with session
- Test: session persistence

## Phase 5: Polish (T13-T14)

### T13: Documentation
- Update README with --plan flag
- Add planning section to architecture docs
- Update AGENTS.md

### T14: Integration test
- End-to-end test: user request → plan → execute → verify
- Test with mock provider
