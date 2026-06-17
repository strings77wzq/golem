# `tests/e2e` — Black-box behavioural tests

This module holds end-to-end tests that exercise the **real `golem` binary** against a **real local LLM (Ollama)** and a **real SQLite database**. The goal is to convert headline product claims from the README into automated, repeatable behavioural evidence.

> **Black-box rule.** This module MUST NOT import any package from the main project (`github.com/strings77wzq/golem/...`). Tests interact with Golem through its compiled binary, its CLI flags, its MCP stdio transport, and its HTTP gateway. See [`doc.go`](./doc.go) and the capability spec at [`openspec/specs/e2e-test-harness/spec.md`](../../openspec/specs/e2e-test-harness/spec.md) (after archive) or [the in-flight version](../../openspec/changes/e2e-harness-and-spec-cleanup/specs/e2e-test-harness/spec.md).

## Why a separate module?

`tests/e2e/go.mod` is its own Go module so that:

- Tests **cannot** accidentally `import "github.com/strings77wzq/golem/core/..."` and lose their black-box property.
- The main `go.mod` and `go.sum` stay clean — none of the E2E test machinery pollutes the production dependency graph.
- The E2E module can ship with **only** the standard library as dependencies, making the build trivially reproducible.

Layer rules from `AGENTS.md §3` are preserved automatically because the main module never imports this directory either.

## Prerequisites

1. **Go 1.25+** (matches the main module).
2. **Ollama** installed and running on `http://localhost:11434`.
   - Linux: `curl -fsSL https://ollama.com/install.sh | sh && ollama serve &`
   - macOS: `brew install ollama && ollama serve &`
   - Windows / Termux: see <https://ollama.com/download>
3. **The `qwen3:0.5b` model** pulled locally:
   ```bash
   ollama pull qwen3:0.5b
   ```

If any of those is missing, tests in this module **skip cleanly** with an actionable message — they do not fail. This is by design (see [the spec scenario "Graceful degradation when Ollama is absent"](../../openspec/changes/e2e-harness-and-spec-cleanup/specs/e2e-test-harness/spec.md)) so that contributors who have not installed Ollama are never blocked from merging unrelated PRs.

## Running locally

From the repository root:

```bash
make e2e
```

This target builds the `golem` binary with `CGO_ENABLED=0` and then runs `go test ./...` from inside the `tests/e2e` module. The `Makefile` target hides the cross-module invocation so contributors do not have to think about it.

Direct invocation (equivalent):

```bash
CGO_ENABLED=0 go build -o ./build/golem ./cmd/golem
cd tests/e2e && go test ./...
```

## Skip behaviour

| Condition                                        | What you see                                                          | Exit code |
| ------------------------------------------------ | --------------------------------------------------------------------- | --------- |
| Ollama daemon is not reachable                   | `--- SKIP: TestX (...) ollama not running at http://localhost:11434`  | 0         |
| `qwen3:0.5b` is not pulled                       | `--- SKIP: TestX (...) model qwen3:0.5b not pulled — run: ollama pull qwen3:0.5b` | 0         |
| Both prerequisites met but a behavioural test fails | `--- FAIL: TestX (...) <invariant violated>`                       | 1         |

A skipped test **never** counts as a failure. `make e2e` is safe to run on any machine.

## CI

The `.github/workflows/e2e.yml` workflow runs this suite on:

- Pull requests that touch agent / provider / tools / security / gateway / mcp / `tests/e2e/` paths, **and**
- Manual `workflow_dispatch` runs.

The workflow is **non-blocking** — it is not in the protected-branch required-checks list. Failures are visible as a separate check; they do not block merges. Once the suite has 30 days of green history, branch protection can be upgraded.

The main `.github/workflows/ci.yml` workflow is **unaffected** — it does not install Ollama and does not run anything from this directory.

## Adding a new behavioural test

1. Decide which product claim you want to convert into automated evidence. If it is not yet listed in the spec under "Behavioural coverage of headline product claims", open a separate proposal first.
2. Create `tests/e2e/<feature>_test.go`.
3. In `TestMain` or each test's first line, call `helpers.SkipIfUnavailable(t, "http://localhost:11434", "qwen3:0.5b")`.
4. Build the binary with `helpers.BuildGolem(t)` and seed any required database with `helpers.SeedDemoDB(t)`.
5. Drive the binary as a subprocess. **Assert behavioural invariants** — tool names, exit codes, JSON-RPC method names, HTTP statuses, error category strings — not LLM-generated text. See spec scenario "Forbidden assertion patterns".
6. Capture transcripts via `helpers.NewTranscript(t.Name())`. The helper redacts secrets automatically.

## Transcripts

Each test writes a transcript to `tests/e2e/transcripts/<test-name>.log`. These files are:

- Useful for debugging failing tests locally.
- Uploaded as a CI artifact by the E2E workflow.
- **Never** committed to git — see `.gitignore`.
- Automatically scrubbed for any string matching a known API-key shape (`Bearer <token>`, `sk-<token>`, etc.) by the redactor in `helpers/redact.go`.

The README of the main project links to the most recent CI artifact as a "proof of life" for the user-facing claims.

## What this suite is not

- **Not a unit-test replacement.** Unit tests stay the contract for individual components and remain at ≥ 80% coverage.
- **Not a benchmark suite.** Performance assertions live elsewhere (future work).
- **Not a snapshot test.** LLM outputs are nondeterministic; we never assert literal text.
- **Not a multi-platform suite (yet).** Linux amd64 only for now; macOS / Windows / Termux ARM64 lanes will come in a future change once the Linux lane has 30 days of green history.
