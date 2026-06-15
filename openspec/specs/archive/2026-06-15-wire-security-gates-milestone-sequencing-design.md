# Wire Security Gates — Milestone Sequencing Design

**Date:** 2026-06-15
**Topic:** Decompose and sequence the in-flight OpenSpec change `wire-security-gates` into shippable milestones.
**Companion artifacts:** `openspec/changes/wire-security-gates/{proposal.md,design.md,tasks.md}`

## Purpose

The `wire-security-gates` OpenSpec change covers eight task sections and roughly fifty discrete tasks across SQL safety, exec sandbox, gateway production controls, provider health, agent metrics, and documentation. Implementing it as one monolithic stream is high risk: long-lived branches, hard-to-review PRs, and weak feedback loops on the most security-critical code in the project.

This design decomposes that change into eight milestones (M0–M7), defines their order, dependencies, and per-milestone gates, and locks in the open-question recommendations from `design.md` so implementation can begin without re-litigating product decisions.

The design *does not* alter the goals, non-goals, or capability set of `wire-security-gates`. It is purely a sequencing and verification plan.

## Scope

### In scope

- Milestone breakdown that maps every task in `wire-security-gates/tasks.md` to exactly one milestone.
- Hard ordering and soft parallelism rules between milestones.
- The minimal new shared primitive (`core/security.EventSink`) needed to satisfy Decision 8 (no `core/` → `internal/metrics` import).
- Per-milestone red-test list, green outline, refactor expectations, and gate criteria.
- Risks introduced by the sequencing itself and their mitigations.
- Resolution of the four open questions from `wire-security-gates/design.md`.

### Out of scope

- Implementation code. This document is a sequencing spec, not a plan; the writing-plans skill produces the plan after this is approved.
- Changes to `wire-security-gates/proposal.md`, `wire-security-gates/design.md`, or the capability set.
- Provider failover (deferred per Open Question 3).
- A `golem_*` metric prefix migration (deferred per Open Question 4).
- Provider-routing rewrites or any change to `LLMProvider`, `StreamingProvider`, `MessageHandler`, or `Tool` signatures.

## Decisions locked in from `wire-security-gates/design.md`

These follow the recommendations in the original design's Open Questions section. They are restated here so the milestones can be planned without further user input.

1. **Write-permission surface (Open Q1).** Config file and CLI flag for local operator use. Environment variable for gateway/server deployment. Implemented in M1.
2. **Audit verbosity (Open Q2).** Default audit fields: operation, database, table, reason. Raw SQL text included only when an explicit audit verbosity flag is enabled. Flag introduced in M1.
3. **Provider failover (Open Q3).** Deferred. M4 ships visibility only. A follow-up OpenSpec change is captured in `TODOS.md` during M6.
4. **Metric prefix migration (Open Q4).** Not in this change. Docs align to existing registered names in M6. A future compatibility-style change can introduce `golem_*` prefixes if desired.

## Process decisions (chosen during brainstorming)

- **Release cadence:** ship as one batched release at the end. Internal milestones land continuously on a feature branch; one release tag publishes the combined change.
- **TDD shape:** vertical slice TDD per milestone (red → green → refactor inside each milestone), not a global red-first phase.
- **Verification posture:** strict per-milestone gate. The next milestone cannot start until the previous milestone's gate is green.
- **Open questions:** adopt the design.md recommendations as defaults (above).
- **Cross-cutting metrics seam:** built first as Milestone 0 so M1, M2, M3, and M5 can wire into a stable interface. M4 does not use the seam (provider health is visibility-only and emits no security events).

## Architecture invariants preserved

- `core/` does not import `internal/` or `feature/`. Security signals leave core via callback types only.
- `cmd/golem` and `internal/wiring` remain the only composition roots that bridge core gates to internal metrics, audit sinks, and gateway middleware.
- Defaults stay safe at every commit between milestones. A milestone reverted in isolation must not break shipped behavior.
- `CGO_ENABLED=0` and pure-Go builds work after every milestone — verified by the per-milestone strict gate.
- No new external runtime services or dependencies.

