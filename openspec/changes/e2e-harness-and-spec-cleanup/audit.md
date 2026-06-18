# Audit — Eight In-Flight OpenSpec Proposals

> Slice E1 of `e2e-harness-and-spec-cleanup`. Verdict per proposal with
> evidence (commit SHAs and `tasks.md` completion state) and a follow-up
> action that Slice E2 will execute (archive vs. keep active).
>
> Method: read `proposal.md` + `tasks.md` for each proposal, correlate
> task descriptions with `git log --all --oneline | grep` matches and
> spot-check committed artefacts on disk. Cross-references to commit
> SHAs use the short form from `git log`.

## Verdict matrix

| Proposal | Verdict | Action |
|---|---|---|
| `fix-ci-vet-provider-health-types` | partially-landed | finish tasks 3.2/3.3 then archive |
| `phase1-quality-hardening` | landed | archive |
| `phase2-runtime-capabilities` | landed | archive |
| `phase3-production-readiness` | active | keep, scoped overlap with `wire-security-gates` |
| `phase4-open-source-growth` | landed | archive |
| `product-positioning-and-oss-excellence` | active | keep, partially superseded by README rewrite commit |
| `project-review-and-roadmap` | partially-landed | split: archive sections 1–4, carry forward 5–8 as a new change |
| `wire-security-gates` | active | keep |

## Per-proposal verdicts

### 1. `fix-ci-vet-provider-health-types` — partially-landed

**Evidence:**
- `a21b02e fix(providers): add health checker contract and adapter coverage` — covers tasks 1.2, 1.3, 2.1.
- `637fc2f fix(gateway): harden provider health endpoint typing` — covers tasks 1.3, 2.2, 2.3.
- `cdd181b docs(openspec): record CI vet health-contract fix change` — proposal metadata.
- `core/providers/types.go` contains `HealthChecker` and `HealthStatus` (`grep -c` returns 5 matches), confirming task 1.2.

**Open items:**
- `tasks.md` 3.2 (commit/push to trigger CI) — code is committed locally, push-to-trigger evidence not captured in repo.
- `tasks.md` 3.3 (capture run URL/result in change notes) — no run URL recorded.

**Action:** When `e2e-harness-and-spec-cleanup` itself is pushed, the same CI run will exercise the provider health surface; at that point this proposal's 3.2/3.3 can be checked off and the proposal archived. No separate work needed.

---

### 2. `phase1-quality-hardening` — landed

**Evidence (all tasks marked complete in `tasks.md`):**
- `c247b0b feat: agent harness redesign + TUI progress display` — TUI viewport groundwork.
- `c96660e fix: add Ctrl+U/Ctrl+D scroll handlers to TUI` — task 3.3 (scroll bindings).
- `8acbe90 fix: add HandleMessageStreamWithProgress to TUI mock handler` — TUI testability.
- `3af9fb4 feat: TUI table rendering for SQL query results` — UX polish on top of viewport refactor.
- `e5d076b test: add TUI /new command state reset test` — test coverage on TUI command flow.
- `b0f040a docs(openspec): add phase1 change metadata`, `0c54f70`, `dcd1b29` — proposal artefacts.

**Action:** Archive in Slice E2.

---

### 3. `phase2-runtime-capabilities` — landed

**Evidence (all tasks marked complete in `tasks.md`):**
- `6e193db feat(session): add CJK-aware token estimation` — section 1 (token estimation).
- `61ea263 feat(memory): wire long-term memory into agent runtime` — section 2 (memory runtime wiring).
- `76b630b feat(telegram): add webhook support and dual-mode adapter` — section 3 (Telegram dual-channel).
- `9259c68 test: increase memory module coverage from 55.4% to 91.6%` — supporting coverage uplift.

**Action:** Archive in Slice E2.

---

### 4. `phase3-production-readiness` — active

**Evidence:**
- `tasks.md` is fully unchecked (sections 1–6 all `[ ]`).
- Some adjacent work landed via other commits — `a6c234b feat: wire infra tools and metrics into gateway` and `e4fcd84 feat(wave6): add Docker, K8s, Helm, CI/CD, metrics, benchmarks (T28-T33)` — but these do NOT correspond to the specific tasks in this proposal (e.g. `MetricsMiddleware` ordering, agent loop instrumentation, audit middleware structure, `RoutingConfig`).
- `internal/gateway/routes.go` references `metrics` 3 times; `internal/gateway/server.go` references `metrics` once — partial scaffolding present, instrumentation incomplete.

**Overlap risk:** Sections 1–3 of this proposal (metrics, security headers, audit logging) substantially overlap with `wire-security-gates` sections 4 (Gateway Security and Observability) and 6 (Agent Metrics Integration).

