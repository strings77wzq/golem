# Design: Feature Layer Metrics

## Architecture

```
cmd/golem/main.go
  └── buildToolRegistry()
        ├── core tools (direct registration)
        └── feature tools (wrapped with metricsToolWrapper)
              ├── rag_retrieve → wrapper → registry
              ├── memory_*    → wrapper → registry
              ├── mcp_*       → wrapper → registry
              └── skills_*    → wrapper → registry
```

## metricsToolWrapper

Implements `tools.Tool` interface. Delegates to inner tool, records metrics before/after.

```go
type metricsToolWrapper struct {
    inner   tools.Tool
    calls   *metrics.Counter
    latency *metrics.Histogram
    errors  *metrics.Counter
}
```

- `Name()`, `Description()`, `Parameters()` delegate to inner
- `Execute()` records Inc/Observe/Inc around inner.Execute()

## Metric Naming Convention

```
feature_{module}_{action}_total     (Counter)
feature_{module}_{action}_latency   (Histogram)
feature_{module}_{action}_errors    (Counter)
```

| Module | Actions |
|--------|---------|
| rag | retrieve |
| memory | recall, store |
| mcp | tool_call |
| skills | execute |

## Registration Pattern

In `cmd/golem/` adapters, after building feature tools, wrap them:

```go
// Before:
registry.Register(ragTool)

// After:
registry.Register(newMetricsTool(ragTool, "rag", "retrieve"))
```

`newMetricsTool` auto-registers metrics in `feature/metrics.DefaultRegistry`.

## Testing Strategy

- Unit test `metricsToolWrapper.Execute()` — verify counter increment, histogram observation, error counting
- Integration: `/metrics` endpoint includes `feature_*` metrics after tool calls
