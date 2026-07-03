## Context

Golem markets itself as "safety-first": read-only default, WHERE enforcement, rollback SQL. A CEO-review code audit showed the safety-critical code paths betray that claim:

- `core/tools/database/sql_analyze.go:48` interpolates an LLM-supplied table name directly into SQL.
- `core/security/gates.go` rollback generators are wired into the live execute path (`core/tools/database/sql_query.go:184,186`) but emit injectable/malformed SQL; `GenerateUpdateRollback` is called with `oldValues=nil`, so every UPDATE gets `UPDATE tbl SET  WHERE …`.
- `core/agent/loop.go` `executeStep` returns provider/LLM errors as the success string `"Error: %v"`, and `processWithPlan` feeds `nil` as the error to `Reflector.Evaluate`, so a failed step is treated as success and the plan keeps advancing.

Current state: `SQLDriver` interface (`core/database/types.go:34`) exposes `Query`/`Execute`/`GetSchema`/`GetSchemaForTable`/`Ping`/`Close` — no `ListTables`. `Reflector.Evaluate(step, result, err)` already has an `err` slot that callers ignore. The driver `Query` accepts `args ...interface{}` (parameterized) but `sql_analyze.go` doesn't use it.

Constraints (AGENTS.md §4): `CGO_ENABLED=0`, `LLMProvider`/`StreamingProvider` interfaces frozen, Bubble Tea isolated, tools sorted alphabetically, streaming condition `canStream := ok && streamFinal && len(toolDefs) == 0`. Pure Go only.

## Goals / Non-Goals

**Goals:**
- F1: reject any SQL identifier not present in the live schema before interpolation, returning a typed error to the LLM, so an LLM cannot inject via the `table` argument.
- F2: rollback generators emit syntactically valid, safe SQL — identifiers validated/quoted, string values single-quote-escaped — and refuse to emit malformed rollback (e.g. UPDATE with no SET clauses). Delete and UPDATE rollback both produce usable SQL.
- F3: `executeStep` propagates a real `error`; `processWithPlan` passes it to `Reflector.Evaluate`; a failed step marks the step failed, triggers revise/abort, and never returns an `"Error: …"` string as the final answer.
- Regression tests pin each defect closed.

**Non-Goals:**
- Full SQL parsing/AST validation of arbitrary LLM SQL (only identifier allowlist guarding for the tool-controlled fetch path; `sql_query` still passes user SQL to the driver).
- Re-architecting the plan loop or reflector heuristics.
- Adding `ListTables` to the `SQLDriver` interface (use `GetSchemaForTable` as a presence probe to avoid touching the driver surface).
- The `loop.go` file-size split, panic-in-library (V3), or any compliance-fix-spec scope.

## Decisions

### D1 — F1 guard uses `GetSchemaForTable` as presence probe, not a new `ListTables` interface method
Rationale: adding a method to `SQLDriver` touches every driver (sqlite/postgres/mysql) and their mocks; `GetSchemaForTable(ctx, table)` already errors when a table is absent (proven by `database_ext_test.go:72 TestSQLiteDriver_GetSchemaForTable_NotFound`). Probe-then-query reuses the existing contract with zero interface churn.
- Validation helper: `validateTableIdentifier(ctx, driver, table) (string, error)` in `core/tools/database` (or a small `identifiers.go`), returning the validated identifier and an error otherwise.
- Helpers considered: (a) new `ListTables` method — rejected (interface churn); (b) parse `GetSchema` string — rejected (stringly-typed, fragile across 3 drivers). `GetSchemaForTable` probe chosen.

