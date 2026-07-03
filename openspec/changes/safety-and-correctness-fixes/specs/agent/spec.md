# agent — safety-and-correctness-fixes delta

## ADDED Requirements

### Requirement: Plan step execution MUST propagate provider/LLM errors to the reflector

When running the planned ReAct loop, `executeStep` MUST return `(string, error)` and surface any provider-resolution or LLM-call error through that error channel. `processWithPlan` MUST pass that real error into `Reflector.Evaluate(step, result, err)`. A step whose provider or LLM call failed MUST be marked `StepFailed`, MAY trigger a plan revision, and the plan MUST NOT advance as if the step produced successful content. The plan loop MUST NOT return an error-string-formatted result (e.g. `"Error: …"`) as a successful final answer; when all steps fail, `processWithPlan` MUST surface the failure distinctly from a normal answer.

#### Scenario: Provider is unavailable during a plan step

- **WHEN** `executeStep` calls the provider and the provider returns an error (no reachable model)
- **THEN** `executeStep` returns a non-nil `error`, `processWithPlan` sets the step status to `StepFailed`, calls `Reflector.Evaluate` with that error, and the step's result is NOT used as if it were successful content

#### Scenario: LLM call inside a plan step errors

- **WHEN** the LLM call within `executeStep` returns an error
- **THEN** `executeStep` returns a non-nil `error` and `lastContent` for that step is not contaminated with an `"Error: …"` string that gets forwarded as the plan's final answer

#### Scenario: Happy path is unchanged

- **WHEN** all plan steps execute with no errors
- **THEN** the plan loop behaves exactly as before this change: steps succeed, reflection succeeds, the final response is the last step's content, and no spurious error branching is triggered

#### Scenario: Every step in the plan fails

- **WHEN** every step in a plan returns a non-nil error
- **THEN** `processWithPlan` surfaces a failure to the caller distinct from a normal answer (the response is not `"Error: …"`) and the emit progress reports the failure rather than implying success