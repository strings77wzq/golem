# Compliance Fix V1/V2/D1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 2 Critical layer violations (V1: core→internal direction error, V2: internal→feature rule violation) and 1 documentation contradiction (D1: background delegation) to move the project from ~90% to ~100% AGENTS.md compliance.

**Architecture:** V1 is a pure package relocation (`internal/metrics` → `core/metrics`) with import path updates across 3 consumers. V2 and D1 are documentation edits to AGENTS.md §3 and openspec/config.yaml. No logic changes, no interface signature changes, no new dependencies.

**Tech Stack:** Go 1.25, pure stdlib metrics package (math, sync, sync/atomic, net/http, sort, fmt, io, time)

**Spec:** `docs/superpowers/specs/2026-06-21-compliance-fix-design.md`

---

## File Structure

### V1 — Package relocation

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/metrics/metrics.go` → `core/metrics/metrics.go` | Move | Counter/Gauge/Histogram/Registry type definitions + DefaultRegistry + predefined HTTP metrics |
| `internal/metrics/handler.go` → `core/metrics/handler.go` | Move | Prometheus exposition format HTTP handler |
| `internal/metrics/middleware.go` → `core/metrics/middleware.go` | Move | HTTP middleware that records request metrics |
| `internal/metrics/metrics_test.go` → `core/metrics/metrics_test.go` | Move | Unit tests for metrics package |
| `core/agent/metrics.go` | Modify line 4 | Import path `internal/metrics` → `core/metrics` |
| `internal/gateway/middleware.go` | Modify line 13 | Import path `internal/metrics` → `core/metrics` |
| `internal/gateway/routes.go` | Modify line 13 | Import path `internal/metrics` → `core/metrics` |
| `docs/MONITORING.md` | Modify | Update any import path examples |
| `AGENTS.md` | Modify §2 layout tree | Move `metrics/` entry from `internal/` block to `core/` block |

### V2 — Legalize `internal/wiring/` in layer rules

| File | Action | Responsibility |
|------|--------|----------------|
| `AGENTS.md` | Modify §3 | Add exception for `internal/wiring/` importing `feature/` |
| `openspec/config.yaml` | Modify lines 28-31 | Add same exception to `layer_dependencies.internal` |

### D1 — Fix background delegation contradiction

| File | Action | Responsibility |
|------|--------|----------------|
| `openspec/config.yaml` | Modify line 171 | Replace background delegation prohibition with guidance aligned to AGENTS.md §9 |

---

## Task 1: Move `internal/metrics` → `core/metrics`

**Files:**
- Move: `internal/metrics/metrics.go` → `core/metrics/metrics.go`
- Move: `internal/metrics/handler.go` → `core/metrics/handler.go`
- Move: `internal/metrics/middleware.go` → `core/metrics/middleware.go`
- Move: `internal/metrics/metrics_test.go` → `core/metrics/metrics_test.go`

- [ ] **Step 1: Verify pre-move state is clean**

Run:
```bash
cd /home/strin/go/src/devLearn/aiLab/goclaw/golem
go build ./...
go test ./internal/metrics/... -race
```
Expected: Build passes, tests pass. This confirms the package is in a known-good state before relocation.

- [ ] **Step 2: Move the directory with git mv**

Run:
```bash
git mv internal/metrics core/metrics
```
Expected: Git tracks the move. Verify with `git status` — should show 4 files renamed from `internal/metrics/` to `core/metrics/`.

- [ ] **Step 3: Verify the move is complete**

Run:
```bash
ls core/metrics/
ls internal/metrics/ 2>&1 || true
```
Expected: `core/metrics/` contains `handler.go`, `metrics.go`, `metrics_test.go`, `middleware.go`. `internal/metrics/` should not exist (or be empty).

- [ ] **Step 4: Attempt build — expect import path errors**

Run:
```bash
go build ./...
```
Expected: FAIL with errors in `core/agent/metrics.go`, `internal/gateway/middleware.go`, `internal/gateway/routes.go` — they still reference `internal/metrics` which no longer exists. This confirms the consumers are correctly identified.

---

## Task 2: Update import paths in 3 consumers

**Files:**
- Modify: `core/agent/metrics.go:4`
- Modify: `internal/gateway/middleware.go:13`
- Modify: `internal/gateway/routes.go:13`

- [ ] **Step 1: Fix `core/agent/metrics.go` import**

Change line 4 from:
```go
	"github.com/strings77wzq/golem/internal/metrics"
