// Package e2e holds black-box, end-to-end tests that exercise the real
// Golem binary against a real local LLM (Ollama) and a real database.
//
// Architectural rule: this module MUST NOT import any package whose
// path begins with "github.com/strings77wzq/golem/" other than through
// the compiled binary surface (os/exec, HTTP, JSON-RPC over stdio).
// Importing core/, feature/, internal/, foundation/, or cmd/ packages
// directly would defeat the black-box guarantee that makes E2E tests
// meaningful and would violate the layer-purity rules in AGENTS.md §3.
//
// Tests in this module assert behavioural invariants only — tool names
// invoked, JSON-RPC method names, HTTP status codes, exit codes,
// presence/absence of safety-gate error categories — and never assert
// literal LLM-generated text. See specs/e2e-test-harness/spec.md,
// requirement "Behavioural invariants, not literal text".
//
// When Ollama is not available locally (or the required model is not
// pulled), every test in this module skips cleanly via t.Skip with an
// actionable message. The overall test run remains green so that
// contributors without Ollama installed are never blocked.
package e2e
