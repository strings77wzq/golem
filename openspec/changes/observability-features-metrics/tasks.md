# Tasks: Feature Layer Metrics

## Task 1: Create feature metrics registry

- [ ] Create `feature/metrics/metrics.go` with `DefaultRegistry` and all 10 metric definitions
- [ ] Unit test: verify all metrics register and increment correctly

## Task 2: Implement metricsToolWrapper

- [ ] Create `cmd/golem/metrics-adapter.go` with `metricsToolWrapper` implementing `tools.Tool`
- [ ] Unit test: wrap mock tool, verify calls/latency/errors recorded
- [ ] Unit test: verify Name/Description/Parameters delegate correctly

## Task 3: Wire wrapper into tool registration

- [ ] Modify `cmd/golem/main.go` `buildToolRegistry()` to wrap feature tools with `newMetricsTool()`
- [ ] Integration: verify `/metrics` endpoint includes `feature_*` metrics

## Task 4: Update documentation

- [ ] Update `AGENTS.md` metrics section with new feature metrics
- [ ] Update `CHANGELOG.md`