```
to:
```go
	"github.com/strings77wzq/golem/core/metrics"
```

- [ ] **Step 2: Fix `internal/gateway/middleware.go` import**

Change line 13 from:
```go
	"github.com/strings77wzq/golem/internal/metrics"
```
to:
```go
	"github.com/strings77wzq/golem/core/metrics"
```

- [ ] **Step 3: Fix `internal/gateway/routes.go` import**

Change line 13 from:
```go
	"github.com/strings77wzq/golem/internal/metrics"
```
to:
```go
	"github.com/strings77wzq/golem/core/metrics"
```

- [ ] **Step 4: Verify build passes**

Run:
```bash
go build ./...
```
Expected: PASS — all 3 consumers now import from `core/metrics`.

- [ ] **Step 5: Verify no stale references remain in Go code**

Run:
```bash
grep -r "strings77wzq/golem/internal/metrics" --include="*.go" .
```
Expected: Zero matches. All Go code now references `core/metrics`.

- [ ] **Step 6: Run affected package tests with race detector**

Run:
```bash
go test ./core/metrics/... ./core/agent/... ./internal/gateway/... -race
```
Expected: All tests PASS. No race conditions detected.

- [ ] **Step 7: Run full test suite to confirm no regression**

Run:
```bash
go test ./... -race
```
Expected: All tests PASS. No regression in any package.

- [ ] **Step 8: Run go vet**

Run:
```bash
go vet ./...
```
Expected: PASS — zero warnings.

- [ ] **Step 9: Commit V1**

```bash
git add core/metrics/ internal/gateway/middleware.go internal/gateway/routes.go core/agent/metrics.go
git commit -m "refactor(metrics): move internal/metrics to core/metrics

Fixes Critical layer violation V1: core/agent/metrics.go was importing
from internal/metrics, violating the 'core/ MUST NOT import internal/'
rule in AGENTS.md §3. The metrics package defines domain abstractions
(Counter, Gauge, Histogram, Registry) that belong in the core layer.

Consumers updated:
- core/agent/metrics.go: internal/metrics → core/metrics
- internal/gateway/middleware.go: internal/metrics → core/metrics
- internal/gateway/routes.go: internal/metrics → core/metrics

No logic changes. All tests pass with -race. go vet clean."
```

---

## Task 3: Update AGENTS.md §2 layout and §3 layer rules

**Files:**
- Modify: `AGENTS.md` §2 (layout tree, line ~47)
- Modify: `AGENTS.md` §3 (layer dependency rules, lines ~60-72)

- [ ] **Step 1: Update §2 layout tree — move metrics/ from internal/ to core/**

In `AGENTS.md`, the layout tree currently shows (around line 42-48):
```
├── internal/              # Internal adapters (only importable within this module)
│   ├── channels/cli/      # Plain readline-style interactive mode
│   ├── channels/tui/      # Bubble Tea TUI (auto-activated on TTY)
│   ├── channels/telegram/ # Telegram bot adapter
│   ├── gateway/           # HTTP gateway server with SSE streaming
│   ├── metrics/           # Prometheus-compatible metrics (no external deps)
│   └── security/          # Auth middleware, rate limiting, command sandbox
```

Remove the `metrics/` line from the `internal/` block:
```
├── internal/              # Internal adapters (only importable within this module)
│   ├── channels/cli/      # Plain readline-style interactive mode
│   ├── channels/tui/      # Bubble Tea TUI (auto-activated on TTY)
│   ├── channels/telegram/ # Telegram bot adapter
│   ├── gateway/           # HTTP gateway server with SSE streaming
│   └── security/          # Auth middleware, rate limiting, command sandbox
```

Then update the `core/` line. The current `core/` line is a single-line summary (no sub-entries):
```
├── core/                  # Domain logic — agent, bus, config, providers, session, tools, usage
```

Change it to include `metrics` in the summary:
```
├── core/                  # Domain logic — agent, bus, config, metrics, providers, session, tools, usage
```

Note: `metrics` is inserted alphabetically between `config` and `providers`.

- [ ] **Step 2: Update §3 layer rules — add internal/wiring/ exception**

In `AGENTS.md` §3, the layer dependency rules currently show (around line 60-66):
```
cmd/         → imports ALL layers (composition root only)
internal/    → imports core/ + foundation/ only
core/        → imports foundation/ only (never internal/ or feature/)
feature/     → imports core/ + foundation/ only; wired into cmd/golem/ via adapters
foundation/  → imports stdlib only (zero project dependencies)
                 exception: github.com/mattn/go-isatty (CGO-free, used in term/detect.go for TTY detection)
