# Tasks — e2e-harness-and-spec-cleanup

> **TDD discipline**: every implementation task is preceded by its failing test task. Never mark an implementation task complete unless its test was red first, then green, with no edits to the test in between.
>
> **Scope boundaries**: each task declares the file paths it is allowed to touch. `git diff --name-only HEAD` MUST stay inside the declared paths before commit.
>
> **Slice order**: complete Slice A1 → A2 → A3 → A4 → A5 (Track A) and Slice E1 → E2 → E3 (Track E) in order. The two tracks are independent; they may interleave commits but a slice MUST land atomically.

## 1. Track A — Slice A1: E2E module skeleton

**Allowed paths**: `tests/e2e/go.mod`, `tests/e2e/go.sum`, `tests/e2e/README.md`, `tests/e2e/doc.go`, `tests/e2e/.gitignore`

- [x] 1.1 Create `tests/e2e/` directory at repo root.
- [x] 1.2 Initialise `tests/e2e/go.mod` declaring `module github.com/strings77wzq/golem/tests/e2e` with Go 1.25+; depend only on the standard library.
- [x] 1.3 Add `tests/e2e/doc.go` with a package comment explaining the module is a black-box test surface and MUST NOT import any `core/` `feature/` `internal/` `foundation/` package.
- [x] 1.4 Add `tests/e2e/.gitignore` for `transcripts/`, `bin/`, `*.db`.
- [x] 1.5 Write `tests/e2e/README.md` with: prerequisite commands (`brew install ollama || curl -fsSL https://ollama.com/install.sh | sh`, `ollama pull qwen3:0.5b`), invocation (`make e2e`), skip behaviour, how to add a new behavioural case, secret-scrub guarantee.
- [x] 1.6 Run `cd tests/e2e && go vet ./...` from the new module — MUST be clean.

## 2. Track A — Slice A2: Ollama lifecycle helper (TDD)

**Allowed paths**: `tests/e2e/helpers/ollama.go`, `tests/e2e/helpers/ollama_test.go`

- [x] 2.1 **(RED)** Write `tests/e2e/helpers/ollama_test.go::TestDetectOllama_NotRunning` — uses an `httptest` server that closes immediately, asserts the helper returns a sentinel `ErrOllamaUnavailable`. Run; MUST fail (no implementation yet).
- [x] 2.2 **(GREEN)** Implement `Detect(baseURL string) error` in `tests/e2e/helpers/ollama.go` to GET `/api/tags`, return `ErrOllamaUnavailable` on connection error or non-200 status.
- [x] 2.3 **(RED)** Write `TestDetectOllama_ModelMissing` — `httptest` returns `{"models":[]}`, helper called with required model `qwen3:0.5b`. Assert returns sentinel `ErrModelNotPulled`. MUST fail first.
- [x] 2.4 **(GREEN)** Extend `Detect` to parse the model list and return `ErrModelNotPulled` when the required model is absent.
- [x] 2.5 **(RED)** Write `TestDetectOllama_HappyPath` — `httptest` returns the required model in the list. Assert returns nil. MUST fail before implementation reaches the success branch.
- [x] 2.6 **(GREEN)** Make happy path return nil; ensure all three tests pass.
- [x] 2.7 **(RED)** Write `TestSkipIfUnavailable_Skips` — uses a fake `*testing.T` recorder; assert `t.Skip` was called with the documented message format when `Detect` returns an error.
- [x] 2.8 **(GREEN)** Implement `SkipIfUnavailable(t *testing.T, baseURL, model string)` that calls `Detect` and `t.Skipf` with the documented messages.
- [x] 2.9 **(REFACTOR)** Run `go test ./tests/e2e/helpers/...`, all green; extract any duplication; re-run.

## 3. Track A — Slice A3: Transcript helper (TDD)

**Allowed paths**: `tests/e2e/helpers/transcript.go`, `tests/e2e/helpers/transcript_test.go`, `tests/e2e/helpers/redact.go`, `tests/e2e/helpers/redact_test.go`

