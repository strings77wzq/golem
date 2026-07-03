## Why

A CEO-review code audit found three Critical defects that silently contradict Golem's "safety-first" positioning. Two are injection vulnerabilities reachable from LLM tool output, one causes plan-path errors to be swallowed and surfaced as successful answers. These exist outside the in-flight `compliance-fix` spec (layer violations) and are orthogonal to it, so they warrant their own change for independent threat modeling, review, and rollback. Safety claims are only credible if the safety-critical code paths are themselves safe.

## What Changes

- **F1 — SQL identifier injection in `sql_analyze`**: `core/tools/database/sql_analyze.go:48` interpolates the LLM-supplied `table` argument directly into `SELECT COUNT(*) FROM %s`. Add identifier allowlist validation against the live schema before execution; reject any identifier not in the schema.
- **F2 — Rollback SQL generator is injectable and emits malformed SQL in the live path**: `core/security/gates.go` `GenerateDeleteRollback` and `GenerateUpdateRollback` build SQL via `fmt.Sprintf` with unescaped identifiers/values. These generators are called from `core/tools/database/sql_query.go:184,186` on every DELETE/UPDATE, and `GenerateUpdateRollback` is invoked with `oldValues=nil` today (line 186) — so every UPDATE produces `UPDATE tbl SET  WHERE …` (empty SET clause), malformed SQL shipped as rollback. Replace with validators that quote identifiers/bind values and refuse to emit malformed rollback SQL. **BREAKING** to the two in-tree call sites (updated in this change) and any external caller.
- **F3 — Plan step errors swallowed and surfaced as success**: `core/agent/loop.go` `executeStep` returns errors as the success string `"Error: %v"`; `processWithPlan` then calls `reflector.Evaluate(step, stepResult, nil)` with a hard-coded `nil` error. Propagate the real `error` through `executeStep`'s return and into `Reflector.Evaluate` so failures mark the step failed and trigger revise/abort instead of advancing the plan.
- Regression tests for each defect (injection blocked, rollback SQL valid on quoted values, plan aborts on provider/LLM error).

## Capabilities

### New Capabilities

- `sql-identifier-guard`: validate SQL identifiers (table names) against the live schema before they are interpolated into queries, returning a typed reject error for out-of-schema identifiers.
- `safe-rollback-generation`: generate valid, escapable rollback SQL for DELETE and UPDATE with identifier validation and value quoting, never producing broken SQL when values contain single quotes.

### Modified Capabilities

- `agent`: plan-step execution must propagate provider/LLM errors to the reflector so a failed step stops or revises the plan instead of returning an `Error: …` string as the step result and final answer.

## Impact

- **Code**: `core/tools/database/sql_analyze.go` (F1), `core/security/gates.go` + tests (F2), `core/agent/loop.go` `executeStep`/`processWithPlan` + `core/agent/reflector.go` consumers (F3), and new `*_test.go` files for each.
- **APIs**: F2 changes `GenerateDeleteRollback`/`GenerateUpdateRollback` semantics (safer output, parameter binding) — **BREAKING** for any external caller; no in-tree callers found. `executeStep` gains an `error` return; `processWithPlan` consumes it.
- **Dependencies**: None new. Pure Go, `CGO_ENABLED=0` preserved. `LLMProvider`/`StreamingProvider` interfaces untouched per AGENTS.md §4.
- **Systems**: No runtime/database-format changes. Existing rollback consumers (if any) must re-validate.
- **Risk**: F2 is the highest risk (security primitive changing shape); gate behind existing tests plus new injection vectors. F3 touches the plan loop's control flow — must not regress the happy-path E2E.