## Why

Golem v0.9.1 has 41 packages and 82.5% unit-test coverage with all CI green, but every public claim — "local-first", "SQL safety model", "natural language → real query", "MCP server" — is currently verified only by **mocked unit tests**, never by an end-to-end run of the actual binary against a real LLM and a real database. As soon as a real user points `golem agent` at a messy SQLite database with an Ollama-served local model, behaviors that the unit tests cannot catch become possible: the agent may fail to call `sql_query`, the SQL safety gate may not trigger on adversarial generations, MCP transport may break under streaming, or the README quickstart may simply not work. Simultaneously, eight OpenSpec proposals sit unarchived in `openspec/changes/` even though most of their work has already landed in `main`, creating onboarding noise for new contributors who cannot tell which proposals are alive. We close both gaps in one change: build an E2E test harness that converts product claims into automated behavioral evidence, and audit the proposal backlog so the spec directory reflects reality.

## What Changes

### Track A — End-to-end test harness

- Add a new top-level `tests/e2e/` Go module that builds the real `golem` binary and exercises it as a black-box subprocess against a live Ollama daemon (`qwen3:0.5b` model) and a temporary SQLite database.
- Provide lifecycle helpers (`tests/e2e/helpers/ollama.go`) that detect, optionally start, and health-check Ollama, plus skip the suite cleanly with a clear message when Ollama is unavailable so the harness never breaks core CI for contributors who have not installed it.
- Provide transcript capture (`tests/e2e/helpers/transcript.go`) that records each end-to-end run to `tests/e2e/transcripts/<test-name>.log` so README and docs can reference real captured runs as proof assets.
- Cover four behavioral contracts as separate test files, each asserting **invariants** (tool names called, ToolResult shape, exit codes, safety-gate error categories) rather than fragile text equality:
  - `agent_sql_query_test.go`: agent + `sql_query` tool happy path against a seeded demo DB.
  - `safety_gate_test.go`: WHERE-less DELETE/UPDATE and DROP attempts must be blocked by the security gate, not executed.
  - `mcp_protocol_test.go`: `golem mcp-server` over stdio answers `tools/list` and dispatches a `sql_query` call correctly.
  - `streaming_test.go`: `gateway` `/api/chat/stream` SSE delivers tokens incrementally for non-tool turns.
- Add a dedicated GitHub Actions workflow (`.github/workflows/e2e.yml`) that installs Ollama, pulls the small model once with caching, and runs the suite on PRs touching agent/provider/security paths or on a manual dispatch. This workflow is **non-blocking** for the main CI gate — it surfaces failures as a separate check so contributors without Ollama locally are not penalised for unrelated PRs.
- Document the harness in `tests/e2e/README.md` with the exact local invocation, skip behavior, and how to add a new behavioral case.

### Track E — Stale proposal archive sweep

- Audit each of the eight unarchived proposals (`phase1-quality-hardening`, `phase2-runtime-capabilities`, `phase3-production-readiness`, `phase4-open-source-growth`, `product-positioning-and-oss-excellence`, `wire-security-gates`, `fix-ci-vet-provider-health-types`, `project-review-and-roadmap`) by cross-referencing their `tasks.md` against `git log`, current code state, and `openspec/specs/archive/` to classify each into **landed**, **partially landed**, or **still active**.
- For **landed** proposals: archive via the existing `openspec archive` workflow into `openspec/specs/archive/` with the standard date prefix.
- For **partially landed** proposals: extract the genuinely remaining tasks into a single `openspec/changes/active-rollover/` follow-up proposal so each surviving task has explicit ownership, then archive the original.
- For **still active** proposals: keep in place but prepend a one-line `> Status: active — last updated YYYY-MM-DD` header so future contributors can tell at a glance.
- Produce `openspec/changes/e2e-harness-and-spec-cleanup/audit.md` recording the per-proposal verdict + evidence (commit SHAs, file paths) so the cleanup is reviewable, not opaque.

## Capabilities

### New Capabilities

- `e2e-test-harness`: defines the requirements for end-to-end behavioral testing of the Golem binary against a real local LLM and a real database, including how invariants are asserted, how lifecycle is managed, how the suite degrades when Ollama is absent, and how transcript artifacts are produced for documentation reuse.
- `proposal-lifecycle-hygiene`: defines the rules for keeping the OpenSpec proposal directory truthful — when a proposal must be archived, how partial completion is handled, and what status header active proposals must carry.

### Modified Capabilities

<!-- None. Track A introduces a new test surface above the existing capabilities; Track E only changes process documentation, not the requirements of any existing capability spec. -->

## Impact

### Code

- **New**: `tests/e2e/` (new top-level Go module, isolated from `core/` `feature/` `internal/` layers — black-box only).
- **New**: `.github/workflows/e2e.yml` GitHub Actions workflow, non-blocking, runs only on relevant path changes or manual dispatch.
- **New**: `tests/e2e/README.md` documentation.
- **New**: `openspec/changes/active-rollover/` (only if any proposal is partially-landed; may be empty/omitted otherwise).
- **Modified**: zero existing source files in `core/`, `feature/`, `internal/`, `foundation/`, `cmd/` are touched. `LLMProvider` and `StreamingProvider` interface signatures remain frozen per AGENTS.md §4.
- **Moved**: each landed proposal directory under `openspec/changes/` is moved to `openspec/specs/archive/<YYYY-MM-DD>-<name>/` per the existing archive convention.

### APIs

- No public API surface changes. Tests interact with the `golem` binary through documented CLI flags and the existing MCP/Gateway transports.

### Dependencies

- No new Go module dependencies (`tests/e2e/` is its own module to avoid polluting the main `go.mod`; it depends only on `os/exec`, `net/http`, `testing`, and the public `golem` CLI surface).
- New external runtime dependency for the E2E suite only: Ollama daemon. The main build, main test suite, and main CI remain CGO-free and self-contained.

### Build invariants

- `CGO_ENABLED=0` preserved.
- Layer dependency rules (cmd/ → internal/ → core/ → foundation/, feature/ via adapters) preserved — `tests/e2e/` is not in any of these layers.
- Bubble Tea isolation, no Alt+key, error wrapping rules — preserved (no TUI/library code is added).

### Risk

- **CI flakiness from real LLM nondeterminism**: mitigated by asserting only behavioral invariants (tool names, ToolResult shape, exit codes), never literal text. Each test has a deterministic timeout and explicit retry-on-flake budget.
- **Ollama install in CI runner**: mitigated by separate non-blocking workflow + caching the model layer; main CI is unaffected if the E2E job fails or is skipped.
- **Archive misclassification**: mitigated by `audit.md` evidence trail and by routing any partially-landed proposals into a `active-rollover` follow-up rather than silently dropping their remaining work.
