# Compliance Fix Design — V1/V2/D1

**Date**: 2026-06-21
**Author**: Sisyphus (via brainstorming skill)
**Status**: Pending user review
**Audit basis**: CLAUDE.md + AGENTS.md hard constraints
**Preceded by**: Golem compliance audit (2026-06-21)

---

## Problem

The compliance audit completed on 2026-06-21 found 2 Critical layer violations and 1 documentation contradiction. This spec fixes all three so the project moves from ~90% to ~100% compliance with AGENTS.md hard constraints.

| ID | Problem | File:line | Severity |
|----|---------|-----------|----------|
| V1 | `core/` imports `internal/` (direction error) | `core/agent/metrics.go:4` | Critical |
| V2 | `internal/` imports `feature/` (violates §3) | `internal/wiring/features.go:7-8` | Critical |
| D1 | `openspec/config.yaml` contradicts `AGENTS.md §9` on background delegation | `openspec/config.yaml:171` | Critical (doc) |

---

## Scope

**In scope**: V1 + V2 + D1 only. Minor audit findings (V3 panic, V4 MCP file path, V5 naming, V6 Skills tool registration, G1-G7 doc gaps) are deferred to future iterations.

**Out of scope**: Any functional change, new feature, or refactoring beyond the minimal fixes described here.

---

## Decisions (confirmed by user)

1. **Direction**: Focus on compliance fix first (option A), not in-flight proposals or new features.
2. **Range**: Fix Critical + doc contradiction only (option B), not all minor findings.
3. **V2 approach**: Update AGENTS.md to legalize `internal/wiring/` as a composition helper that may import `feature/`. Code unchanged.
4. **D1 approach**: Keep background delegation (AGENTS.md §9 wins). Fix `openspec/config.yaml:171`.
5. **V1 approach**: Move `internal/metrics` → `core/metrics` as a whole (option A). Type definitions are domain abstractions belonging in core.

---

## Design

### V1 — Move `internal/metrics` → `core/metrics`

**Rationale**: `internal/metrics` defines `Counter`, `Gauge`, `Histogram`, and `Registry` types that are domain abstractions, not I/O adapters. `core/agent/metrics.go` consumes these types, so they belong in `core/` (same layer or below the consumer). The current `core → internal` dependency is a direction violation with no documented exception.

The package's dependencies are stdlib only (`math`, `sync`, `sync/atomic`, `net/http`, `sort`, `fmt`, `io`, `time`). Moving it to `core/` introduces no new cross-layer dependencies. `core/metrics` will be consumed by:
- `core/agent/metrics.go` — same-layer import (legal)
- `internal/gateway/middleware.go` — internal→core downward import (legal, strictly better than the current internal→internal)
- `internal/gateway/routes.go` — same as above

**File changes**:

| File | Change |
|------|--------|
| `internal/metrics/` → `core/metrics/` | Move entire directory (4 files: `metrics.go`, `handler.go`, `middleware.go`, `metrics_test.go`) |
| `core/agent/metrics.go:4` | Import path: `internal/metrics` → `core/metrics` |
| `internal/gateway/middleware.go` | Import path: `internal/metrics` → `core/metrics` |
| `internal/gateway/routes.go` | Import path: `internal/metrics` → `core/metrics` |
| `core/metrics/metrics.go:1` | Update package doc comment to note the layer placement rationale |
| `docs/MONITORING.md` | Update any import path examples |

**Verification**:
```bash
go build ./...
go test ./core/metrics/... ./core/agent/... ./internal/gateway/... -race
go test ./... -race
go vet ./...
grep -r "strings77wzq/golem/internal/metrics" --include="*.go" .  # expect zero matches
```

**Risk**: Low. Pure import path change, no logic modification. All consumers are identified (4 files).

---

### V2 — Update AGENTS.md §3 to legalize `internal/wiring/`

**Rationale**: `internal/wiring/features.go` imports `feature/skills` and `feature/skills/builtins`. The README architecture diagram already shows `cmd/ → internal/wiring/ → core/*, feature/*`, indicating this is an intentional design. `internal/wiring/` exists as a composition helper to keep `cmd/golem/main.go` thin by extracting provider/tool/skill registration into reusable functions. The AGENTS.md §3 rule should reflect this reality.