```

Change the `internal/` line to:
```
cmd/         → imports ALL layers (composition root only)
internal/    → imports core/ + foundation/ only
                 exception: internal/wiring/ may import feature/ as a composition wiring helper
core/        → imports foundation/ only (never internal/ or feature/)
feature/     → imports core/ + foundation/ only; wired into cmd/golem/ via adapters
foundation/  → imports stdlib only (zero project dependencies)
                 exception: github.com/mattn/go-isatty (CGO-free, used in term/detect.go for TTY detection)
```

- [ ] **Step 3: Update §3 forbidden cross-layer imports — add internal/wiring/ exception**

In `AGENTS.md` §3, the forbidden imports section currently shows (around line 68-72):
```
**Forbidden cross-layer imports:**
- `foundation/` MUST NOT import `core/`, `internal/`, or `feature/`
- `core/` MUST NOT import `internal/`, `feature/`, or `cmd/`
- `feature/` MUST NOT import `internal/` or `cmd/`
- `internal/` MUST NOT import `feature/` or `cmd/`
```

Change the `internal/` forbidden line to:
```
**Forbidden cross-layer imports:**
- `foundation/` MUST NOT import `core/`, `internal/`, or `feature/`
- `core/` MUST NOT import `internal/`, `feature/`, or `cmd/`
- `feature/` MUST NOT import `internal/` or `cmd/`
- `internal/` MUST NOT import `feature/` or `cmd/`
  exception: `internal/wiring/` is the composition wiring helper layer and MAY import `feature/` — it exists to keep `cmd/golem/main.go` thin by extracting provider/tool/skill registration into reusable functions. This matches the README architecture diagram: `cmd/ → internal/wiring/ → core/*, feature/*`.
```

- [ ] **Step 4: Verify AGENTS.md reads correctly**

Read the modified §2 and §3 sections to confirm:
- `metrics/` appears in `core/` block, NOT in `internal/` block
- `internal/wiring/` exception is stated in both the rule block and the forbidden imports block
- The exception text is clear and references the README architecture diagram

- [ ] **Step 5: Commit V2 (AGENTS.md part)**

```bash
git add AGENTS.md
git commit -m "docs(agents): legalize internal/wiring/ in layer rules + relocate metrics to core/

