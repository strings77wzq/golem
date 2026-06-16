# Auto-Compaction + Provider Registry + Loop Refactor — SDD

## [S1] Problem

After P0 wiring, three architectural issues remain:

1. **No auto-compaction** — when context window fills up, agent truncates silently. Users lose conversation history without warning.
2. **Provider factory switch-case** — 9 hardcoded vendors. Adding a new vendor requires modifying `internal/wiring/providers.go`.
3. **loop.go 971 lines** — too many responsibilities in one file. Hard to maintain, test, and extend.

## [S2] Solution Overview

| Phase | Task | Impact | Effort |
|-------|------|--------|--------|
| A | Auto-compaction in context manager | HIGH | 1-2 days |
| B | Provider registry pattern | MEDIUM | 1 day |
| C | loop.go decomposition | HIGH | 2-3 days |

---

## [S3] Phase A: Auto-Compaction

### A1: Add auto-compaction trigger to context manager

**File:** `core/context/manager.go`

Add `ShouldCompact()` method that checks if token count exceeds threshold:

```go
func (m *Manager) ShouldCompact(messages []providers.Message) bool {
    total := 0
    for _, msg := range messages {
        total += m.tokenFunc(msg)
    }
    return total > m.budget.TotalTokens*8/10  // 80% threshold
}
```

**File:** `core/agent/loop.go`

In `processMessage`, after building context, check if compaction needed:

```go
// Auto-compact if context is too large
if a.compactor != nil && a.contextManager.ShouldCompact(sess.GetMessages()) {
    a.compactor.Compact(ctx, sess, budget)
}
```

### A2: Add auto-compaction tests

**File:** `core/agent/compactor_autotrig_test.go`

Tests:
- `TestAutoCompact_TriggersAtThreshold` — compact when >80% full
- `TestAutoCompact_SkipsUnderThreshold` — no compact when <80%
- `TestAutoCompact_PreservesSystemPrompt` — system prompt never compacted

---

## [S4] Phase B: Provider Registry

### B1: Create provider registry

**File:** `core/providers/registry.go` (new)

```go
type Registry struct {
    mu          sync.RWMutex
    constructors map[string]func(cfg ProviderConfig) LLMProvider
}

type ProviderConfig struct {
    Vendor  string
    APIKey  string
    APIBase string
}

func (r *Registry) Register(vendor string, fn func(ProviderConfig) LLMProvider)
func (r *Registry) Create(cfg ProviderConfig) (LLMProvider, error)
```

### B2: Each provider registers itself

**File:** `core/providers/openai/register.go` (new)

```go
func init() {
    providers.GlobalRegistry.Register("openai", func(cfg providers.ProviderConfig) providers.LLMProvider {
        var opts []Option
        if cfg.APIBase != "" {
            opts = append(opts, WithAPIBase(cfg.APIBase))
        }
        return New(cfg.APIKey, opts...)
    })
}
```

### B3: Replace switch-case in wiring

**File:** `internal/wiring/providers.go`

Replace 9-case switch with:

```go
for _, entry := range cfg.ModelList {
    provider, err := providers.GlobalRegistry.Create(providers.ProviderConfig{
        Vendor:  entry.Vendor(),
        APIKey:  entry.APIKey,
        APIBase: entry.APIBase,
    })
    if err != nil {
        continue
    }
    factory.Register(entry.Vendor(), providers.NewRetryProvider(provider, providers.RetryConfig{}))
}
```

### B4: Provider registry tests

**File:** `core/providers/registry_test.go`

Tests:
- `TestRegistry_RegisterAndCreate` — register and create provider
- `TestRegistry_UnknownVendor` — error on unknown vendor
- `TestRegistry_Overwrite` — overwrite existing registration

---

## [S5] Phase C: Loop Decomposition

### C1: Extract streaming logic

**File:** `core/agent/streaming.go` (new)

Move `invokeProvider`, `wrapTokenEmitter` from loop.go.

### C2: Extract tool execution logic

**File:** `core/agent/tool_executor.go` (new)

Move parallel tool execution, result processing, hook calls from loop.go.

### C3: Extract hook orchestration

**File:** `core/agent/hook_runner.go` (new)

Move hook calling logic from loop.go.

### C4: Verify loop.go shrinks

Target: loop.go from ~971 lines to ~400 lines.

---

## [S6] Verification

```bash
go test -race ./core/... ./internal/... ./cmd/...
go vet ./...
go build ./cmd/golem/
```

## [S7] Risk

| Risk | Mitigation |
|------|------------|
| Auto-compaction too aggressive | 80% threshold, configurable |
| Provider registry breaks existing tests | Keep mock provider registered |
| Loop refactor introduces regressions | 41 existing test packages as safety net |