- [x] 3.1 **(RED)** Write `tests/e2e/helpers/redact_test.go::TestRedact_BearerToken` — feeds `"Authorization: Bearer sk-abcdefghijklmnopqrstuvwxyz1234"`, asserts output substitutes the token with `[REDACTED]`. MUST fail first.
- [x] 3.2 **(GREEN)** Implement `Redact(line string) string` in `redact.go` matching `Bearer [A-Za-z0-9_\-]{20,}` and `sk-[A-Za-z0-9]{20,}`, replacing with `[REDACTED]`.
- [x] 3.3 **(RED)** Add `TestRedact_DropUncertainLine` — feeds a line that contains a 60-char unbroken alphanumeric run not preceded by a known prefix; assert helper returns empty string (drop). MUST fail.
- [x] 3.4 **(GREEN)** Extend redactor with the conservative drop heuristic; all redact tests pass.
- [x] 3.5 **(RED)** Write `TestTranscript_FileCreated` — `New("foo")` then `Close()`, assert a file at `transcripts/foo.log` exists with a header line `# foo @ <RFC3339>`. MUST fail.
- [x] 3.6 **(GREEN)** Implement `Transcript` type in `transcript.go` with `New(name string) *Transcript`, `Write(p []byte) (int, error)`, `Close() error`. `Write` MUST pipe each line through `Redact`.
- [x] 3.7 **(RED)** Write `TestTranscript_RedactsOnWrite` — write a Bearer-token line, assert the file content shows `[REDACTED]`. MUST fail before integration of Redact into Write.
- [x] 3.8 **(GREEN)** Wire `Redact` into `Transcript.Write` per-line; all transcript tests pass.

## 4. Track A — Slice A4: First behavioural test — agent + sql_query

**Allowed paths**: `tests/e2e/agent_sql_query_test.go`, `tests/e2e/helpers/binary.go`, `tests/e2e/helpers/binary_test.go`, `tests/e2e/helpers/demo_db.go`, `tests/e2e/fixtures/seed.sql`

- [ ] 4.1 **(RED)** Write `tests/e2e/helpers/binary_test.go::TestBuildBinary_ProducesExecutable` — calls `BuildGolem(t)`, asserts the returned path exists, is executable, and `<path> --version` exits 0. MUST fail before implementation.
- [ ] 4.2 **(GREEN)** Implement `BuildGolem(t *testing.T) string` in `binary.go` that runs `go build` with `CGO_ENABLED=0` from the repo root and returns the binary path; cleans up on `t.Cleanup`.
- [ ] 4.3 Add `tests/e2e/fixtures/seed.sql` creating a `users` table with 5 rows so behavioural assertions have a known fixture.
- [ ] 4.4 Implement `tests/e2e/helpers/demo_db.go::SeedDemoDB(t *testing.T) string` that creates a temp SQLite DB, executes `seed.sql`, returns the path; cleanup in `t.Cleanup`.
- [ ] 4.5 **(RED)** Write `tests/e2e/agent_sql_query_test.go::TestAgent_SqlQueryHappyPath` — uses `SkipIfUnavailable`, `BuildGolem`, `SeedDemoDB`; runs `golem agent --provider ollama --model qwen3:0.5b --db <path> --json-events -m "How many rows are in the users table?"`; asserts via parsing structured `--json-events` output that (a) the agent invoked tool `sql_query` at least once, (b) exit code is 0, (c) the ToolResult shape contains rows. MUST fail first because `--json-events` may not exist yet.
- [ ] 4.6 If `--json-events` does not exist on the binary, **STOP and reassess**: prefer adding a minimal structured-events flag in a separate scoped task rather than parsing free-form stdout. Open question — see Slice A4.5 below.
- [ ] 4.6.1 (Conditional) Add `--json-events` flag to `cmd/golem agent` emitting one JSON event per tool call. Allowed paths: `cmd/golem/agent.go` (or equivalent), with paired unit test in `cmd/golem/agent_test.go`. RED-first: write `TestAgent_JsonEvents_EmitsToolEvent` before implementation.
- [ ] 4.7 **(GREEN)** With `--json-events` available, make `TestAgent_SqlQueryHappyPath` green by parsing events and asserting invariants per spec scenario "Successful tool invocation".
- [ ] 4.8 Capture transcript via the helper; assert transcript file exists and contains no Bearer tokens.

## 5. Track A — Slice A5: Remaining behavioural tests + CI + docs

**Allowed paths per sub-slice**: each test file + only its directly-needed helper extension.

### 5.a Safety gate