**File changes**:

In `AGENTS.md` §3, update the `internal/` rule block:

Current:
```
internal/    → imports core/ + foundation/ only
```

New:
```
internal/    → imports core/ + foundation/ only
                 exception: internal/wiring/ may import feature/ as a composition wiring helper
```

In the "Forbidden cross-layer imports" section, update:

Current:
```
- `internal/` MUST NOT import `feature/` or `cmd/`
```

New:
```
- `internal/` MUST NOT import `feature/` or `cmd/`
  exception: `internal/wiring/` is the composition wiring helper layer and MAY import `feature/` — it exists to keep `cmd/golem/main.go` thin by extracting provider/tool/skill registration into reusable functions. This matches the README architecture diagram: `cmd/ → internal/wiring/ → core/*, feature/*`.
```

**Sync update**: `openspec/config.yaml:28-31` — add the same exception to the `internal/` rule block in the `layer_dependencies` section.

**Verification**: Manual review. Code unchanged, `go build`/`go test` unaffected.

---

### D1 — Fix `openspec/config.yaml:171` background delegation contradiction

**Rationale**: `openspec/config.yaml:171` states "不使用后台 agent（background delegation 在此环境不可靠）" but `AGENTS.md §9` encourages background `task()` calls with `run_in_background=true`. Background delegation has been validated as effective (the compliance audit used 4 parallel explore agents, all returned complete results within 1m30s-2m). The prohibition in `openspec/config.yaml` is an early-stage observation that has been disproven by actual usage. `AGENTS.md §9` is the current authoritative guidance.

**File change**:

In `openspec/config.yaml:171`, replace:

Current:
```yaml
  - "不使用后台 agent（background delegation 在此环境不可靠）"
```

New:
```yaml
  - "并行 agent 委托：对独立探索任务使用 background task() 调用（run_in_background=true），与 AGENTS.md §9 一致"
```

**Verification**: Manual review confirming `openspec/config.yaml` and `AGENTS.md §9` no longer contradict.

---

## Execution Order

1. **V1 first** (code change, highest risk, needs compile+test validation)
2. **V2 second** (doc change, legalizes the `internal/wiring/` pattern that V1 doesn't touch but is related to layer rules)
3. **D1 third** (doc change, independent)

Each fix lands as a separate commit for clean rollback if needed.

---

## Invariants (must not change)

- `LLMProvider` / `StreamingProvider` interface signatures (`core/providers/types.go`)
- `CGO_ENABLED=0` build constraint
- All existing tests pass
- Bubble Tea isolation to `internal/channels/tui/`
- Tool alphabetical sorting in registry
- Streaming condition `canStream := ok && streamFinal && len(toolDefs) == 0`

---

## Testing Strategy

**V1** (code change):
- `go build ./...` — must pass
- `go test ./core/metrics/... ./core/agent/... ./internal/gateway/... -race` — must pass
- `go test ./... -race` — must pass (no regression)
- `go vet ./...` — must pass
- `grep -r "strings77wzq/golem/internal/metrics" --include="*.go" .` — must return zero matches

**V2** (doc change):
- Manual review of AGENTS.md §3 and `openspec/config.yaml` consistency
- `openspec validate` if the CLI is available

**D1** (doc change):
- Manual review confirming no contradiction between `openspec/config.yaml` and `AGENTS.md §9`

---

## Out of Scope (deferred to future iterations)

| ID | Finding | Why deferred |
|----|---------|--------------|
| V3 | `internal/channels/tui/command.go:28` panic | Minor, in internal package, not library code |
| V4 | MCP `ParseMCPConfig` missing file path support | Minor, spec vs implementation gap |
| V5 | `core/context/prompt.go` `skillPrompts` naming | Minor, cosmetic |
| V6 | Skills not registered as independent tools | Minor, design choice ambiguity |
| G1 | `openspec/` missing README | Major doc gap but not a compliance violation |
| G2 | `docs/study/` missing Wave labels | Minor, AGENTS.md §2 description mismatch |
| G4 | TODOS.md references nonexistent docs | Minor, stale references |

These will be addressed in a separate spec after this fix lands.
