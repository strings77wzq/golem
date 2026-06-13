## Why

Golem has solid core architecture and feature modules, but significant production readiness gaps remain: provider health checking and routing code exist in `feature/` but are not wired into the runtime, the metrics module is implemented but not registered in the gateway, security headers and audit logging are missing, and documentation claims metrics that don't exist. Phase 3 closes these gaps so the project can actually run in production and be maintained by other developers.

## What Changes

- Wire existing `feature/health/manager.go` and `feature/routing/router.go` into the runtime so provider health is monitored and automatic failover works when a provider goes down.
- Wire existing `internal/metrics/handler.go` into the gateway server so the `/metrics` endpoint is available for Prometheus scraping.
- Implement the 13 missing metrics documented in `docs/MONITORING.md` (agent, token, tool, provider, session metrics).
- Add security headers middleware (CSP, X-Frame-Options, X-Content-Type-Options, HSTS, Referrer-Policy) to the gateway.
- Add request body size limiting middleware to the gateway.
- Add audit logging for security events (failed auth, rate limit hits, IP blocks).
- Fix `docs/MONITORING.md` to match actual implemented metrics.
- Update Helm chart with ServiceMonitor for Prometheus Operator integration.

## Capabilities

### New Capabilities

- `provider-health-failover`: Defines how provider health is monitored and how automatic failover works when a provider becomes unhealthy.
- `security-headers-hardening`: Defines what security headers the gateway adds to every response and what request size limits apply.
- `metrics-observability`: Defines what metrics are exposed at `/metrics` and how they are collected from the agent, tools, providers, and sessions.
- `audit-logging`: Defines what security events are logged and in what format.

### Modified Capabilities

- None.

## Impact

- Affected code: `cmd/golem/main.go`, `internal/gateway/routes.go`, `internal/gateway/server.go`, `internal/gateway/middleware.go`, `internal/metrics/metrics.go`, `core/agent/loop.go`, `core/agent/agent.go`, `docs/MONITORING.md`, `k8s/`, `docker/monitoring/`.
- Affected user flows: gateway startup, provider failover during outages, Prometheus scraping, security audit.
- Dependencies: existing `feature/health`, `feature/routing`, `internal/metrics` modules (all implemented but not wired).