- [ ] 5.1 **(RED)** `tests/e2e/safety_gate_test.go::TestSafetyGate_BlocksDeleteWithoutWhere` — prompt forces a DELETE attempt; assert agent received an error event whose category matches the documented "WHERE required" string from `internal/security/`; assert demo DB row count is unchanged.
- [ ] 5.2 **(GREEN)** Iterate prompt and parsing until invariants hold consistently across 10 runs (manual repeat); commit when green.

### 5.b MCP server

- [ ] 5.3 **(RED)** `tests/e2e/mcp_protocol_test.go::TestMcp_ToolsListIncludesSqlQuery` — start `golem mcp-server --db <path>` over stdio, send JSON-RPC `tools/list`, parse response, assert `sql_query` is in the tool list.
- [ ] 5.4 **(GREEN)** Implement; commit when green.
- [ ] 5.5 **(RED)** `TestMcp_ToolsCallSqlQuery` — issue a `tools/call` for `sql_query` with a SELECT, assert non-empty content array.
- [ ] 5.6 **(GREEN)** Implement; commit when green.

### 5.c Gateway streaming

- [ ] 5.7 **(RED)** `tests/e2e/streaming_test.go::TestGateway_StreamEmitsTokens` — start `golem gateway` on a free port, POST to `/api/chat/stream` with a non-tool prompt, parse SSE events, assert ≥ 2 distinct `data:` events and a documented terminator event.
- [ ] 5.8 **(GREEN)** Implement; commit when green.

### 5.d CI workflow + Makefile + README proof slot

- [ ] 5.9 Write `.github/workflows/e2e.yml`: triggers `pull_request` with paths filter from spec scenario "Workflow triggers only on relevant paths or manual dispatch" + `workflow_dispatch`; installs Ollama; restores cached `~/.ollama/models` keyed on model SHA; pulls `qwen3:0.5b` if cache miss; runs `make e2e`; uploads `tests/e2e/transcripts/` as artifact.
- [ ] 5.10 Add `e2e:` target to `Makefile` that builds the binary and runs `cd tests/e2e && go test ./...`.
- [ ] 5.11 Confirm `.github/workflows/ci.yml` is unchanged — no Ollama, no `tests/e2e/` references — per spec scenario "Main CI is unaffected".
- [ ] 5.12 Add a "Proof of life" section to `README.md` linking to the most recent E2E artifact (badge or text link). Allowed paths: `README.md` only. Diff MUST be additive.
- [ ] 5.13 Run `golangci-lint run ./...` and `go test ./...` from the repo root — both MUST be green and unchanged from baseline.

## 6. Track E — Slice E1: Audit the eight in-flight proposals

**Allowed paths**: `openspec/changes/e2e-harness-and-spec-cleanup/audit.md`

- [x] 6.1 For each of the eight proposals (`phase1-quality-hardening`, `phase2-runtime-capabilities`, `phase3-production-readiness`, `phase4-open-source-growth`, `product-positioning-and-oss-excellence`, `wire-security-gates`, `fix-ci-vet-provider-health-types`, `project-review-and-roadmap`), read the full `tasks.md` and `proposal.md`.
- [x] 6.2 For each task in each proposal, run `git log --all --oneline --grep '<keyword>'` and inspect file paths in `git log -p` to determine if the task has landed.
- [x] 6.3 Classify each proposal as `landed` / `partially-landed` / `active`.
- [x] 6.4 Write `audit.md` per spec scenario "Audit document captures the verdict" — one section per proposal with name, verdict, evidence (commit SHAs or specific open tasks), and resulting action.
- [x] 6.5 Self-review: every `landed` verdict has at least one commit SHA per task; every `partially-landed` verdict lists the specific open tasks; every `active` verdict has a justification why work is still planned.

## 7. Track E — Slice E2: Execute archive + rollover

**Allowed paths**: `openspec/changes/<archived-proposal-name>/**` (moves), `openspec/specs/archive/<YYYY-MM-DD>-<name>/**` (destinations), `openspec/changes/active-rollover/**` (only if needed)