## Shared primitive (built in M0, used by M1, M2, M3, and M5)

A small event-sink interface lives in `core/security` and is the single seam through which security-relevant decisions cross from `core/` to `internal/`.

```go
// core/security/events.go (illustrative shape, not final names)
type EventType string

const (
    EventSQLAllowed              EventType = "sql_allowed"
    EventSQLDenied               EventType = "sql_denied"
    EventExecAllowed             EventType = "exec_allowed"
    EventExecDenied              EventType = "exec_denied"
    EventGatewayAuthFailure      EventType = "gateway_auth_failure"
    EventGatewayRateLimitHit     EventType = "gateway_rate_limit_hit"
    EventGatewayIPBlocked        EventType = "gateway_ip_blocked"
    EventGatewayRequestTooLarge  EventType = "gateway_request_too_large"
)

type Decision string

const (
    DecisionAllow Decision = "allow"
    DecisionDeny  Decision = "deny"
)

type Event struct {
    Type     EventType
    Decision Decision
    Reason   string            // human-readable; never contains secrets
    Fields   map[string]string // operation, database, table, client_ip, path, method, severity
}

type EventSink interface {
    Emit(ctx context.Context, evt Event)
}
```

`internal/wiring` provides one concrete sink that fans into the existing audit logger and `internal/metrics` counters. Decision 8's cap stands: if this interface needs more than a handful of methods, stop and write a follow-up production-runtime change rather than expanding the seam in this slice.

## Milestone breakdown

### M0 — Security-event callback seam

**Goal.** Establish the cross-layer seam used by every later milestone.

**Scope.**
- New file `core/security/events.go` with `EventType`, `Decision`, `Event`, `EventSink`, a no-op default sink, and a multi-sink composer that isolates panics between sinks.
- Functional options on the database and exec tools that accept an `EventSink`. The options are wired but not yet emitting; M1 and M2 fill in the call sites.
- New file in `internal/wiring` constructs an `EventSink` that fans into the existing audit logger and `internal/metrics` counters.

**Red tests.**
- No-op sink can be called concurrently with no observable effect.
- Multi-sink composer fans an emitted event to every registered child sink.
- A panicking child sink does not stop other child sinks from receiving the event.
- The wiring adapter can be constructed in a unit test without booting the gateway.

**Green outline.** Smallest possible interface and types; one composer; one adapter.

**Refactor.** Naming pass once M1 starts using it; otherwise leave alone.

**Acceptance.**
- `go test -race -cover ./core/security/...` passes.
- `go test -race -cover ./internal/wiring/...` passes for the new adapter.
- No runtime behavior change anywhere in the binary.

**Strict gate.** Same as every milestone (see "Per-milestone gate" below).

### M1 — SQL safety gates

**Goal.** Unsafe SQL never reaches the driver. Allowed and denied decisions emit `EventSink` events.

**Tasks covered.** `wire-security-gates/tasks.md` §1.2, §1.3, §1.4, §1.5, §2.1–§2.8.

**Scope.**
- A conservative SQL normalizer/scanner used as the single source of truth for classification, table extraction, WHERE/SET extraction, and `QualityGate.CheckSQL`. No SQL parser dependency.
- Default permission level is read-only. Write, delete, and admin operations require explicit operator configuration via config file, CLI flag, or environment variable (per Decision 1 above).
- WHERE enforcement on UPDATE and DELETE before driver execution. Effectively-empty WHERE clauses (`WHERE 1=1`, `WHERE TRUE`, etc.) are rejected by the scanner.
- Rollback SQL is best-effort. DELETE rollback is generated only when the table and WHERE are known. UPDATE rollback never fabricates old values; if old values are not captured, the result is a rollback warning string instead of a synthetic statement.
- Audit fields: operation, database, table, reason, status, optional rollback SQL, optional affected-row estimate. Raw SQL text is included only when the audit verbosity flag is on.
- `EventSink.Emit` fires on every classification decision (allow and deny) before driver execution.

