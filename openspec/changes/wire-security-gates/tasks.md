## 1. Baseline and Red Tests

- [ ] 1.1 Run `go test -cover ./internal/gateway ./internal/security ./internal/metrics ./feature/health ./feature/routing ./core/security ./core/tools/database ./core/tools/exec` and save the baseline output in the implementation notes.
- [ ] 1.2 Add failing SQL safety tests proving default `sql_query` permits SELECT and denies INSERT, UPDATE, DELETE, DROP, ALTER, TRUNCATE, unknown operations, empty SQL, and multi-statement SQL before driver execution.
- [ ] 1.3 Add failing SQL quality-gate tests proving UPDATE and DELETE without meaningful WHERE clauses are denied before driver execution.
- [ ] 1.4 Add failing SQL rollback tests proving DELETE rollback is generated only when table and WHERE are known, and UPDATE rollback does not fabricate unavailable old values.
- [ ] 1.5 Add failing SQL audit tests proving allowed writes and denied SQL decisions emit audit events when an audit sink is configured.
- [ ] 1.6 Add failing exec sandbox tests proving denied commands, non-allowlisted commands, and custom validator denials do not start operating system processes.
- [ ] 1.7 Add failing gateway security tests proving security headers appear on success and error responses.
- [ ] 1.8 Add failing gateway body-limit tests proving oversized `/api/chat` and `/api/chat/stream` requests return HTTP 413 and do not invoke agent handlers.
- [ ] 1.9 Add failing gateway audit tests proving missing/invalid auth, rate-limit hits, IP blocks, and oversized requests emit redacted audit events when audit is configured.
- [ ] 1.10 Add failing provider health route tests proving `/health/providers` returns configured fake provider statuses and explicit `not_configured` behavior.
- [ ] 1.11 Add failing metrics/docs tests or verification scripts proving every metric name documented in monitoring docs is registered or explicitly marked planned.

## 2. SQL Safety Gates

- [ ] 2.1 Implement or harden a conservative SQL normalization and classification path that rejects empty input, unknown executable operations, leading-comment bypasses, and multiple statements by default; use the same normalized view for classification, table extraction, WHERE extraction, SET extraction, and `QualityGate.CheckSQL`.
- [ ] 2.2 Harden permission checks so default database tool configuration is read-only and write/delete/admin operations require explicit operator configuration.
- [ ] 2.3 Harden destructive SQL quality checks so UPDATE and DELETE require a meaningful WHERE clause before driver execution.
- [ ] 2.4 Update rollback generation so DELETE rollback is best-effort and UPDATE rollback does not fabricate old values when old values were not captured.
- [ ] 2.5 Emit audit events for both allowed and denied SQL security decisions without leaking provider keys, authorization headers, or unredacted parameter values.
- [ ] 2.6 Add agent/security metrics increments for SQL gate allowed and denied decisions when registered metrics exist.
- [ ] 2.7 Keep SQL safety implementation dependency-safe: `core/tools/database` may import `core/security`, but must not import `internal/`, `feature/`, or `cmd/`.
- [ ] 2.8 Run focused tests for `core/security` and `core/tools/database` with `go test -race -cover`.

## 3. Exec Sandbox Enforcement

- [ ] 3.1 Preserve direct execution without shell interpretation as the default `exec` behavior.
- [ ] 3.2 Ensure built-in allowlist and denylist checks run before process creation in all security modes where they apply.
- [ ] 3.3 Wire or adapt `internal/security.Sandbox` through the existing custom validator seam without introducing an import from `core/tools/exec` to `internal/security`.
- [ ] 3.4 Ensure path-aware validation blocks denied paths and paths outside configured allowed roots before process creation.
- [ ] 3.5 Emit audit events for blocked and successful exec decisions when an audit sink is configured, without logging full command output by default.
- [ ] 3.6 Add agent/security metrics increments for exec gate denied decisions when registered metrics exist.
- [ ] 3.7 Add or update benchmarks if command validation enters a hot path and include `b.ReportAllocs()`.
- [ ] 3.8 Run focused tests for `core/tools/exec` and `internal/security` with `go test -race -cover`.

## 4. Gateway Security and Observability

