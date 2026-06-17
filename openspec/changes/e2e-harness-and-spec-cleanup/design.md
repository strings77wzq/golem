## Context

Golem v0.9.1 has 41 packages, 82.5% unit-test coverage, and CI green on every push to `main`. All claimed integrations (Ollama, sql_query safety gates, MCP server, gateway streaming) are implemented and pass mocked tests. What is missing is **black-box behavioural evidence** that the assembled binary behaves as the README promises when driven by a real local LLM against a real database.

At the same time, eight OpenSpec proposals sit in `openspec/changes/` whose tasks have largely landed in `main` (verifiable via `git log`), but none have been moved to `openspec/specs/archive/`. New contributors reading the repo cannot tell which proposals are real work-in-progress vs. historical record.

These two problems share one solution surface — both are governance debts that grow if left alone — and one shared constraint set:

- **Layer purity** (AGENTS.md §3): the existing `cmd/` `internal/` `core/` `feature/` `foundation/` boundaries cannot be perforated to make E2E testing easier.
- **Build invariant** (AGENTS.md §4): `CGO_ENABLED=0` everywhere, pure Go, single static binary.
- **Frozen interfaces** (AGENTS.md §5): `LLMProvider`, `StreamingProvider`, `MessageHandler`, `Runner` must not change.
- **Existing CI green budget**: zero tolerance for breaking the main `ci.yml` workflow during this change.

Stakeholders: maintainer (`strings77wzq`), future contributors who will write E2E cases or add capabilities, downstream users whose trust depends on the README claims being verifiable.

## Goals / Non-Goals

**Goals:**

1. Convert four headline product claims (`agent ↔ sql_query`, `safety gate`, `MCP server`, `gateway streaming`) into automated black-box tests that a contributor can run locally with a single `go test ./tests/e2e/...` invocation.
2. Make the E2E suite **non-blocking for routine PRs** so a contributor without Ollama installed never sees a red required check from this work, while still surfacing failures clearly when Ollama is available.
3. Produce a **transcript artifact** for each test that doubles as a README/docs proof asset, so claim discipline (per `product-positioning-and-oss-excellence`) has real evidence behind it.
4. Bring the OpenSpec change directory into a **truthful state**: archive what has landed, retain only genuinely active proposals, and leave an audit trail explaining each verdict.
5. Establish a **proposal lifecycle hygiene** spec so this housekeeping does not silently drift again in three months.

**Non-Goals:**

- **Replacing the unit test suite**: E2E supplements unit coverage, never substitutes for it. Unit tests stay the contract for individual components.
- **Asserting LLM output text exactly**: real LLM outputs are nondeterministic; we assert *structural* and *behavioural* invariants only.
- **Cross-platform E2E (macOS, Windows, Termux ARM64) in this change**: only Linux amd64 with Ollama installed locally or in CI. Other platforms can come later as separate workflows.
- **Adding new product features**: this change adds *evidence*, not capabilities. No `core/` `feature/` `internal/` source file is modified by Track A.
- **Re-litigating archived proposals**: Track E only reclassifies proposals that are currently in `openspec/changes/`; it does not modify anything already under `openspec/specs/archive/`.
- **Performance benchmarks** (Candidate B from CEO review): out of scope here, deferred to a future change once the E2E foundation exists.

## Decisions

### D1. Place E2E suite at top-level `tests/e2e/` as its own Go module

**Decision**: Create `tests/e2e/go.mod` so the E2E suite is its own Go module, importing the public `golem` binary as a black-box subprocess rather than importing any internal Go packages.

**Alternatives considered:**

- *Inline under the existing module as `internal/e2e/`*: would tempt future contributors to import `core/` or `feature/` types directly to "make tests easier", which would defeat the black-box property and violate AGENTS.md §3 layer rules.
- *External repo*: would lose proximity to the code being tested, making "green PR ⇒ green E2E" coupling impossible.

**Rationale**: A separate top-level module under the same repo gives us colocation (one `git clone` runs everything) without the architectural temptation of internal-package imports. The E2E module can have its own minimal dependencies (`testing`, `os/exec`, `net/http`) without polluting the main `go.mod`.

### D2. Use Ollama with `qwen3:0.5b` as the deterministic-enough local LLM

