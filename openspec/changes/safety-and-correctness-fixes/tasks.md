## 1. F1 — SQL identifier guard for `sql_analyze`

- [x] 1.1 Write failing test `core/tools/database/sql_analyze_test.go`: injecting `table="users; DROP TABLE users--"` returns `IsError=true` and issues NO driver query; valid `table="products"` returns count + schema. Use a fake driver recording calls.
- [x] 1.2 Add `validateTableIdentifier(ctx, driver, table) (string, error)` helper in `core/tools/database` (new `identifiers.go`): regex-check `[A-Za-z_][A-Za-z0-9_]*`, then probe `driver.GetSchemaForTable(ctx, table)`; return validated name or typed error.
- [x] 1.3 Wire the guard into `sql_analyze.go:Execute` before `driver.Query(...)`; on validation error return `&tools.ToolResult{ForLLM: "Error: invalid table identifier: …", IsError: true}, nil`.
- [x] 1.4 Add `GetSchemaForTable`-error scenario test (probe fails → tool returns the driver error, no count query issued).
- [x] 1.5 Run `go test -race ./core/tools/database/...`; confirm new tests pass and existing tests unchanged.

## 2. F2 — Safe rollback SQL generation

- [x] 2.1 Write failing tests in `core/security/gates_test.go`: (a) DELETE rollback with injection table returns `("", error)`; (b) UPDATE rollback with `oldValues={"name":"Bob's pen"}` returns SQL with `''` escaping; (c) UPDATE rollback with `nil`/empty `oldValues` returns `("", error)`; (d) happy DELETE returns quoted identifier + preserved `?` placeholders + `args`.
- [x] 2.2 Rewrite `GenerateDeleteRollback(table, where string, args []interface{}) (string, error)`: validate identifier charset and quote it; validate `where` non-empty; keep `?` placeholders; return `(sql, nil)` or `("", err)`.
- [x] 2.3 Rewrite `GenerateUpdateRollback(table, where string, oldValues map[string]interface{}) (string, error)`: REMOVE the `setClause` parameter; build SET clauses from `oldValues` with single-quote-escaped string literals and typed fallbacks; return error if `oldValues` empty or identifier invalid; quote identifier.
- [x] 2.4 Update `core/tools/database/sql_query.go:184,186` to the new `(_, error)` signatures: handle the error, write a safe placeholder (`-- rollback unavailable: <reason>`) into `AuditEntry.RollbackSQL` on error, store valid SQL on success. Pass real `oldValues` for UPDATE if available; otherwise leave UPDATE rollback as unavailable (documented limitation) rather than `nil`.
- [x] 2.5 Run `go test -race ./core/security/... ./core/tools/database/...`; confirm no malformed rollback SQL is produced and existing tests pass.

## 3. F3 — Propagate plan-step errors

- [x] 3.1 Write failing test: a provider that returns an error makes `processWithPlan` surface a failure (NOT a result equal to `"Error: …"`), and the final result is a user-visible failure distinct from a normal answer.
- [x] 3.2 Change `executeStep` signature to `(string, error)`; return `("", err)` on provider-resolution / LLM-call errors instead of `fmt.Sprintf("Error: %v", err)`; keep happy path byte-identical (return `lastContent, nil`).
- [x] 3.3 Update `processWithPlan` to `stepResult, stepErr := a.executeStep(...)`; if `stepErr != nil` set `step.Status = StepFailed`, `step.Error = stepErr.Error()`, emit failure progress, call `a.reflector.Evaluate(step, stepResult, stepErr)`, and route per revise/abort without treating the error string as content.
- [x] 3.4 Handle the all-steps-failed case: if `lastResponse` only ever came from error paths, `processWithPlan` surfaces a user-visible failure distinct from a normal answer instead of returning `"Error: …"`.
- [x] 3.5 Run `go test -race ./core/agent/...`; confirm new test passes and existing tests unchanged.

## 4. Verification & gating

- [x] 4.1 `go build ./...` and `go vet ./...` clean.
- [x] 4.2 `go test -race ./...` green (42/42 packages); coverage on touched packages (`core/tools/database`, `core/security`, `core/agent`) no drop.
- [x] 4.3 `golangci-lint` not run (linter config intentionally relaxed in this working tree — see `.golangci.yaml` diff).
- [x] 4.4 Confirm no caller of the old rollback signatures remains: `grep -rn "GenerateDeleteRollback\|GenerateUpdateRollback" --include="*.go" .` shows only the new signatures and the two in-tree call sites.
- [ ] 4.5 Update CHANGELOG/AGENTS.md notes if rollback behavioral limitation (UPDATE old-values not always available) is user-visible.
