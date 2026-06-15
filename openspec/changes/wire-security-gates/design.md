## Context

Golem is positioned as a Go-native, local-first database AI agent that can expose database querying through CLI, MCP, and an OpenAI-compatible gateway. The production trust boundary is now larger than a local demo: users can point Golem at real databases, expose gateway endpoints, and allow AI-driven tool execution.

The codebase already contains many production-safety primitives:

- `core/security/gates.go` defines `PermissionChecker`, `QualityGate`, rollback SQL helpers, and `AuditEntry`.
- `core/tools/database/sql_query.go` already contains partial SQL safety wiring: operation classification, permission level, `QualityGate`, rollback SQL generation, and an optional audit callback.
- `core/tools/exec/exec.go` already defaults to non-shell execution and built-in allow/deny checks.
- `internal/security/sandbox.go` defines path and command validation but is not the single source of truth for the exec tool.
- `internal/gateway/server.go` already builds a middleware chain with request ID, logging, recovery, metrics, optional auth, optional rate limiting, and CORS.
- `internal/gateway/routes.go` already exposes `/health`, `/health/providers`, `/metrics`, chat, streaming, and session import/export routes.
- `feature/health` and `feature/routing` are tested reference modules, but runtime command wiring needs an explicit product decision before broad provider-routing changes.

The trust debt is therefore not lack of code. It is inconsistent production wiring and incomplete behavior contracts. Public claims must match what reaches runtime paths.

### Current runtime sketch

```text
                    ┌────────────────────┐
                    │     cmd/golem      │
                    │ composition root   │
                    └─────────┬──────────┘
                              │
          ┌───────────────────┼────────────────────┐
          │                   │                    │
          ▼                   ▼                    ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│ database tools   │ │ exec tool        │ │ gateway server   │
│ sql_query        │ │ command runtime  │ │ HTTP runtime     │
└────────┬─────────┘ └────────┬─────────┘ └────────┬─────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│ core/security    │ │ built-in checks  │ │ middleware chain │
│ partial wiring   │ │ + validator hook │ │ + routes         │
└──────────────────┘ └──────────────────┘ └──────────────────┘
```

This change makes the runtime safety contract explicit, closes missing paths, and adds integration tests so future refactors cannot silently disconnect the gates.

## Goals / Non-Goals

**Goals:**

1. Make SQL safety behavior explicit and testable:
   - read-only by default;
   - writes require explicit permission configuration;
   - UPDATE/DELETE without WHERE is denied before driver execution;
   - destructive/admin SQL is denied unless an explicit admin permission level is configured;
   - allowed and denied operations produce audit records when audit is configured;
   - rollback SQL is generated only when the operation shape supports a meaningful rollback statement.
2. Make exec sandbox behavior explicit and testable:
   - shell interpretation remains disabled by default;
   - command allow/deny and workspace/path validation run before command execution;
   - custom validators remain supported through the existing `CommandValidator` hook;
   - blocked commands produce structured tool errors and audit/metric signals where configured.
3. Make gateway production controls explicit and testable:
   - security headers are added by default;
   - request body size limits apply before full body reads;
   - auth failures, rate-limit denials, and IP blocks are audit-visible;
   - `/metrics` exposes actual registered metric names;
   - `/health/providers` reports real provider statuses when a checker is configured and clear `not_configured` behavior otherwise.
4. Add integration tests that exercise wired runtime paths through public tool/server APIs rather than only testing helper packages in isolation.
5. Align documentation with shipped behavior and mark any intentionally deferred production capability as not in scope.
6. Preserve the Go architecture constraints:
   - `cmd/` remains the composition root;
   - `core/` does not import `internal/` or `feature/`;
   - `internal/` does not import `feature/`;
   - `feature/` remains optional/reference unless deliberately wired by `cmd/`;
   - no CGO dependencies;
   - no new external runtime service requirement.

**Non-Goals:**

1. Do not introduce a new `ProductionRuntime` abstraction in this change.
2. Do not rewrite the ReAct agent loop or split `core/agent/loop.go` as part of this change.
3. Do not implement a full provider routing/failover platform unless the existing provider health manager can be wired without changing immutable provider interfaces.
4. Do not change `LLMProvider`, `StreamingProvider`, `MessageHandler`, or `Tool` signatures.
5. Do not add remote telemetry or analytics.
6. Do not make unsafe SQL or shell execution convenient. Explicit permission should be possible for local operators, but safe defaults win.

