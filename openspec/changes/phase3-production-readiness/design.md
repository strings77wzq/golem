## Context

Phase 3 addresses production readiness and developer experience gaps discovered during exploration. The project has strong foundational modules (health manager, router, metrics library) that are implemented but not wired into the runtime. Security middleware is partially implemented (auth, rate limiting, CORS, sandbox) but missing critical headers and audit logging. Documentation claims capabilities that don't exist in code.

Tech stack: Go 1.25+, cobra CLI, modernc.org/sqlite (pure Go), Bubble Tea v1.3.10, Docker, Kubernetes, GitHub Actions. CGO_ENABLED=0 is mandatory.

## Goals / Non-Goals

**Goals:**
- Wire provider health checking and automatic failover into the agent runtime.
- Wire existing metrics library into the gateway and implement documented metrics.
- Add security headers, request body limits, and audit logging to the gateway.
- Fix documentation to match actual implementation.

**Non-Goals:**
- Adding TLS/HTTPS support (deferred to Phase 4 — requires certificate management infrastructure).
- Distributed rate limiting (current in-memory implementation is sufficient for single-instance deployments).
- Brute force protection with progressive backoff (deferred — lower priority than headers and audit).

## Decisions

### 1. Wire health and routing via existing `--mcp`/`--rag` adapter pattern
- Create `cmd/golem/health_adapter.go` to instantiate HealthManager and register providers.
- Create `cmd/golem/routing_adapter.go` to instantiate Router with fallback chains from config.
- Add `RoutingConfig` to `core/config/config.go` with model alias → provider chain mapping.
- Agent wraps provider factory with Router in `core/agent/agent.go`.

**Alternatives considered:**
- Direct wiring in main.go (rejected: violates adapter pattern established by MCP/RAG/Memory).
- Making Router part of core/ (rejected: routing is optional, belongs in feature/).

### 2. Wire metrics via middleware chain
- Register `/metrics` endpoint in `internal/gateway/routes.go`.
- Add `MetricsMiddleware` to the middleware chain in `internal/gateway/server.go`.
- Instrument agent loop, tool execution, provider calls, and session tracking with metric definitions.

**Alternatives considered:**
- Using prometheus/client_golang (rejected: CGO_ENABLED=0 constraint, pure Go implementation already exists).
- Separate metrics server on different port (rejected: adds operational complexity, single endpoint is standard).

### 3. Security headers as gateway middleware
- Add `SecurityHeadersMiddleware()` to `internal/gateway/middleware.go`.
- Configurable via `GatewayConfig.SecurityHeaders` struct in `core/config/config.go`.
- Default enabled with sensible values (CSP: 'self', X-Frame-Options: DENY, etc.).

**Alternatives considered:**
- External proxy (nginx/caddy) for headers (rejected: adds deployment dependency, should be app-native).
- Per-route headers (rejected: over-engineered for current scope).

### 4. Audit logging as structured log entries
- Add `AuditMiddleware()` to `internal/gateway/middleware.go`.
- Logs failed auth attempts, rate limit hits, and blocked IPs as structured JSON.
- Reuses existing `foundation/logger` (no new logging infrastructure).

**Alternatives considered:**
- Separate audit log file (rejected: adds operational complexity, structured logs can be filtered).
- Syslog integration (rejected: out of scope for current deployment targets).

## Risks / Trade-offs

- **Metrics implementation may not match documentation exactly** → Mitigate by updating docs to reflect actual metrics before implementing new ones.
- **Provider failover may hide provider outages** → Mitigate by logging failover events and exposing failover metrics.
- **Security headers may break existing clients** → Mitigate by making headers configurable and providing sensible defaults.
- **Audit logging may increase log volume** → Mitigate by logging only security-relevant events, not all requests.

## Migration Plan

1. Wire metrics into gateway (lowest risk, pure addition).
2. Add security headers middleware (low risk, configurable defaults).
3. Add audit logging middleware (low risk, structured logs only).
4. Wire provider health checking (medium risk, affects provider initialization).
5. Wire provider routing/failover (medium risk, changes agent behavior on provider errors).
6. Update documentation to match implementation.
7. Run `go test ./...` and verify all packages pass.
8. Manual QA: verify /metrics endpoint, security headers, health check output.

## Open Questions

- Should provider failover be opt-in via config or always-on when multiple providers are configured?
- What CSP policy is appropriate for the gateway API (most restrictive: 'self' only)?
- Should audit logging be enabled by default or require explicit configuration?
