# Plan: Golem v1.0 Trust Build

> CEO Review completed 2026-07-03. Mode: HOLD SCOPE. Approach B selected.

## Summary

Fix three trust gaps blocking v1.0 adoption: coverage gaps in critical packages, documentation drift, and release completeness. TLS deferred — local-first users don't need it yet.

## Premises (agreed)

1. Coverage is the bottleneck — 15 packages below 80%, `core/database` at 33.2% is a credibility problem for a DB agent
2. Documentation drift kills adoption — AGENTS.md says "memory in development" when it's been complete for weeks
3. GoReleaser CI exists but Docker push is missing — the gap is GHCR auth + `dockers:` config, not the trigger
4. TLS is not urgent for v1.0 — primary use case is local-first (CLI + TUI + MCP over stdio)

## Success Criteria

| # | Criterion | Current | Target |
|---|-----------|---------|--------|
| 1 | `internal/gateway` coverage | 66.2% | ≥ 80% |
| 2 | `core/tools/fileops` coverage | 64.8% | ≥ 80% |
| 3 | `core/agent` coverage | 62.6% | ≥ 75% |
| 4 | `internal/channels/tui` coverage | 63.7% | ≥ 70% |
| 5 | AGENTS.md status markers | Outdated | Match code reality |
| 6 | Docker push to GHCR | Not configured | Working on tag push |
| 7 | `go test -race ./...` | Passing | Passing (maintain) |
| 8 | Weighted average coverage | ~79% | ≥ 82% (by LOC per package) |

## Timeline (10 days)

| Days | Task | Details |
|------|------|---------|
| 1 | Documentation + Docker CI | AGENTS.md fix (trivial). Add `packages: write` + Docker login to `release.yml`. Add `dockers:` stanza to `.goreleaser.yml`. |
| 2-4 | internal/gateway tests | SSE streaming (`httptest` + goroutine/channel), OpenAI compat endpoints, middleware branches. 66.2% → 80% |
| 5-6 | core/tools/fileops tests | `safePath()` edge cases (symlink escape, `..` traversal, relative paths). Write `setupTestWorkspace(t)` helper. 64.8% → 80% |
| 7-8 | core/agent tests | ReAct loop error recovery, tool execution pipeline, context compaction triggers, provider fallback. 62.6% → 75% |
| 9 | internal/channels/tui tests | Key handling, progress parsing, viewport state. 63.7% → 70% |
| 10 | Final verification | `go test -race -cover ./...`, coverage report, release dry-run (`goreleaser release --snapshot --clean`) |

## Testing Strategy

- **internal/gateway:** `httptest` server for HTTP endpoints. SSE streaming tests with goroutine/channel orchestration. OpenAI compat format validation. No new dependencies.
- **core/tools/fileops:** `setupTestWorkspace(t)` helper creates temp dirs with symlinks, `..` references, binary files. Platform-aware (skip symlink tests on Windows CI if needed). Tests `safePath()` escape scenarios.
- **core/agent:** Targeted edge cases — tool error feedback paths, compaction triggers, provider resolution fallback. Concurrency paths covered by E2E tests.
- **internal/channels/tui:** Key handling, progress event parsing, viewport state. Bubble Tea model tests where feasible; accept diminishing returns.
- **No new dependencies.** Use `:memory:` SQLite (already in project), stdlib `httptest`, and existing test patterns. No sqlmock, no gomock.

## Risks and Contingencies

| Risk | Mitigation |
|------|------------|
| gateway SSE tests are flaky | Use `httptest` + deterministic channel signals, not time.Sleep |
| fileops symlink tests platform-dependent | Skip on Windows CI, test on Linux/macOS only |
| core/agent edge cases hard to isolate | Use MockProvider + MockTool, test specific error paths not full loop |
| Coverage baseline drift from new code | Run `go test -cover` daily, track trends |

## Deferred to Post-v1.0

- TLS support for HTTP gateway
- Security headers middleware
- Audit logging
- Helm chart ServiceMonitor
- Homebrew formula + tap repo
- Docker registry push automation (beyond GoReleaser)

## What Changed from brainstorm.md

3 factual errors corrected:
1. "CI only Linux amd64" → CI runs Linux + macOS; GoReleaser already configured for 6 platforms
2. "routing/health/metrics not wired" → All three are wired via CLI flags and gateway endpoints
3. "14 packages below 80%" → Actually 15

1 premise revised:
- "GoReleaser not in CI" → GoReleaser CI exists (`.github/workflows/release.yml`); gap is Docker push config

## GSTACK REVIEW REPORT

| Item | Status |
|------|--------|
| Runs | 3 review iterations + 1 outside voice |
| Issues found | 19 (review) + 4 (outside voice) |
| Issues fixed | 23 |
| Remaining | 0 |
| Quality score | 7/10 → 9/10 (post-fixes) |
| VERDICT | APPROVED — scope revised per outside voice |

### Section Findings

| Section | Finding |
|---------|---------|
| Architecture | No issues — no new components or data flows |
| Error & Rescue | No issues — no new codepaths |
| Security | GHCR auth: use GITHUB_TOKEN with `packages: write` permission |
| Data Flow | No issues — no new data flows |
| Code Quality | No issues — test files follow existing patterns |
| Test | Scope revised: gateway + fileops + agent + TUI (replaced core/database + cmd/golem) |
| Performance | No issues — no runtime impact |
| Observability | No issues — no new codepaths |
| Deployment | Dry-run `goreleaser release --snapshot --clean` before first real release |
| Long-Term | No issues — coverage reduces maintenance risk |
| Design & UX | Skipped — no UI scope |

### Outside Voice Impact

The outside voice challenged the scope selection. Key changes adopted:
1. **Replaced core/database + cmd/golem with internal/gateway + core/tools/fileops** — trust impact > coverage percentage
2. **No sqlmock** — use `:memory:` SQLite (already in project), stdlib `httptest`
3. **Docker CI = 30 min task** — not a full day
4. **Reordered by trust impact** — gateway → fileops → agent → TUI

NO UNRESOLVED DECISIONS
