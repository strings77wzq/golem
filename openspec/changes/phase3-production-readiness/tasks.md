## 1. Metrics observability

- [ ] 1.1 Register `/metrics` endpoint in `internal/gateway/routes.go` using `metrics.Handler()`
- [ ] 1.2 Add `MetricsMiddleware` to the middleware chain in `internal/gateway/server.go`
- [ ] 1.3 Define agent metrics (requests, errors, latency) in `internal/metrics/metrics.go` and instrument `core/agent/loop.go`
- [ ] 1.4 Define token/cost metrics and instrument provider response handling in `core/agent/loop.go`
- [ ] 1.5 Define provider metrics (requests, errors, latency) and instrument `core/providers/factory.go`
- [ ] 1.6 Define session metrics (active, total) and instrument `core/session/store.go`
- [ ] 1.7 Run `go test ./internal/metrics ./internal/gateway` and confirm tests pass

## 2. Security headers hardening

- [ ] 2.1 Add `SecurityHeadersMiddleware()` to `internal/gateway/middleware.go` with CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy
- [ ] 2.2 Add `SecurityHeadersConfig` to `core/config/config.go` with enable/disable per header
- [ ] 2.3 Add request body size limiting middleware to `internal/gateway/middleware.go`
- [ ] 2.4 Wire security headers middleware into the gateway middleware chain in `internal/gateway/server.go`
- [ ] 2.5 Run `go test ./internal/gateway ./internal/security` and confirm tests pass

## 3. Audit logging

- [ ] 3.1 Add `AuditMiddleware()` to `internal/gateway/middleware.go` that logs failed auth, rate limit hits, and blocked IPs
- [ ] 3.2 Define audit event structure (event_type, client_ip, request_path, timestamp, severity)
- [ ] 3.3 Wire audit middleware into the gateway middleware chain in `internal/gateway/server.go`
- [ ] 3.4 Run `go test ./internal/gateway ./internal/security` and confirm tests pass

## 4. Provider health and failover

- [ ] 4.1 Add `RoutingConfig` to `core/config/config.go` with model alias → provider chain mapping
- [ ] 4.2 Create `cmd/golem/health_adapter.go` to instantiate HealthManager and register providers
- [ ] 4.3 Create `cmd/golem/routing_adapter.go` to instantiate Router with fallback chains from config
- [ ] 4.4 Add `--routing` CLI flag to `cmd/golem/main.go` and wire routing adapter
- [ ] 4.5 Wrap provider factory with Router in `core/agent/agent.go` for health-aware provider selection
- [ ] 4.6 Add failover logging in `core/agent/loop.go` when provider failover occurs
- [ ] 4.7 Run `go test ./feature/health ./feature/routing ./core/agent ./cmd/golem` and confirm tests pass

## 5. Documentation alignment

- [ ] 5.1 Update `docs/MONITORING.md` to match actual implemented metrics
- [ ] 5.2 Add documentation for security headers configuration in `docs/SECURITY.md`
- [ ] 5.3 Add documentation for provider failover configuration in `docs/CONFIG-REFERENCE.md`

## 6. Regression verification

- [ ] 6.1 Run `go test ./...` and confirm all packages pass
- [ ] 6.2 Manual QA: verify `/metrics` endpoint returns Prometheus format
- [ ] 6.3 Manual QA: verify security headers are present in HTTP responses
- [ ] 6.4 Manual QA: verify audit logging produces structured log entries on auth failure