**Decision**: The E2E suite assumes Ollama is available at `http://localhost:11434` and that the `qwen3:0.5b` model has been pulled. Both conditions are checked at suite startup; missing prerequisites produce a clean `t.Skip` with a clear message, not a failure.

**Alternatives considered:**

- *Mock provider only*: defeats the entire point of this change — mocked tests already exist.
- *Larger model (e.g., `llama3:8b`)*: too slow for CI (multi-second per turn × 4 tests × possible retries), too memory-hungry for free GitHub runners (7 GB RAM ceiling).
- *Smaller model than 0.5b (e.g., `qwen3:0.3b`)*: not consistently published; 0.5b is the smallest broadly available with reasonable tool-use behaviour.

**Rationale**: 350 MB on disk, runs comfortably on a 7 GB GitHub runner, completes a tool-call turn in 5–15 s on CPU, and produces structurally valid tool calls reliably enough for invariant assertions.

### D3. Assert behavioural invariants, never literal text

**Decision**: Tests assert **what** the agent did (which tool name was called, what shape the `ToolResult` had, whether the safety gate fired with a recognisable error category, what exit code was returned) — not **what** the LLM said in plain text.

**Alternatives considered:**

- *Snapshot testing of LLM responses*: would be flaky on the first model upgrade or seed change; would create permanent maintenance burden.
- *Regex-on-output*: same flakiness, just with extra steps.

**Rationale**: Behavioural invariants are stable across model upgrades and across the natural variance of LLM sampling. A test that asserts "the agent invoked `sql_query` with a SELECT statement and returned exit code 0" is robust; a test that asserts the agent said "Here are the users: ..." is not.

### D4. Use a separate non-blocking GitHub Actions workflow for E2E

**Decision**: Add `.github/workflows/e2e.yml` triggered by `pull_request` on relevant paths (`core/agent/**`, `core/providers/**`, `core/tools/**`, `internal/security/**`, `internal/gateway/**`, `feature/mcp/**`, `tests/e2e/**`) and by `workflow_dispatch`. The job is **not** added to `branch protection required checks`. Failure is visible as an unmerged check, not as a blocked merge.

**Alternatives considered:**

- *Add to main `ci.yml` as required*: would block every PR until Ollama installs reliably in every CI run, and would slow the CI feedback loop from ~3 min to ~10 min for changes that don't touch the agent path at all.
- *Cron-only nightly run*: would lose tight coupling between PR and verification; bugs introduced today get surfaced tomorrow morning at the earliest.

**Rationale**: Decoupling lets us iterate on E2E reliability without holding the project's existing fast feedback loop hostage. Once the suite has 30+ days of green history, branch protection upgrade is a one-line change.

### D5. Ollama lifecycle is "detect, don't manage" by default

**Decision**: The lifecycle helper checks `GET http://localhost:11434/api/tags` and either proceeds (if Ollama responds and `qwen3:0.5b` is listed) or `t.Skip`s with instructions. The helper does **not** spawn Ollama itself except in CI where an explicit `GOLEM_E2E_MANAGE_OLLAMA=1` env var is set.

**Alternatives considered:**

- *Always spawn ollama subprocess from Go*: invasive on developer machines; risks port conflicts with locally running Ollama; makes graceful skip impossible.
- *Always require Ollama to be pre-running*: too rigid for CI where we want to install + run + cache.