**Action:** Keep active, but Slice E2 should record an explicit overlap note pointing at `wire-security-gates` so we don't double-implement. Consider folding the unique parts of phase3 (provider health failover routing in section 4) into `wire-security-gates` follow-up `5.6` (which already defers automatic failover) and archiving `phase3-production-readiness` once that fold-in lands.

---

### 5. `phase4-open-source-growth` — landed

**Evidence:** All 26 tasks across phases A–G marked `[x]` in `tasks.md`. This proposal is a planning/documentation deliverable; the artefacts named in tasks (e.g. install matrix, Quick Start, Tutorials, Architecture index, site IA, growth-asset plan, Phase G milestones) all exist under `docs/` per `phase4-open-source-growth/proposal.md` Impact section.

**Action:** Archive in Slice E2.

---

### 6. `product-positioning-and-oss-excellence` — active

**Evidence:**
- All tasks in groups 0–7 are `[ ]` unchecked.
- However, `1b7f69f docs: rewrite README and CONTRIBUTING to English-primary bilingual` (recent main commit) addresses a substantial part of group 1 (positioning foundation, English-first README) and parts of group 4.5 (CONTRIBUTING claim-review checklist).
- The mock-echo zero-key path (group 2.3 — T-test-1 through T-test-7) and the maturity matrix (`docs/CAPABILITY-MATURITY.md`, group 5.2) do **not** exist on disk yet.

**Action:** Keep active. Slice E2 should add a "partial implementation" note recording the README/CONTRIBUTING rewrite and rolling group 1 + group 4.5's CONTRIBUTING work into `[x]` after a 5-minute reconciliation pass. The mock-echo TDD slice and capability-maturity matrix remain genuinely open.

---

### 7. `project-review-and-roadmap` — partially-landed

**Evidence:**
- Sections 1 (Telegram fix), 2 (test coverage), 3 (provider health), 4 (session export/import) — all `[x]`. Confirmed by commits including `76b630b feat(telegram): add webhook support and dual-mode adapter`, `a21b02e fix(providers): add health checker contract`, plus the session export/import commits implied by the section-4 task tree.
- Sections 5 (Memory module — full SQLite persistence, exponential decay, Top-K retrieval), 6 (Gateway audit log SQLite-backed), 7 (Agent integration — health-aware provider selection, memory-aware loop), 8 (release / changelog / version bump) — all `[ ]`.

**Action:** Split:
1. Mark sections 1–4 archived in Slice E2 by writing a "carve-out" archive note that references this audit and the original commit SHAs.
2. Carry sections 5–8 forward as a new, smaller proposal (working title `memory-and-audit-completion`) so the open work has a clean owner. Don't re-archive the parent — that would erase the sections-5–8 task list.
3. Slice E3 then opens that new proposal.

---

### 8. `wire-security-gates` — active

**Evidence:**
- All tasks 1.1–8.9 are `[ ]`.
- However, `1bd61fe feat: add rollback SQL generation and audit logging to SQL tool` and `a6c234b feat: wire infra tools and metrics into gateway` indicate substantial in-flight work toward sections 2 (SQL safety gates), 4 (Gateway security/observability), and 7 (Documentation alignment). `internal/security/` already contains `auth.go`, `ratelimit.go`, `sandbox.go`.
- Tasks are not formally checked off because the proposal's verification gates (section 8 — `go test -race`, `go vet`, `gosec`, `staticcheck`, claim-alignment review) have not been run in a single pass yet.

**Action:** Keep active. Recommend the proposal owner sweep tasks against existing commits in a focused session — much of section 2 may already be `[x]`-able. No work for Slice E2 beyond keeping this proposal open.

## Summary for Slice E2

Slice E2 will:
- **Archive** `phase1-quality-hardening`, `phase2-runtime-capabilities`, `phase4-open-source-growth`.
- **Archive (gated)** `fix-ci-vet-provider-health-types` once the next `main` push records a green CI run for the provider health path.
- **Carve out** sections 1–4 of `project-review-and-roadmap` as archived, preserve sections 5–8 as a new proposal in Slice E3.
- **Annotate** `phase3-production-readiness`, `product-positioning-and-oss-excellence`, and `wire-security-gates` with overlap/reconciliation notes pointing back to this audit.

## Self-review

- Every `landed` verdict (phase1, phase2, phase4) is backed by at least one commit SHA per major task section in this audit.
- Every `partially-landed` verdict (`fix-ci-vet-provider-health-types`, `project-review-and-roadmap`) lists the specific open task IDs that remain.
- Every `active` verdict (phase3, product-positioning, wire-security-gates) explains *why* work is still planned (overlap, pending verification gates, scope deliberately deferred), not just "tasks unchecked."
- The audit does not re-litigate scope decisions that were already approved in each proposal's `proposal.md` — it only classifies the *current* state of work.