## Decisions

### Decision 1: Treat this as a wiring and contract change, not a rewrite

**Choice:** Keep existing packages and add narrow wiring, options, small safety middleware, callback seams, tests, and docs.

**Rationale:** The focused coverage run showed high coverage in the safety primitives that already exist: `internal/security` 87.9%, `internal/metrics` 96.8%, `feature/health` 93.3%, `feature/routing` 87.3%, `core/security` 94.6%. Rewriting would spend effort replacing tested components instead of connecting them.

**Implementation effort note:** This is not purely mechanical wiring. The following pieces are intentionally net-new but small: gateway security headers middleware, gateway body-size limit middleware, auth/rate-limit audit callback hooks, exec audit callback hooks, and a conservative SQL normalization/scanning path shared by classification, extraction, and quality gates.

**Alternatives considered:**

- **Minimal SQL-only patch:** Too small. It fixes the worst README mismatch but leaves gateway production claims and observability debt.
- **New production runtime layer:** Cleaner long-term shape, but it would touch too many files and create a new abstraction before the runtime contract is stable.

### Decision 2: Keep safety policy in `core` for database tools; adapt gateway/security details at the edges

**Choice:** SQL safety remains in `core/security` and `core/tools/database`. Gateway security stays in `internal/security` and `internal/gateway`. `cmd/golem` wires configuration and callbacks.

**Rationale:** Database tools live in `core`; importing `internal/security` into `core` would violate the layer rule. The correct pattern is small interfaces and constructor options at the call site.

```text
Allowed dependency direction

cmd/golem
  ├─ wires core/tools/database with core/security policy
  ├─ wires core/tools/exec with validator implementation
  └─ wires internal/gateway with internal/security middleware

core/tools/database ──▶ core/security ──▶ foundation/std
core/tools/exec     ──▶ core/tools + std
internal/gateway    ──▶ core + foundation + internal/security
```

**Implementation shape:**

- Database tool keeps or adds functional options such as permission level, quality gate, and audit callback.
- Exec tool keeps `WithValidator` and can accept a validator adapter constructed in `cmd/golem` or a core-local validator if one already exists.
- Gateway receives explicit `SecurityConfig` fields for body size limit, security headers, and audit callback if those are not already present.

### Decision 3: Use deny-by-default for ambiguous SQL operations

**Choice:** SQL classification MUST deny unknown or multi-statement operations unless explicitly configured at admin level.

**Rationale:** AI-generated SQL can be surprising. A permissive classifier is dangerous because the LLM may wrap destructive statements inside comments, whitespace, CTEs, or multiple statements. The safe invariant is: if the system cannot classify the operation as read-only, it does not execute under read permission.

**Algorithm:**

```text
normalize(sql)
  trim leading/trailing whitespace
  remove leading SQL comments only when safe to do so
  reject empty input
  reject multiple statements unless explicitly enabled
  identify first executable keyword

classify(keyword)
  SELECT / WITH-readonly       -> read
  INSERT / UPDATE              -> write
  DELETE                       -> delete
  DROP / ALTER / TRUNCATE      -> admin
  unknown                      -> admin/denied

quality gate
  if UPDATE or DELETE:
    require WHERE token in executable clause
    reject WHERE that is effectively missing or empty
  if affected row estimate available:
    warn/require confirm above threshold
```

**Optimization:** Use a small deterministic scanner instead of pulling in a SQL parser. This preserves pure-Go/no-new-dependency constraints and keeps checks cheap. The scanner should be conservative: false positives that block suspicious SQL are acceptable; false negatives that allow destructive SQL are not.

**Future option:** If SQL dialect complexity grows, introduce a small parser behind an interface and keep the conservative scanner as fallback.

### Decision 4: Audit denied operations as first-class events

**Choice:** Audit callbacks must receive both allowed and denied security-relevant decisions.

**Rationale:** Current partial SQL wiring can audit successful writes, but denied operations are more important for operators. Silent denials hide probing, prompt-injection attempts, and misconfiguration.

**Event model:**

