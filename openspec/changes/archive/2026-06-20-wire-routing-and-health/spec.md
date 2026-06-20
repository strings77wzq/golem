# Spec: Wire Routing and Health Modules

## Purpose

Wire the existing `feature/routing` and `feature/health` modules into the runtime composition root so they are available when enabled via CLI flags. This closes the "production readiness" gap where these modules are complete and tested but not accessible to users.

## Acceptance Criteria

### Routing (`--routing` flag)

1. A new `--routing` CLI flag accepts a JSON string defining model-to-provider fallback chains
2. When `--routing` is set, the agent uses `routing.Router.Chat()` instead of direct `factory.GetProviderForModel()` for LLM calls
3. When `--routing` is NOT set, behavior is identical to current (zero overhead, no code path change)
4. The Router wraps the existing Factory — no changes to `core/providers/` interfaces
5. Fallback on retryable errors works: if provider A fails with retryable error, try provider B
6. Non-retryable errors stop the fallback chain immediately

**Flag format** (example):
```json
{
  "routes": {
    "gpt-4o": ["openai/gpt-4o", "anthropic/claude-3.5-sonnet"],
    "deepseek-chat": ["deepseek/deepseek-chat", "openai/gpt-4o-mini"]
  }
}
```

### Health (`--health` flag)

1. A new `--health` CLI flag (boolean or JSON with interval) enables health checking
2. When enabled, a `health.Manager` is created and all providers implementing `HealthChecker` are registered
3. The Manager's background loop starts on agent startup
4. The Manager is injected into the gateway server via `SetHealthChecker()`
5. `GET /health/providers` returns real provider health statuses instead of "not_configured"
6. The Manager stops cleanly on context cancellation / SIGINT

**Flag format**:
- `--health` (boolean, default interval 5m)
- `--health {"interval": "60s"}` (custom interval)

## Files to Create

| File | Purpose |
|------|---------|
| `cmd/golem/routing_adapter.go` | Parse `--routing` flag, create Router, bridge to agent |
| `cmd/golem/health_adapter.go` | Parse `--health` flag, create Manager, bridge to gateway |
| `cmd/golem/routing_adapter_test.go` | Unit tests for routing adapter |
| `cmd/golem/health_adapter_test.go` | Unit tests for health adapter |

## Files to Modify

| File | Change |
|------|--------|
| `cmd/golem/main.go` | Add `--routing` and `--health` flags, conditionally create and inject |
| `cmd/golem/agent.go` | Accept optional Router, use it for provider resolution when set |

## Constraints

- MUST NOT modify `core/providers/types.go` interfaces
- MUST NOT modify `core/agent/agent.go` public interface
- Both features disabled by default (zero overhead when not used)
- Follow adapter pattern established by `mcp_adapter.go`, `rag_adapter.go`, `memory_adapter.go`
- `CGO_ENABLED=0` maintained
- All existing tests must pass without modification