- [ ] 4.1 Add default gateway security headers middleware for `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, and Content Security Policy.
- [ ] 4.2 Add explicit configuration for enabling, disabling, or overriding individual gateway security headers.
- [ ] 4.3 Add request body size limit middleware using streaming limit enforcement before route handlers call `io.ReadAll`.
- [ ] 4.4 Add audit hooks for authentication failures, rate-limit denials, IP allowlist denials, and oversized request denials.
- [ ] 4.5 Ensure gateway denial audit events include method, path, client IP, event type, severity, and reason while redacting tokens and raw request bodies.
- [ ] 4.6 Verify middleware ordering so security headers apply to success, validation errors, auth denials, rate-limit denials, panic recovery, and not-found responses.
- [ ] 4.7 Verify `/metrics` returns Prometheus text format and that HTTP request metrics update for success and denial paths.
- [ ] 4.8 Add or update gateway benchmarks for added middleware overhead if request hot-path code changes.
- [ ] 4.9 Run focused tests for `internal/gateway`, `internal/security`, and `internal/metrics` with `go test -race -cover`.

## 5. Provider Health Runtime

- [ ] 5.1 Wire a provider health checker into the gateway command path when provider health checking is configured.
- [ ] 5.2 Preserve explicit `not_configured` behavior when no provider health checker is attached.
- [ ] 5.3 Ensure provider health responses include status, last check time, latency when known, and sanitized errors without leaking credentials.
- [ ] 5.4 Ensure provider health checking starts and stops with a context-controlled lifecycle and does not leak goroutines on gateway shutdown.
- [ ] 5.5 Document provider health as visibility-only unless request routing actually uses health status for failover.
- [ ] 5.6 Explicitly defer automatic provider failover unless existing request routing already uses provider health without broad refactor; capture follow-up acceptance criteria in `TODOS.md` or a separate OpenSpec change.
- [ ] 5.7 Run focused tests for `feature/health`, `cmd/golem`, and `internal/gateway` with `go test -race -cover`; include `feature/routing` only if this change touches routing code.

## 6. Agent Metrics Integration

- [ ] 6.1 Verify existing LLM call, LLM error, LLM latency, token, tool call, tool latency, and tool error metrics update during agent message processing.
- [ ] 6.2 Wire missing context token and context compression metrics when the agent builds or compresses context.
- [ ] 6.3 Wire active session and total session metrics only on real session lifecycle transitions, not on tool construction.
- [ ] 6.4 Wire security gate allowed and denied counters from SQL and exec runtime decisions through callback or functional-option seams supplied by `cmd/golem`, without importing `internal/metrics` from `core/` packages.
- [ ] 6.5 Fix any metric that is currently recorded on the wrong path, such as plan duration being updated on non-planning paths if confirmed by tests.
- [ ] 6.6 Add tests that prove metric counters do not increment during validation-only setup or tool registration.
- [ ] 6.7 Run focused tests for `core/agent` and affected metrics packages with `go test -race -cover`.

## 7. Documentation and Claim Alignment

- [ ] 7.1 Update `README.md` safety claims to match runtime behavior and label rollback SQL as best-effort when exact rollback cannot be produced.
- [ ] 7.2 Update gateway/API docs so authentication behavior, no-auth local default, security headers, body limits, and health endpoints match code.
- [ ] 7.3 Update `docs/MONITORING.md` so every PromQL metric name is registered by code or explicitly marked planned.
- [ ] 7.4 Remove or correct references to missing Grafana dashboard files unless those files are added in this change.
- [ ] 7.5 Capture deferred production work in `TODOS.md` or follow-up OpenSpec changes, including provider failover if deferred, metric prefix migration if desired, and rate-limit goroutine lifecycle if not fixed.
- [ ] 7.6 Add changelog or release-note entry explaining production safety gate behavior changes and any operator-facing configuration changes.

## 8. Full Verification Gates

- [ ] 8.1 Run `openspec status --change wire-security-gates` and confirm artifacts remain apply-ready.
- [ ] 8.2 Run `go test -race ./...` and fix all failures.
- [ ] 8.3 Run `go test -cover ./...` and confirm no affected production package regresses below the project threshold.
- [ ] 8.4 Run `go vet ./...` and fix all findings.
- [ ] 8.5 Run `staticcheck ./...` when available and fix all new findings.
- [ ] 8.6 Run `gosec ./...` when available and fix all high or critical findings.
- [ ] 8.7 Run `CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath ./cmd/golem` to verify the single-binary invariant.
- [ ] 8.8 Run focused benchmarks for any changed hot path and reject regressions greater than 10% when statistically significant.
- [ ] 8.9 Perform final code review with security reviewer and general reviewer before shipping.