**Red tests.**
- Default-config `sql_query` permits SELECT and denies INSERT, UPDATE, DELETE, DROP, ALTER, TRUNCATE, unknown operations, empty SQL, and multi-statement SQL.
- WHERE-less UPDATE and DELETE are denied before driver execution. A fake driver records call attempts; the test asserts zero attempts on denied paths.
- DELETE rollback is generated only when table and WHERE are known.
- UPDATE rollback returns a warning string when old values are not captured; it does not return synthetic SQL.
- Allowed SELECT and (with permission) allowed INSERT both produce a `sql_allowed` event.
- Denied operations produce a `sql_denied` event with a non-empty reason and no provider keys, authorization headers, or unredacted parameter values in any field.

**Green outline.** Harden `core/security/gates.go` and `core/tools/database/sql_query.go` to use the normalizer and emit events. Reuse the M0 functional option to receive the sink.

**Refactor.** Extract the scanner into a small private package or file inside `core/security` once green. Confirm no SQL parser was pulled in.

**Architecture check.** `core/tools/database` may import `core/security` but must not import `internal/`, `feature/`, or `cmd/`.

**Strict gate.** `go test -race -cover ./core/security/... ./core/tools/database/...`, `go vet`, `staticcheck` (when available), `CGO_ENABLED=0 go build ./cmd/golem`, no coverage regression versus the §1.1 baseline for touched packages.

### M2 — Exec sandbox enforcement

**Goal.** Every command goes through allow/deny and path validation before `os/exec`. Denied commands emit events. Shell interpretation stays off by default.

**Tasks covered.** `wire-security-gates/tasks.md` §1.6, §3.1–§3.8.

**Scope.**
- Shell interpretation off by default; allowlist on.
- Built-in allow/deny checks run before process creation regardless of any custom `CommandValidator`.
- An adapter constructed in `internal/wiring` lets `internal/security.Sandbox` plug into the existing `WithValidator` seam. `core/tools/exec` does not import `internal/security`.
- Path-aware validation rejects denied paths and paths outside configured allowed roots before process creation.
- Audit events for blocked and successful exec decisions. Full command stdout and stderr are not logged in audit by default; only exit code, duration, and arg-length summary.

**Red tests.**
- Denied commands never reach `exec.Cmd.Start`. A fake command runner records `Start` invocations; denied tests assert zero.
- Non-allowlisted commands are denied.
- Custom validator denial is honored even when the built-in allowlist would permit.
- Workspace-escape paths are rejected.
- Successful exec emits `exec_allowed`; blocked emits `exec_denied`.
- Audit fields contain no raw stdout, stderr, or full argument bodies beyond a summary.

**Green outline.** Add the validator-adapter wiring in `internal/wiring`. Emit events through the M0 sink. Keep `core/tools/exec` free of `internal/` imports.

**Refactor.** If command validation enters a hot path, add a benchmark with `b.ReportAllocs()` per Decision 9.

**Strict gate.** Scoped to `core/tools/exec/...` and `internal/security/...`.

### M3 — Gateway security and observability

**Goal.** Every gateway response carries security headers. Oversized requests die at middleware. Auth, rate-limit, IP, and oversize denials emit redacted audit events. `/metrics` exposes only registered names.

**Tasks covered.** `wire-security-gates/tasks.md` §1.7, §1.8, §1.9, §4.1–§4.9.

