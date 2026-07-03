# Proposal: Feature Layer Metrics via Tool Wrapper Pattern

## Why

Only `core/agent/metrics.go` has observability instrumentation (16 metrics). Feature modules (RAG, Memory, MCP, Skills) have zero metrics — their latency, error rates, and throughput are invisible. When a RAG retrieval takes 5 seconds or an MCP tool fails, operators have no signal.

## What Changes

Add a `metricsToolWrapper` in `cmd/golem/` that decorates feature-provided tools with Counter/Histogram/Counter metrics at registration time. Zero changes to feature module code.

## Capabilities

### New Capabilities

- Feature tool metrics exposed at `/metrics` endpoint
- Per-feature latency histograms with percentile visibility
- Error rate monitoring for RAG/Memory/MCP/Skills tools

### Modified Capabilities

- Tool registration in `cmd/golem/` wraps feature tools with metrics decorator

## Impact

- Affected code: `cmd/golem/` (adapter layer only)
- New files: `cmd/golem/metrics-adapter.go`, `feature/metrics/metrics.go`
- Dependencies: none (uses existing `foundation/metrics/`)
- Breaking changes: none