V2: AGENTS.md §3 now explicitly allows internal/wiring/ to import feature/
as a composition wiring helper. This matches the README architecture diagram
(cmd/ → internal/wiring/ → core/*, feature/*) and the actual code in
internal/wiring/features.go which imports feature/skills.

Also updates §2 layout tree to reflect that metrics/ moved from internal/
to core/ (part of V1 fix)."
```

---

## Task 4: Update openspec/config.yaml for V2 and D1

**Files:**
- Modify: `openspec/config.yaml` lines 28-31 (V2 — layer dependency exception)
- Modify: `openspec/config.yaml` line 171 (D1 — background delegation)

- [ ] **Step 1: Update V2 — add internal/wiring/ exception to layer_dependencies**

In `openspec/config.yaml`, the `internal/` layer rule currently shows (around lines 28-31):
```yaml
    - layer: "internal/"
      may_import: ["core/", "foundation/"]
      must_not_import: ["feature/", "cmd/"]
      role: "I/O adapters (channels, gateway, metrics, security). Not importable outside this module."
```

Change to:
```yaml
    - layer: "internal/"
      may_import: ["core/", "foundation/"]
      must_not_import: ["feature/", "cmd/"]
      exception: "internal/wiring/ may import feature/ as a composition wiring helper (matches README architecture diagram)"
      role: "I/O adapters (channels, gateway, security) and composition wiring helpers. Not importable outside this module."
```

Note: `metrics/` is removed from the role description since it moved to `core/`.

- [ ] **Step 2: Update D1 — fix background delegation contradiction**

In `openspec/config.yaml`, line 171 currently shows:
```yaml
  - "不使用后台 agent（background delegation 在此环境不可靠）"
```

Replace with:
```yaml
  - "并行 agent 委托：对独立探索任务使用 background task() 调用（run_in_background=true），与 AGENTS.md §9 一致"
```

- [ ] **Step 3: Verify openspec/config.yaml reads correctly**

Read the modified `layer_dependencies.internal` section and the `ai_collaboration` section to confirm:
- `internal/wiring/` exception is clearly stated
- `metrics/` is no longer in the internal/ role description
- Background delegation line now aligns with AGENTS.md §9
- No remaining contradictions between openspec/config.yaml and AGENTS.md

- [ ] **Step 4: Validate openspec config if CLI is available**

Run:
```bash
which openspec 2>/dev/null && openspec validate openspec/ || echo "openspec CLI not installed, skipping validation"
```
Expected: Either validation passes, or "openspec CLI not installed" message (acceptable — manual review is sufficient for a YAML edit).

- [ ] **Step 5: Commit V2+D1 (openspec/config.yaml)**

```bash
git add openspec/config.yaml
git commit -m "docs(openspec): legalize internal/wiring/ exception + fix background delegation contradiction

V2: Adds internal/wiring/ exception to layer_dependencies.internal rule,
matching AGENTS.md §3 update. Removes metrics/ from internal/ role
description (moved to core/ in V1).

D1: Replaces '不使用后台 agent' prohibition with guidance aligned to
AGENTS.md §9, which encourages background task() calls for independent
exploration. The prohibition was an early-stage observation disproven
by actual usage (compliance audit used 4 parallel explore agents
successfully)."
```

---

## Task 5: Update docs/MONITORING.md import references

**Files:**
- Modify: `docs/MONITORING.md` (any `internal/metrics` import path examples)

- [ ] **Step 1: Search for internal/metrics references in MONITORING.md**

Run:
```bash
grep -n "internal/metrics" docs/MONITORING.md
```
Expected: Lists all lines referencing `internal/metrics` that need updating.

- [ ] **Step 2: Replace all internal/metrics references with core/metrics**

For each match found in Step 1, replace:
```
internal/metrics
```
with:
```
core/metrics
```

- [ ] **Step 3: Verify no stale references in docs**

Run:
```bash
grep -r "internal/metrics" docs/
```
Expected: Zero matches (or only historical references in CHANGELOG.md which should NOT be changed — changelog records past state).

- [ ] **Step 4: Commit doc update**

```bash
git add docs/MONITORING.md
git commit -m "docs(monitoring): update metrics import path references to core/metrics

Follows V1 package relocation from internal/metrics to core/metrics.
Updates import path examples in MONITORING.md."
```

---

## Task 6: Final verification

- [ ] **Step 1: Full build + test + vet**

Run:
```bash
go build ./...
go test ./... -race
go vet ./...
```
Expected: All three pass with zero errors.

- [ ] **Step 2: Verify zero stale internal/metrics references in Go code**

Run:
```bash
grep -r "strings77wzq/golem/internal/metrics" --include="*.go" .
```
Expected: Zero matches.

- [ ] **Step 3: Verify AGENTS.md and openspec/config.yaml consistency**

Run:
```bash
grep -A2 "internal/" AGENTS.md | head -5
grep -A5 "internal/" openspec/config.yaml | head -8
grep "background" openspec/config.yaml
grep "background" AGENTS.md
```
Expected:
- AGENTS.md shows `internal/wiring/` exception
- openspec/config.yaml shows `internal/wiring/` exception
- Both files have consistent background delegation guidance (no contradiction)

- [ ] **Step 4: Verify git log shows 4 clean commits**

Run:
```bash
git log --oneline -4
```
Expected: 4 commits visible:
1. `refactor(metrics): move internal/metrics to core/metrics`
2. `docs(agents): legalize internal/wiring/ in layer rules + relocate metrics to core/`
3. `docs(openspec): legalize internal/wiring/ exception + fix background delegation contradiction`
4. `docs(monitoring): update metrics import path references to core/metrics`

- [ ] **Step 5: Mark plan complete**

All 3 fixes (V1, V2, D1) are landed and verified. The project is now ~100% compliant with AGENTS.md hard constraints.