### D2 — Identifier quoting is allowlist+literal, not free-form escaping
Rationale: escaping arbitrary identifiers is error-prone; the safe choice is to reject anything not in the schema, then quote the known-good name. Quote with the driver-agnostic backtick/double-quote rule? **Decision: do NOT guess the dialect** — since we only ever emit SQL that is then bound to `driver.Query`/`Execute` with `?` placeholders (DELETE/UPDATE rollback is informational/audit), and the original SQL already ran against the right driver, we validate the identifier is a real table and pass the validated name through verbatim (lowercased schema-name match). Concretely: count-from-`%s` becomes `SELECT COUNT(*) … FROM <validated>` with the validated identifier guaranteed to be a schema table. For F2 rollback output, the rollback SQL is stored in `AuditEntry.RollbackSQL` for a human/operator to review/run — it must be valid in the dialect, so we DO apply dialect-aware quoting but only to identifiers we already validated as in-schema.

### D3 — F2 rollback helpers return `(string, error)` and refuse malformed output
Rationale: a rollback string that is silently malformed is worse than none (false confidence). New signatures:
- `GenerateDeleteRollback(table, where string, args []interface{}) (string, error)`
- `GenerateUpdateRollback(table, where string, oldValues map[string]interface{}) (string, error)`
Return `("", err)` when `oldValues` is empty (refuse empty SET), when identifiers contain characters outside a strict identifier charset (`[A-Za-z_][A-Za-z0-9_]*`), and when string values contain un-escapable content. String values are single-quote-escaped (`''`); the `setClause` parameter is REMOVED from `GenerateUpdateRollback` (it was redundant with `oldValues` and the source of the `nil`-map bug — the SET clause is rebuilt from `oldValues`). This is a **BREAKING** signature change to in-tree callers updated in this change.
- Alternative considered: keep signatures, just fix escaping internally — rejected because the `nil`-map / redundant-`setClause` design is the root cause; a bandaid preserves the bug class.

### D4 — F3: `executeStep` returns `(string, error)`, error threaded to reflector
Rationale: the bug is a contract violation (`Evaluate`'s `err` slot fed `nil`).
- `executeStep(...) (string, error)` instead of `() string`.
- `processWithPlan` calls `result, stepErr := a.executeStep(...)`; on `stepErr != nil`, set `step.Status = StepFailed`, `step.Error = stepErr.Error()`, emit failure progress, call `reflector.Evaluate(step, result, stepErr)` (real error), and `continue`/`plan.MarkComplete()` as today but WITHOUT treating the error string as a real result. The final answer must never be `"Error: …"`; when all steps errored, return a real error from `processWithPlan` (new: it gains returning the error or returns a user-visible failure message distinct from a normal answer).
- "Default: assume success (optimistic)" in `Reflector.Evaluate` (line 74) stays — it's a separate heuristic; F3 only fixes the error-channel.

## Risks / Trade-offs

- [F1 probe cost] `GetSchemaForTable` is called on every `sql_analyze`. → Mitigation: drivers already cache schema (SQLiteDriver has `schemaCache`/`schemaTTL`); probe is one cached lookup. Acceptable.
- [F2 breaking callers] `sql_query.go:184,186` must be updated in the same change. → Mitigation: update call sites in the same task; the UPDATE site already passes `nil`, expected to now pass real old-row values or refuse rollback for UPDATE until a SELECT-before-UPDATE is captured. **Open question Q1**: do we capture old values, or emit a "manual rollback required" note for UPDATE?
- [F2 rollback SQL for UPDATE is hard without old values] True useful UPDATE rollback needs the prior row snapshot. → Mitigation: phase this — emit valid, safe, human-readable rollback SQL (e.g. `-- UPDATE rollback: re-apply original SET values; <generated template>`) and document that automatic pre-snapshot is a follow-up. We refuse to emit a broken `UPDATE … SET  WHERE …`.
- [F3 control-flow change regressions] Threading error through the plan loop can change the happy-path E2E (`tests/e2e` agent+sql happy path). → Mitigation: only branch on `stepErr != nil`; happy path (`stepErr == nil`) is byte-identical to today. Add a unit test for the error path that today silently returns `"Error: …"`.
- [Dialect quoting] Quoting identifiers for sqlite (double-quote/brackets) vs mysql/postgres differs. → Mitigation: rollback SQL is destined for audit display and is run by an operator who knows the dialect; quote conservatively (double-quote, ANSI) and validate identifiers against the strict charset. Document the limitation.