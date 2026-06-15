## Why

Golem publicly promises production-grade safety behaviors such as read-only database access by default, WHERE enforcement for destructive SQL, rollback SQL generation, audit logging, gateway security controls, metrics, and provider health visibility. The repository already contains many of the building blocks, but several are not wired into runtime execution paths, which creates trust debt: users may rely on protections that are documented but not actually enforced.

This change turns the existing safety, observability, and health modules into a coherent trusted production slice. Most work is wiring and contract hardening, but a few small production controls are net-new implementation: security headers middleware, request body limit middleware, audit callback seams, and conservative SQL normalization. The goal is not a rewrite. The goal is to make Golem's production-facing claims true, tested, observable, and documented while preserving the pure-Go, CGO-free, single-binary architecture.

## What Changes

- Wire SQL permission and quality gates into the database tool execution path so read-only defaults, explicit write authorization, WHERE enforcement, and rollback SQL behavior are enforced before SQL reaches a driver.
- Wire command and path sandbox validation into the exec tool runtime path so shell execution follows the same allowlist and workspace-boundary rules already modeled in `internal/security`.
- Wire gateway health, security, audit, and metrics behavior into the HTTP runtime path so `/health/providers`, `/metrics`, authentication failures, rate-limit denials, IP blocks, security headers, and body-size limits are observable and testable.
- Align provider-health behavior with the current architecture by exposing real health status first, while avoiding a broad provider-routing rewrite in this change.
- Add integration tests for the wired runtime paths, not just isolated unit tests for helper packages.
- Correct public documentation and monitoring examples so metric names, security claims, gateway authentication, and production readiness claims match shipped behavior.
- Preserve existing immutable interfaces unless an explicit compatibility shim is included. In particular, do not break `core/providers.LLMProvider`, `core/providers.StreamingProvider`, `core/agent.MessageHandler`, or `core/tools.Tool`.

## Capabilities

### New Capabilities

- `sql-safety-gates`: Defines runtime enforcement for database query safety, including read-only default behavior, write authorization, destructive SQL quality checks, rollback SQL generation, and audit records for allowed and denied SQL operations.
- `exec-sandbox-enforcement`: Defines runtime enforcement for shell command execution and file path validation, including command allow/deny decisions, workspace boundary handling, and audit records for blocked commands.
- `gateway-security-observability`: Defines gateway production controls, including security headers, request body size limits, authentication/rate-limit/IP-denial audit logs, Prometheus metrics exposure, and integration-level verification.
- `provider-health-runtime`: Defines runtime provider health exposure through the gateway, including configured provider checks, degraded/unhealthy status reporting, not-configured behavior, and operator-facing error details that do not leak secrets.
- `production-claim-alignment`: Defines the documentation and release-claim gate that keeps README, API docs, monitoring docs, and production-readiness statements aligned with wired, tested behavior.

### Modified Capabilities

- `agent`: Agent-facing production metrics and security-gate counters must reflect real runtime behavior when message processing triggers LLM calls, tool calls, context compression, sessions, and security-gate decisions.

## Impact

- Affected runtime code areas:
  - `core/tools/database/` for SQL execution safety and rollback/audit behavior.
  - `core/tools/exec/` and composition wiring for command sandbox enforcement.
  - `internal/security/` for middleware, sandbox, audit, and reusable gateway controls.
  - `internal/gateway/` for health, metrics, security middleware, and OpenAI-compatible endpoint behavior.
  - `internal/metrics/` and `core/agent/metrics.go` for metric registration and runtime updates.
  - `feature/health/` for provider health management used by gateway runtime wiring.
  - `cmd/golem/` as the composition root that wires these modules without creating layer cycles.
- Affected documentation:
  - `README.md` safety claims.
  - `docs/MONITORING.md` metric names and Prometheus examples.
  - Gateway/API docs that describe authentication, security headers, and health behavior.
  - Release notes or changelog entries for production-safety behavior changes.
- API and behavior impact:
  - Unsafe SQL and blocked shell commands produce explicit structured errors instead of reaching the driver or shell.
  - Gateway responses include production security headers by default.
  - Oversized gateway requests return HTTP 413.
  - `/health/providers` returns configured provider status when health checks are wired, and explicit `not_configured` status otherwise.
  - `/metrics` exposes only metric names actually registered by the codebase.
- Dependency impact:
  - No new external runtime dependency is expected.
  - The change must remain compatible with `CGO_ENABLED=0` and pure-Go builds.
  - Any optional provider health check must degrade cleanly when provider credentials are absent.