**Rationale**: Local developer ergonomics (skip cleanly if Ollama isn't there) and CI determinism (start it explicitly when expected) are different needs; one env-var switch is the cleanest separation.

### D6. Transcripts are appended to `tests/e2e/transcripts/<name>.log`, not asserted against

**Decision**: Each E2E test captures stdout/stderr of the `golem` subprocess plus its own structured event log into a transcript file checked into the repo via a CI artifact (not committed to git). README and docs link to the most recent CI artifact for the proof asset; locally, the file appears in the working directory.

**Alternatives considered:**

- *Commit transcripts to git*: would create churn and version control noise on every model update.
- *Discard transcripts*: loses the proof-asset value entirely.

**Rationale**: Artifact upload from CI is free and persistent, gives us a public link to a real run, and avoids polluting the git history.

### D7. Track E archiving uses the existing `openspec archive` command per proposal, with a single `audit.md` summary

**Decision**: For each of the eight proposals in scope, decide *landed* / *partially landed* / *active* by reading its `tasks.md` and grepping `git log` for the implementing commits. Record the verdict + evidence (commit SHAs, file paths) in `openspec/changes/e2e-harness-and-spec-cleanup/audit.md`. Run `openspec archive <name>` for each landed proposal. Move partially-landed remainders into a single new `active-rollover` proposal so no work is silently dropped.

**Alternatives considered:**

- *Rewrite each proposal in place to mark sections as "✅ DONE"*: would clutter the proposal file with status markers and leave the change directory still misleading.
- *Delete proposals outright*: loses historical context and breaks the OpenSpec archive convention.
- *One large rollover proposal absorbing every remainder*: would mix unrelated concerns; we keep `active-rollover` only if the partial set is genuinely related, otherwise emit one rollover per coherent group.

**Rationale**: The OpenSpec tooling already supports an archive workflow; using it preserves the project's process discipline rather than inventing a one-off.

### D8. Establish `proposal-lifecycle-hygiene` as a capability spec, not a doc

**Decision**: The hygiene rules (when to archive, how to mark active, what `audit.md` must contain) become a capability spec under `openspec/specs/proposal-lifecycle-hygiene/spec.md` with formal SHALL/MUST requirements and testable scenarios.

**Alternatives considered:**

- *CONTRIBUTING.md prose section*: easy to ignore; doesn't show up in spec validation.
- *No formalisation, rely on this change as one-off cleanup*: virtually guaranteed to drift again within 6 months.

**Rationale**: Encoding it as a spec makes future drift visible (the spec gets violated, validation surfaces it) rather than relying on tribal memory.

## Risks / Trade-offs

| Risk | Severity | Mitigation |
|---|---|---|
| LLM nondeterminism causes E2E flakes | High | D3 (invariants only) + per-test 90 s timeout + automatic 1 retry on transient failure (network, daemon-not-ready) but **never** on assertion failure |
| Ollama install fails in GitHub Actions runner | Medium | D4 (non-blocking workflow) + explicit prerequisite step that skips the job rather than failing if install fails + cache restore for the model layer to keep first-run cost bounded |
| `qwen3:0.5b` is too small to call tools reliably | Medium | Pin model + version in one constant; if first 30 days show <90% green rate, escalate to `qwen3:1.7b` (still under 1.5 GB) — explicit review gate |
| Track E misclassifies a partially-landed proposal as fully landed | Medium | `audit.md` evidence requirement (commit SHAs per task) makes the misclassification reviewable; reviewer can override before archival |
| Adding `tests/e2e/` as a separate Go module fragments contributor mental model | Low | `tests/e2e/README.md` + a single `make e2e` target that hides the cross-module invocation |
| E2E workflow caches grow unbounded over time | Low | Use GitHub's standard cache eviction (7-day idle) + scoped cache key including the Ollama model SHA |
| `proposal-lifecycle-hygiene` spec adds maintenance burden | Low | Spec is short (≤ 5 requirements) and aligned with the `openspec` tooling that already exists |

## Migration Plan

This change is additive — there is no migration in the deploy sense. The rollout is:

1. **Land Track A first** (E2E harness + workflow + transcripts) on its own commit set so the PR is reviewable in isolation. CI must stay green.
2. **Land Track E second** (audit + archive + hygiene spec) on a separate commit set so the archive moves are reviewable separately from the new test code.
3. After both tracks land, run `openspec validate` across the whole repo to confirm no orphan references.
4. Update `README.md` proof-asset slot to reference the most recent E2E artifact URL.

**Rollback strategy:**

- *Track A rollback*: delete `tests/e2e/` and `.github/workflows/e2e.yml`. No production code is touched, so rollback is mechanical.
- *Track E rollback*: revert the archive moves with `git mv` and delete `audit.md`. Archived files are still in git history.

## Open Questions

1. **Should `active-rollover` be created unconditionally, or only if at least one proposal turns out partially-landed?** Default to *only if needed* — empty rollover proposals are noise. Decision deferred until audit results are in.
2. **Do we want to add a Termux/Android E2E lane in this change?** Default to no (out of scope per Non-Goals); revisit when Linux E2E has 30 days of green history.
3. **Should transcripts be redacted of any tokens that resemble API keys before artifact upload?** Default to yes — add a simple regex scrub in the transcript helper, fail-safe (drop the line entirely if uncertain). Implementation detail for tasks.md.