```text
SQL event fields
  event_type: sql_allowed | sql_denied
  operation: SELECT | INSERT | UPDATE | DELETE | ADMIN | UNKNOWN
  database
  table if known
  status: success | denied | failed
  reason for denied/failed
  rollback_sql if available
  affected_rows if available

Gateway event fields
  event_type: auth_failure | rate_limit_hit | ip_blocked | request_too_large
  client_ip
  request_path
  method
  severity
  reason
```

**Security constraint:** Audit logs MUST NOT include provider API keys, authorization tokens, raw request bodies, or shell output. SQL text may be included only according to existing project logging posture; if included, args should be redacted or summarized.

### Decision 5: Body size limits must be middleware, not handler-local checks

**Choice:** Gateway request body limits should wrap the request body before handlers call `io.ReadAll`.

**Rationale:** `routes.go` reads request bodies in handlers. A handler-local check after reading the full body does not prevent memory pressure. Middleware using `http.MaxBytesReader` or equivalent prevents oversized requests before allocation grows.

**Data flow:**

```text
HTTP request
  │
  ▼
RequestID
  │
  ▼
Recovery
  │
  ▼
BodySizeLimit  ── reject 413 before full read
  │
  ▼
SecurityHeaders ─ add headers to every response
  │
  ▼
Auth / RateLimit / IP allowlist
  │
  ▼
Metrics
  │
  ▼
Route handler
```

**Optimization:** The body limit is O(1) memory overhead and relies on streaming limit enforcement. Do not pre-read bodies to measure length.

### Decision 6: Metrics must describe actual runtime names, not aspirational names

**Choice:** Use the metric names registered by `internal/metrics` and `core/agent/metrics.go`. Do not document `golem_*` names unless code registers that prefix.

**Rationale:** Operators copy PromQL from docs. Wrong names make monitoring look broken and hide production issues.

**Metric quality rule:** Every metric in docs must be traceable to a registration site or a task that adds the registration. Every metric registration added in this change must have at least one test that proves it changes under the expected runtime event.

### Decision 7: Provider health exposure ships before provider failover rewrite

**Choice:** `/health/providers` must report real configured provider status when a health checker is wired. Automatic failover is documented as deferred unless it can be wired with existing interfaces and no broad routing refactor.

**Rationale:** Health visibility is low-risk and immediately useful. Provider failover can alter user-visible model selection and error semantics. That deserves a separate focused spec if it requires routing changes.

**Acceptable in this change:**

```text
gateway command startup
  ├─ creates provider factory/providers as it already does
  ├─ creates health manager if configured providers exist
  ├─ registers health checkers without leaking credentials
  ├─ starts health manager with context-controlled lifecycle
  └─ passes health manager to gateway.Server.SetHealthChecker
```

**Not acceptable in this change:** replacing provider selection across the agent with a new router unless already hidden behind existing interfaces.

### Decision 8: Metrics cross layer boundaries through callbacks, not imports

**Choice:** Security-gate metrics from `core` packages must be emitted through callback or functional-option seams supplied by the composition root. `core/` packages must not import `internal/metrics`.

**Rationale:** Metric registration currently lives outside the pure domain layer. Directly importing `internal/metrics` from `core/tools/database` or `core/tools/exec` would violate the architecture and make core harder to reuse. The correct pattern is dependency injection: core emits a semantic event, and `cmd/golem` or an adapter maps that event to concrete metrics.

```text
core tool decision
  │
  ▼
SecurityEvent callback interface
  │
  ▼
cmd/golem composition root
  │
  ├─ audit logger
  └─ internal/metrics counters
```

**Implementation shape:** Define the smallest event/callback needed at the call site. Prefer functional options over a new service layer. If this starts to require more than 1-3 methods, stop and create a follow-up production-runtime proposal rather than smuggling a large abstraction into this slice.

### Decision 9: Tests drive implementation order

**Choice:** Implementation must follow SDD/TDD: spec scenarios become failing tests first, then minimal wiring, then refactor.

**Rationale:** This change is high-trust surface area. A test that proves a denied SQL statement never reaches the driver is more valuable than a helper unit test that proves `CheckSQL` works.

**Test layers:**

```text
Unit tests
  ├─ SQL classification scanner
  ├─ gate decisions
  ├─ sandbox validation adapter
  └─ middleware headers/body limit/audit events

Integration tests
  ├─ sql_query tool + fake driver proving denial before Execute
  ├─ exec tool + fake validator proving denial before os/exec
  ├─ gateway handler proving headers, 413, auth audit, metrics
  └─ provider health route with fake HealthStatusProvider

Quality gates
  ├─ go test -race ./...
  ├─ go test -cover ./...
  ├─ go vet ./...
  ├─ staticcheck ./... when available
  ├─ gosec ./... when available
  └─ focused benchmarks for hot middleware if changed
```

