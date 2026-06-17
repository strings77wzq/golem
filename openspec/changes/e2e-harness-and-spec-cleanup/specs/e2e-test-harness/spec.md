## ADDED Requirements

### Requirement: Black-box test execution against the real Golem binary

The E2E test harness SHALL execute the production `golem` binary as a subprocess and assert its observable behaviour through CLI flags, stdin/stdout, exit codes, and HTTP/stdio transports — never by importing internal Go packages of the project under test.

#### Scenario: Build and invoke the real binary

- **WHEN** an E2E test runs
- **THEN** it MUST first compile `cmd/golem` to a temporary binary path with `CGO_ENABLED=0`
- **AND** it MUST invoke the binary as a subprocess via `os/exec`
- **AND** it MUST NOT import any package whose path begins with `github.com/strings77wzq/golem/` other than through the binary surface

#### Scenario: Layer purity preserved

- **WHEN** the E2E module is inspected
- **THEN** `tests/e2e/go.mod` SHALL be a separate Go module from the main project module
- **AND** its only dependencies on the project SHALL be on the compiled `golem` binary

### Requirement: Behavioural invariants, not literal text

E2E tests SHALL assert structural and behavioural properties (tool names invoked, JSON-RPC method names, exit codes, HTTP status codes, presence/absence of safety-gate error categories) and SHALL NOT assert exact LLM-generated text content.

#### Scenario: Successful tool invocation

- **WHEN** an E2E test verifies that the agent answered a database question
- **THEN** it SHALL assert the agent invoked the `sql_query` tool at least once
- **AND** it SHALL assert the resulting `ToolResult.ForLLM` field contains valid SQL result rows
- **AND** it SHALL NOT assert any specific natural-language phrasing in the agent's reply

#### Scenario: Forbidden assertion patterns

- **WHEN** a contributor proposes an E2E test
- **THEN** the test MUST NOT contain `strings.Contains` assertions over LLM-authored response text
- **AND** the test MUST NOT use snapshot files of LLM output

### Requirement: Graceful degradation when Ollama is absent

The E2E suite SHALL detect Ollama availability at startup and SHALL skip cleanly with an actionable message when Ollama is not running or the required model is not pulled, rather than failing the test run.

#### Scenario: Ollama daemon not running

- **WHEN** the E2E suite starts and `GET http://localhost:11434/api/tags` returns connection refused
- **THEN** every test in the suite SHALL call `t.Skip` with the message `"ollama not running at http://localhost:11434 — see tests/e2e/README.md"`
- **AND** the overall `go test` exit code SHALL be 0 (success)

#### Scenario: Required model not pulled

- **WHEN** the E2E suite starts and Ollama is reachable but `qwen3:0.5b` is not in the model list
- **THEN** every test SHALL call `t.Skip` with the message `"model qwen3:0.5b not pulled — run: ollama pull qwen3:0.5b"`
- **AND** the overall `go test` exit code SHALL be 0 (success)

#### Scenario: CI mode manages Ollama explicitly

- **WHEN** the environment variable `GOLEM_E2E_MANAGE_OLLAMA=1` is set
- **THEN** the harness MAY start `ollama serve` as a subprocess before the suite runs
- **AND** it MUST stop the subprocess in a `t.Cleanup` regardless of test outcome

### Requirement: Behavioural coverage of headline product claims

The E2E suite SHALL provide at least one passing test for each of the following product claims that are referenced in the README or AGENTS.md: agent + sql_query happy path, SQL safety gate enforcement, MCP server protocol, and gateway streaming.

#### Scenario: agent + sql_query happy path

- **WHEN** `golem agent --provider ollama --model qwen3:0.5b --db <demo.db>` is invoked with a natural-language database question
- **THEN** the test SHALL assert the agent invoked `sql_query` at least once
- **AND** the test SHALL assert the binary exited with code 0
- **AND** the test SHALL assert the ToolResult contained at least one row matching the seeded fixture

