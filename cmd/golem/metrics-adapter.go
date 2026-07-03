package main

import (
	"context"
	"time"

	"github.com/strings77wzq/golem/core/tools"
	featuremetrics "github.com/strings77wzq/golem/feature/metrics"
	"github.com/strings77wzq/golem/foundation/metrics"
)

// metricsToolWrapper decorates a tools.Tool with latency/error/call metrics.
type metricsToolWrapper struct {
	inner   tools.Tool
	calls   *metrics.Counter
	latency *metrics.Histogram
	errors  *metrics.Counter
}

// newMetricsTool wraps a tool with auto-registered metrics based on its name prefix.
// Returns the original tool unchanged if it doesn't match a known feature prefix.
func newMetricsTool(tool tools.Tool) tools.Tool {
	m := featuremetrics.LookupByToolName(tool.Name())
	if m == nil {
		return tool
	}
	return &metricsToolWrapper{
		inner:   tool,
		calls:   m.Calls,
		latency: m.Latency,
		errors:  m.Errors,
	}
}

func (w *metricsToolWrapper) Name() string                      { return w.inner.Name() }
func (w *metricsToolWrapper) Description() string               { return w.inner.Description() }
func (w *metricsToolWrapper) Parameters() []tools.ToolParameter { return w.inner.Parameters() }

func (w *metricsToolWrapper) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	w.calls.Inc()
	start := time.Now()
	result, err := w.inner.Execute(ctx, args)
	w.latency.Observe(time.Since(start).Seconds())
	if err != nil || (result != nil && result.IsError) {
		w.errors.Inc()
	}
	return result, err
}