**Scope.**
- Default security headers middleware: `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, conservative Content Security Policy. Per-header configuration to enable, disable, or override values.
- Body-size limit middleware uses streaming enforcement (`http.MaxBytesReader` or equivalent) before any handler reads. Oversized requests return HTTP 413 without invoking handler logic.
- Audit hooks for `gateway_auth_failure`, `gateway_rate_limit_hit`, `gateway_ip_blocked`, and `gateway_request_too_large`. Audit fields: method, path, client IP, severity, reason. Tokens, raw bodies, and provider keys are never logged.
- Middleware order: `RequestID → Recovery → BodySizeLimit → SecurityHeaders → Auth → RateLimit → IPAllowlist → Metrics → handler`. Security headers must apply to error and not-found responses too.
- `/metrics` returns Prometheus text format. HTTP request metrics update for both success and denial paths.

**Red tests.**
- Oversized POST to `/api/chat` and `/api/chat/stream` returns 413 and the agent handler is never invoked. A mock agent records calls; the test asserts zero.
- Security headers are present on 200, 4xx, 5xx, panic-recovered, and 404 responses.
- Auth failure with a malformed bearer token emits a redacted audit event. The raw header value is not present in any event field.
- Oversized requests emit `gateway_request_too_large`.
- `/metrics` exposes the names registered by code. The test reflects over the registry rather than asserting against a hardcoded list, so adding a metric does not silently break docs.
- Middleware order is asserted by a test that inspects the chain's wrapping; reorders fail.

**Green outline.** Add the two new middlewares (security headers, body limit). Hook the four audit events into the existing chain. Verify ordering with the new test. Use the M0 sink to fan events into audit and metrics.

**Refactor.** If middleware overhead is touched on a hot path, add `BenchmarkMiddlewareChain` with `b.ReportAllocs()` and assert allocation overhead within the target band described in `wire-security-gates/design.md` Decision 9.

**Strict gate.** Scoped to `internal/gateway/...`, `internal/security/...`, `internal/metrics/...`.

### M4 — Provider health runtime

**Goal.** `/health/providers` reports real provider statuses when configured. Returns explicit `not_configured` otherwise. No credential leakage. Clean shutdown.

**Tasks covered.** `wire-security-gates/tasks.md` §1.10, §5.1–§5.7.

**Scope.**
- `internal/wiring` constructs a `feature/health` manager when configured providers exist, registers checkers without leaking credentials, and passes the manager to `gateway.Server.SetHealthChecker`.
- Lifecycle is bound to the gateway context. Shutdown waits for health goroutines.
- Provider health response fields: `provider`, `status`, `last_checked`, `latency_ms` (when known), `error` (sanitized).
- Explicit `not_configured` response when no checker is attached.
- Provider failover is deferred per Open Question 3. `TODOS.md` captures the follow-up OpenSpec change name during M6.

**Red tests.**
- A fake `HealthStatusProvider` is injected; `/health/providers` returns its statuses.
- Without a checker attached, `/health/providers` returns `not_configured`.
- The sanitized error field never contains the literal API key handed to a fake provider.
- After gateway shutdown, no health-related goroutines remain. A `goleak`-style assertion or a manual goroutine grep verifies the lifecycle.

**Green outline.** Wire the manager in `internal/wiring`; pass it through the existing `SetHealthChecker` API; ensure shutdown is bound to the gateway context.

**Strict gate.** Scoped to `feature/health/...`, `internal/gateway/...`, and `cmd/golem/...`. `feature/routing/...` is included only if M4 actually touches it; the design says it should not.

### M5 — Agent metrics integration

**Goal.** Runtime metrics reflect what is actually happening. Nothing increments during setup or registration.

**Tasks covered.** `wire-security-gates/tasks.md` §6.1–§6.7.

**Scope.**
- Verify LLM call, LLM error, LLM latency, token, tool call, tool latency, and tool error counters increment during real `agent.Run` paths.
- Wire context-token and context-compression metrics on actual context build and compress events, not on tool registration.
- Active-session and total-session metrics fire only on real session lifecycle transitions.
- Security-gate counters are fed from M0's `EventSink` via the `internal/wiring` adapter. `core/` packages do not import `internal/metrics`.
- Fix any metric currently recorded on the wrong path (for example, plan duration being updated on non-planning paths if confirmed by tests).

**Red tests.**
- A fake LLM round-trip during `agent.Run` increments call, latency, and token counters exactly once.
- Tool registration alone does not increment any counter.
- A SQL denial event from M1 increments the security-gate denied counter by one.
- A context compression event increments the compression counter.
- Session create, pause, and delete update `active_sessions` correctly at each transition.
- At least one regression test exists for each previously-misrouted metric fixed under §6.5.

**Green outline.** Audit each metric registration site against agent runtime paths; route through the M0 adapter for security-gate counters; correct any misrouted updates.

**Strict gate.** Scoped to `core/agent/...` and `internal/metrics/...`.

### M6 — Documentation and claim alignment

**Goal.** Every public claim points at code that exists.

**Tasks covered.** `wire-security-gates/tasks.md` §1.11, §7.1–§7.6.

**Scope.**
- `README.md` safety section rewritten to match runtime behavior. Rollback SQL labeled best-effort. Defaults table accurate. Gateway auth defaults documented.
- `docs/MONITORING.md` regenerated against the metrics registry. PromQL examples use real registered names. Aspirational metrics removed or marked `(planned)`.
- Gateway and API docs (`docs/GATEWAY-API.md`, related sections) reflect actual middleware behavior, headers, body-limit response, and `/health/providers` contract.
- Missing Grafana dashboards either removed from docs or added to the repo.
- `TODOS.md` captures deferred items: provider failover follow-up, optional `golem_*` prefix migration, and the rate-limit cleanup goroutine if untouched in this change.
- `CHANGELOG.md` Unreleased entry summarizes production safety gate behavior changes and operator-facing config changes.

**Red tests / verification.**
- A docs-grep verification script in `scripts/` parses `docs/MONITORING.md`, extracts metric names, and asserts each is either registered in code or appears in an explicit `(planned)` allowlist. The script runs in CI.
- A README claims-check test (a Go test or a grep-based shell test) ensures the Safety Model table words match the runtime defaults baked in M1 and M2.

**Strict gate.** Full repo: `go test -race -cover ./...`, `go vet ./...`, `staticcheck` and `gosec` when available, `CGO_ENABLED=0 go build`. This is the last broad gate before M7 release prep.

### M7 — Final verification and release prep

**Goal.** The change is apply-ready. Release artifacts are coherent. One batched release goes out.

**Tasks covered.** `wire-security-gates/tasks.md` §8.1–§8.9.

**Scope.**
- `openspec status --change wire-security-gates` reports apply-ready.
- Full `go test -race ./...`, `go test -cover ./...`, `go vet ./...`, `staticcheck` and `gosec` clean (high and critical).
- `CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath ./cmd/golem` succeeds.
- Benchmark deltas on touched hot paths: regressions greater than ten percent are rejected when statistically significant.
- Final reviews: a `security-reviewer` pass and a general code review pass, in that order, in separate lanes from the implementation lane.
- Tag a release. Publish notes drawn from M6's CHANGELOG entry.
- `openspec archive wire-security-gates` once tagged.

## Sequencing diagram

```
                            ┌──▶ M1 (SQL safety) ───┐
