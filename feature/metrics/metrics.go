// Package metrics defines Prometheus-compatible metrics for feature modules.
// Feature tools are wrapped at registration time in cmd/golem/ with zero
// changes to feature module code.
package metrics

import (
	"sync"

	"github.com/strings77wzq/golem/foundation/metrics"
)

// Feature-level metrics for observability.
var (
	// RAG metrics
	RAGRetrieveCalls   *metrics.Counter
	RAGRetrieveLatency *metrics.Histogram
	RAGRetrieveErrors  *metrics.Counter

	// Memory metrics
	MemoryRecallCalls   *metrics.Counter
	MemoryRecallLatency *metrics.Histogram
	MemoryRecallErrors  *metrics.Counter

	// MCP metrics
	MCPCalls   *metrics.Counter
	MCPLatency *metrics.Histogram
	MCPErrors  *metrics.Counter

	// Skills metrics
	SkillsExecuteCalls   *metrics.Counter
	SkillsExecuteLatency *metrics.Histogram
	SkillsExecuteErrors  *metrics.Counter
)

// ModuleMetrics holds the three standard metrics for a feature module.
type ModuleMetrics struct {
	Calls   *metrics.Counter
	Latency *metrics.Histogram
	Errors  *metrics.Counter
}

// byModule maps tool name prefixes to their metrics. Built lazily in ensureInit().
var byModule map[string]ModuleMetrics

var initOnce sync.Once

// ensureInit lazily registers all feature metrics and builds the prefix map.
// Called from LookupByToolName — safe for concurrent use.
func ensureInit() {
	initOnce.Do(func() {
		reg := metrics.DefaultRegistry

		RAGRetrieveCalls = reg.NewCounter("feature_rag_retrieve_total", "Total RAG retrieve calls")
		RAGRetrieveLatency = reg.NewHistogram("feature_rag_retrieve_latency_seconds", "RAG retrieve latency", metrics.DefaultBuckets)
		RAGRetrieveErrors = reg.NewCounter("feature_rag_retrieve_errors_total", "Total RAG retrieve errors")

		MemoryRecallCalls = reg.NewCounter("feature_memory_recall_total", "Total memory recall calls")
		MemoryRecallLatency = reg.NewHistogram("feature_memory_recall_latency_seconds", "Memory recall latency", metrics.DefaultBuckets)
		MemoryRecallErrors = reg.NewCounter("feature_memory_recall_errors_total", "Total memory recall errors")

		MCPCalls = reg.NewCounter("feature_mcp_tool_calls_total", "Total MCP tool calls")
		MCPLatency = reg.NewHistogram("feature_mcp_tool_latency_seconds", "MCP tool latency", metrics.DefaultBuckets)
		MCPErrors = reg.NewCounter("feature_mcp_tool_errors_total", "Total MCP tool errors")

		SkillsExecuteCalls = reg.NewCounter("feature_skills_execute_total", "Total skills execute calls")
		SkillsExecuteLatency = reg.NewHistogram("feature_skills_execute_latency_seconds", "Skills execute latency", metrics.DefaultBuckets)
		SkillsExecuteErrors = reg.NewCounter("feature_skills_execute_errors_total", "Total skills execute errors")

		byModule = map[string]ModuleMetrics{
			"rag_":        {RAGRetrieveCalls, RAGRetrieveLatency, RAGRetrieveErrors},
			"memory_":     {MemoryRecallCalls, MemoryRecallLatency, MemoryRecallErrors},
			"mcp_":        {MCPCalls, MCPLatency, MCPErrors},
			"skill_":      {SkillsExecuteCalls, SkillsExecuteLatency, SkillsExecuteErrors},
			"summarize_":  {SkillsExecuteCalls, SkillsExecuteLatency, SkillsExecuteErrors},
			"code_review": {SkillsExecuteCalls, SkillsExecuteLatency, SkillsExecuteErrors},
		}
	})
}

// LookupByToolName returns the metrics for a tool name, or nil if not a feature tool.
func LookupByToolName(toolName string) *ModuleMetrics {
	ensureInit()
	for prefix, m := range byModule {
		if len(toolName) >= len(prefix) && toolName[:len(prefix)] == prefix {
			return &m
		}
	}
	return nil
}