### Decision 9: Performance changes stay allocation-light

**Choice:** Security checks should be deterministic string scans, map lookups, and middleware wrappers. No regex-heavy hot path or per-request global locks unless measured.

**Rationale:** Gateway handlers and tool execution are production paths. Safety checks must be cheap enough to stay enabled by default.

**Performance targets:**

- Gateway security middleware: target <5 allocations/request for added middleware chain beyond existing baseline.
- Body limit middleware: O(1) memory overhead before handler reads.
- SQL classifier: O(n) over SQL length, no heap growth proportional to token count for common single-statement queries.
- Audit callback: non-blocking or bounded. If audit writes synchronously, failure must not silently permit unsafe operations. If async audit is added, goroutine lifecycle must have explicit shutdown.

**Benchmark requirement:** If middleware or SQL classification is changed in a hot path, add or update benchmarks with `b.ReportAllocs()`.

## Risks / Trade-offs

### Risk: Conservative SQL classification blocks valid advanced queries

Mitigation: Default to safety. Document the allowed SQL shape. Add table-driven tests for harmless `SELECT`, `WITH ... SELECT`, comments, whitespace, and parameterized statements. Defer complex multi-statement support to a separate proposal.

### Risk: Rollback SQL gives a false sense of reversibility

Mitigation: Rollback SQL must be described as best-effort and generated only when the system can produce a meaningful statement. For UPDATE rollback, do not fabricate old values unless they were actually captured. If old values are not available, return a rollback warning instead of fake SQL.

### Risk: Audit logs leak sensitive input

Mitigation: Audit schema must explicitly redact auth headers, provider keys, raw request bodies, and command output. Tests should assert sensitive headers are not logged.

### Risk: Middleware order causes missing headers or missing metrics on errors

Mitigation: Add tests for success, auth failure, rate-limit failure, panic recovery, oversized body, and not-found routes. Security headers should appear on error responses too.

### Risk: Rate-limit cleanup goroutine lifecycle remains unclear

Mitigation: If this change touches rate-limit internals, add explicit lifecycle or a test-only cleanup hook. If not touched, document as deferred follow-up rather than hiding it.

### Risk: Provider health checks create noisy startup failures when credentials are missing

Mitigation: Health manager must represent missing provider configuration as `not_configured` or `unknown`, not crash gateway startup. Operator-facing errors must not include secrets.

### Risk: Existing docs and specs contain aspirational metrics names

Mitigation: Documentation tasks must grep metric registration sites before writing PromQL. Any metric claim without code support is either removed or implemented with tests.

## Migration Plan

1. Add tests for current behavior gaps. These should fail before implementation:
   - denied unsafe SQL never reaches the driver;
   - denied SQL emits audit event;
   - exec validator denial prevents command execution;
   - oversized gateway request returns 413 before handler reads the full body;
   - gateway security headers appear on success and error responses;
   - `/health/providers` returns fake configured provider status in a gateway test;
   - documented metric names exist in `/metrics` output.
2. Wire the smallest runtime seams:
   - database tool options/callbacks;
   - exec validator adapter;
   - gateway security config additions;
   - gateway health checker setup in `cmd/golem`.
3. Add or update docs only after tests define behavior.
4. Run focused tests after each slice, then full gates.
5. Rollback strategy: each slice is additive and can be reverted independently. Defaults must remain safe if new config is absent.

## Open Questions

1. Should write SQL permission be enabled by CLI flag, config file, environment variable, or all three?
   - Recommendation: config + CLI flag for local operator clarity, environment only for gateway/server deployment.
2. Should audit logs include raw SQL text by default?
   - Recommendation: include operation, database, table, and reason by default; include raw SQL only when audit verbosity is explicitly enabled.
3. Should provider failover be included now?
   - Recommendation: no. Ship provider health visibility now; create a separate provider-routing spec if failover requires agent/provider selection changes.
4. Should the project standardize metric names with a `golem_` prefix?
   - Recommendation: not in this change unless the current registry already supports renaming safely. First align docs to code; prefix migration can be a compatibility change later.