#### Scenario: Safety gate blocks WHERE-less DELETE

- **WHEN** the agent is prompted with `"delete every row in the users table"` against a writable demo database
- **THEN** the test SHALL assert the security gate produced an error matching the documented WHERE-required category
- **AND** the test SHALL assert the demo database row count is unchanged after the run

#### Scenario: MCP server tools/list and tools/call

- **WHEN** `golem mcp-server --db <demo.db>` is invoked over stdio with a JSON-RPC `tools/list` request
- **THEN** the test SHALL assert the response includes a tool whose name is `sql_query`
- **AND** a subsequent `tools/call` for `sql_query` with a valid SELECT SHALL return a non-empty content array

#### Scenario: Gateway SSE stream emits tokens

- **WHEN** the gateway is started and a request to `POST /api/chat/stream` is made for a non-tool prompt
- **THEN** the test SHALL assert at least two distinct SSE `data:` events are received
- **AND** the test SHALL assert the response stream terminates with the documented end-of-stream marker

### Requirement: Transcript artifacts for proof and debugging

For each E2E test, the harness SHALL capture a transcript of the binary's stdout, stderr, and key structured events to a file under `tests/e2e/transcripts/<test_name>.log`, suitable for upload as a CI artifact and reuse as documentation evidence.

#### Scenario: Transcript file produced per test

- **WHEN** an E2E test completes (pass or fail)
- **THEN** a file at `tests/e2e/transcripts/<sanitised_test_name>.log` SHALL exist
- **AND** the file SHALL contain a header line with the test name and timestamp
- **AND** the file SHALL contain the captured stdout and stderr of the binary

#### Scenario: Secret redaction in transcripts

- **WHEN** the transcript helper writes a line that matches a known API-key shape (e.g., a string of 30+ alphanumerics following `Bearer ` or `sk-`)
- **THEN** the helper MUST replace the matched substring with `[REDACTED]` before writing
- **AND** if the helper is uncertain about a line containing secrets, it MUST drop the line entirely rather than risk leakage

### Requirement: Non-blocking CI integration

The E2E suite SHALL run in a dedicated GitHub Actions workflow that is NOT a required check on protected branches, so contributors without Ollama are never blocked from merging unrelated PRs.

#### Scenario: Workflow triggers only on relevant paths or manual dispatch

- **WHEN** a PR is opened
- **THEN** the E2E workflow SHALL run only if the PR touches files matching `core/agent/**`, `core/providers/**`, `core/tools/**`, `internal/security/**`, `internal/gateway/**`, `feature/mcp/**`, or `tests/e2e/**`
- **AND** the workflow SHALL also be runnable via `workflow_dispatch`
- **AND** failure of the E2E workflow SHALL NOT block merging by branch protection

#### Scenario: Main CI is unaffected

- **WHEN** the existing `.github/workflows/ci.yml` runs
- **THEN** no step in `ci.yml` SHALL depend on Ollama
- **AND** no step in `ci.yml` SHALL run any test from `tests/e2e/`

### Requirement: Local developer ergonomics

The E2E suite SHALL be runnable from a fresh clone with one documented command and SHALL have its prerequisites and skip behaviour documented in `tests/e2e/README.md`.

#### Scenario: Single-command invocation

- **WHEN** a developer runs `make e2e` from the repository root
- **THEN** the target SHALL build the `golem` binary with `CGO_ENABLED=0`
- **AND** SHALL invoke `go test ./tests/e2e/...` with the appropriate working directory and module flags
- **AND** SHALL exit 0 if all tests pass or skip cleanly

#### Scenario: README explains skip behaviour

- **WHEN** a developer reads `tests/e2e/README.md`
- **THEN** the document SHALL explain how to install Ollama, how to pull the model, and what skip messages mean
- **AND** SHALL state that skipped tests do not count as failures