M0 (event-sink seam) ──┬─── ┤                       │
                       │    ├──▶ M2 (exec)          ├──▶ M5 (agent metrics) ─┐
                       │    └──▶ M3 (gateway)       ┘                        │
                       │                                                     ├──▶ M6 (docs) ──▶ M7 (verify+release)
                       └──────▶ M4 (provider health) ────────────────────────┘
```

**Hard ordering.** `M0 → {M1, M2, M3} → M5 → M6 → M7`. M4 runs in parallel with M1/M2/M3/M5; it must complete before M6 because docs depend on it.

**Soft parallelism.** M3 (gateway) and M4 (provider health) do not share files with M1 or M2 and may proceed in parallel branches if reviewer bandwidth allows. M1 and M2 share only the M0 event sink and could also be developed in parallel after M0 ships.

**Why M5 sits where it does.** M5 consolidates the security-gate counters fed by M1, M2, and M3, plus the agent runtime metrics. Putting it after those three feeders avoids touching `core/agent` and the metrics registry twice. M4 does not feed M5.

**Why M6 is single-milestone, not interleaved.** Per-milestone doc deltas still happen inside M1–M5 for their own surfaces, but the global cross-check (metrics grep, claim alignment, missing-dashboard cleanup) needs the registry and runtime to be stable. M6 doing it all at once produces one coherent doc-state delta.

## Per-milestone gate (applies to every milestone)

Every milestone must pass all of the following before the next milestone starts:

- `go test -race -cover` for the milestone's scoped packages, with no coverage regression versus the §1.1 baseline.
- `go vet` for the same packages, clean.
- `staticcheck` for the same packages when available, no new findings.
- `CGO_ENABLED=0 go build ./cmd/golem` succeeds.
- The milestone's red tests committed first as `test:` commits exist in history before the green commits.
- A coverage-delta note recorded in the implementation notes or branch description.

M6 additionally runs the full repo gate. M7 runs the full repo gate plus `gosec`, benchmarks, and the formal review passes.

## Per-milestone deliverable shape

Every milestone, regardless of whether it ships individually or batches into the final release, produces:

1. **Failing tests committed first** as `test:` commits covering the milestone's red-test slice.
2. **Implementation commits** (`feat:` and `fix:`) that turn red green, kept small enough to review individually.
3. **Refactor commits** (`refactor:`) once green, for example extracting the SQL scanner.
4. **Doc deltas scoped to the milestone** (M1 updates the SQL safety README section, M3 the gateway section, etc.). M6 handles the global cross-check.
5. **Coverage-delta note** on the branch describing baseline vs after, per touched package.
6. **Strict-gate evidence** pasted into the implementation notes or PR description.

Commit messages follow the repository's existing Conventional Commits style. AI co-author attribution is suppressed per the global git-workflow rule.

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| M0's `EventSink` is over-engineered before any caller exists. | M0 ships only the interface, two types, a no-op, a multi-sink composer, and the wiring adapter. No fields are added until M1 needs them. Decision 8's cap on the seam stands: stop at one to three methods. |
| M1 ships before audit verbosity is final. | The default fields are operation/database/table/reason. The audit verbosity flag is added in M1 itself; raw SQL text is gated behind it. Default verbosity stays low. |
| M3 middleware order regresses silently when someone reorders. | A middleware-order test inspects the chain wrapping or marker types and fails any reorder PR. |
| M4 scope creep into provider failover. | M4 is visibility only. `TODOS.md` captures a stub OpenSpec change name (for example, `provider-failover-routing`) during M6. PR reviewers reject failover code in M4. |
| M5 reveals existing metric bugs that need fixes outside §6 scope. | Each metric bug becomes its own commit inside M5. If a fix grows beyond `wire-security-gates/tasks.md` §6, spin a follow-up change rather than expanding scope. |
| M6 docs-grep script is flaky. | The script lives in `scripts/`, runs as a soft check in CI first, and is escalated to required only after one clean CI run. |
| Branch staleness from batched release. | Rebase main into the branch after each milestone's gate passes. Milestone PRs (or merge commits) keep the conflict surface small. |
| A previously-passing milestone regresses while later milestones are in flight. | Every milestone's red tests stay in the suite. The full repo gate at M6 catches regressions; the per-milestone gate catches them earlier when only the touched packages are at risk. |
| Hard ordering blocks reviewer throughput. | Use the soft-parallelism rules. M1+M2 may proceed in parallel after M0. M3+M4 may proceed in parallel after M0. M5 strictly waits for its feeders. |

## Success criteria

- Every red test in `wire-security-gates/tasks.md` §1.x is committed and green.
- Every verification gate in `wire-security-gates/tasks.md` §8.x passes in M7.
- `README.md`, `docs/MONITORING.md`, gateway API docs, and any safety-related documentation cite only behavior backed by code.
- `openspec archive wire-security-gates` succeeds and produces a clean delta against the five new capabilities (`sql-safety-gates`, `exec-sandbox-enforcement`, `gateway-security-observability`, `provider-health-runtime`, `production-claim-alignment`) and the modified `agent` capability.
- No regression in baseline coverage for any touched package.
- One release tag publishes the batched change with a single coherent CHANGELOG entry.

## Open follow-ups (intentionally deferred)

- Provider failover routing (Open Question 3). New OpenSpec change to be opened during M6.
- `golem_*` metric prefix migration (Open Question 4). Future compatibility-shaped change.
- Rate-limit cleanup goroutine lifecycle if not touched by M3. Captured in `TODOS.md` during M6.
- A potential `core/security` SQL parser if scanner-based classification proves insufficient on real-world dialects. Not in this change.
