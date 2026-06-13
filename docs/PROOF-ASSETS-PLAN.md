# Proof Assets Plan

This plan organizes benchmark/comparison/showcase content sources.

## Benchmark

- Source candidates:
  - existing Go benchmarks in `internal/gateway/benchmark_test.go`
  - latency and throughput snapshots from test runs
- Output target:
  - `docs/BENCHMARK.md`
  - one chart-ready data table

## Comparison

- Positioning dimensions:
  - runtime model (pure Go single binary)
  - platform support (Linux/Termux)
  - interaction modes (CLI/TUI/Gateway)
  - extension model (MCP/RAG/skills)
- Output target:
  - comparison table in README/docs

## Showcase

- Use-case categories:
  - coding assistant workflows
  - docs summarization workflows
  - local RAG support workflows
- Output target:
  - `docs/SHOWCASE.md`
  - 3-5 reproducible scenario cards