- [x] 7.1 For each `landed` proposal in `audit.md`, run `openspec archive <name>` (or the project-defined equivalent) and verify the move into `openspec/specs/archive/<YYYY-MM-DD>-<name>/`.
- [x] 7.2 For each `partially-landed` proposal, copy the still-open tasks into `openspec/changes/active-rollover/tasks.md`, with each task block prefixed by `<!-- inherited from <source-proposal> -->`. Write `openspec/changes/active-rollover/proposal.md` summarising the consolidation per spec scenario "Partial proposal becomes archive + rollover". Then archive the source proposal as if it were landed.
- [x] 7.3 If no proposal is partially-landed, `active-rollover/` MUST NOT be created (no empty rollover).
- [x] 7.4 For each remaining `active` proposal, prepend the line `> Status: active — last reviewed <YYYY-MM-DD>` to its `proposal.md`. Allowed paths: only that proposal's `proposal.md`.
- [x] 7.5 Run `openspec validate` across the whole repo; MUST be clean. *(All changes/specs created or modified in this slice validate cleanly. 7 pre-existing specs missing `## Purpose` (`agent`, `chunker-loop-fix`, `golem-enhancement`, `p0-wire-components`, `p1-auto-compact-registry-refactor`, untracked `critical-bug-fixes`, untracked `test-coverage-improvement`) predate this slice and lie outside its allowed paths; documented as a follow-up rather than fixed in-scope.)*

## 8. Track E — Slice E3: Lock in the hygiene spec

**Allowed paths**: `openspec/specs/proposal-lifecycle-hygiene/spec.md` (new, copied from change-set spec at archive time), `CONTRIBUTING.md`

- [ ] 8.1 At archive time of this change, the `proposal-lifecycle-hygiene` spec from `openspec/changes/e2e-harness-and-spec-cleanup/specs/proposal-lifecycle-hygiene/spec.md` SHALL be promoted to `openspec/specs/proposal-lifecycle-hygiene/spec.md`. (This is automatic via `openspec archive` for a spec-driven change.) Verify the promotion landed. *(Cannot complete while this change is in flight. The source spec exists and validates; promotion happens automatically when this change is archived.)*
- [x] 8.2 Add a one-paragraph reference to `CONTRIBUTING.md` pointing maintainers at `openspec/specs/proposal-lifecycle-hygiene/spec.md` for cleanup-pass procedure. Allowed paths: `CONTRIBUTING.md` only. Diff MUST be additive.

## 9. Verification gate

**Allowed paths**: none (read-only verification). Any code changes here open a new task.

- [ ] 9.1 `cd tests/e2e && go vet ./...` clean.
- [ ] 9.2 `make e2e` from repo root: with Ollama running and `qwen3:0.5b` pulled, all four behavioural tests pass and produce transcripts.
- [ ] 9.3 `make e2e` with Ollama NOT running: every test skips cleanly, exit code 0.
- [ ] 9.4 Repo-root `go test ./...` green and unchanged from baseline (no new failures, no slowdowns > 10%).
- [ ] 9.5 Repo-root `golangci-lint run ./...` green.
- [ ] 9.6 `git diff --name-only main...HEAD` confined to the union of all task-declared allowed paths above.
- [ ] 9.7 `openspec validate` across the repo clean.
- [ ] 9.8 Manual smoke: open `tests/e2e/README.md` from a fresh shell, follow it verbatim, confirm it works.
- [ ] 9.9 Push branch; observe `e2e.yml` workflow run in CI (or skip cleanly if Ollama install fails) and confirm `ci.yml` is green.

## 10. Atomic commit ordering (per AGENTS.md & rules/common/git-workflow.md)

- [ ] 10.1 Commit Slice A1 as `test: scaffold tests/e2e module`.
- [ ] 10.2 Commit Slice A2 as `test: add ollama lifecycle helper with TDD`.
- [ ] 10.3 Commit Slice A3 as `test: add transcript helper with secret redaction`.
- [ ] 10.4 If Slice A4 added `--json-events`, commit as `feat: add --json-events flag for E2E observability` BEFORE the test commit.
- [ ] 10.5 Commit Slice A4 as `test: e2e — agent + sql_query happy path`.
- [ ] 10.6 Commit each sub-slice of A5 as its own `test:` commit; CI workflow + Makefile + README as `ci:` and `docs:` respectively.
- [ ] 10.7 Commit Slice E1 as `docs: audit unarchived openspec proposals`.
- [ ] 10.8 Commit Slice E2 as `chore: archive landed openspec proposals` (one commit per archive move is fine if reviewer prefers).
- [ ] 10.9 Commit Slice E3 as `docs: promote proposal-lifecycle-hygiene to specs`.
- [ ] 10.10 Push and run `./scripts/ci-monitor.sh` per CLAUDE.md CI Monitoring Protocol; do not declare done until CI is green.
