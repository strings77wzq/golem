package agent

import (
	"github.com/strings77wzq/golem/foundation/metrics"
)

// Agent-level metrics for observability.
var (
	// LLM metrics
	AgentLLMCalls   *metrics.Counter
	AgentLLMLatency *metrics.Histogram
	AgentLLMTokens  *metrics.Counter
	AgentLLMErrors  *metrics.Counter

	// Tool metrics
	AgentToolCalls   *metrics.Counter
	AgentToolLatency *metrics.Histogram
	AgentToolErrors  *metrics.Counter

	// Plan metrics
	AgentPlanSteps     *metrics.Gauge
	AgentPlanRevisions *metrics.Counter
	AgentPlanDuration  *metrics.Histogram

	// Context metrics
	AgentContextTokens      *metrics.Gauge
	AgentContextCompression *metrics.Gauge

	// Session metrics
	AgentSessionsActive *metrics.Gauge
	AgentMessagesTotal  *metrics.Counter

	// Security metrics
	AgentSecurityGates  *metrics.Counter
	AgentSecurityDenied *metrics.Counter

	// Cost metrics
	AgentLLMCostUSD *metrics.Counter
)

func init() {
	reg := metrics.DefaultRegistry

	AgentLLMCalls = reg.NewCounter("agent_llm_calls_total", "Total number of LLM calls")
	AgentLLMLatency = reg.NewHistogram("agent_llm_latency_seconds", "LLM call latency in seconds", metrics.DefaultBuckets)
	AgentLLMTokens = reg.NewCounter("agent_llm_tokens_total", "Total tokens consumed by LLM calls")
	AgentLLMErrors = reg.NewCounter("agent_llm_errors_total", "Total LLM call errors")

	AgentToolCalls = reg.NewCounter("agent_tool_calls_total", "Total number of tool executions")
	AgentToolLatency = reg.NewHistogram("agent_tool_latency_seconds", "Tool execution latency in seconds", metrics.DefaultBuckets)
	AgentToolErrors = reg.NewCounter("agent_tool_errors_total", "Total tool execution errors")

	AgentPlanSteps = reg.NewGauge("agent_plan_steps", "Number of steps in current plan")
	AgentPlanRevisions = reg.NewCounter("agent_plan_revisions_total", "Total plan revisions")
	AgentPlanDuration = reg.NewHistogram("agent_plan_duration_seconds", "Plan execution duration", []float64{1, 5, 10, 30, 60, 120, 300})

	AgentContextTokens = reg.NewGauge("agent_context_tokens_used", "Tokens used in current context window")
	AgentContextCompression = reg.NewGauge("agent_context_compression_ratio", "Context compression ratio")

	AgentSessionsActive = reg.NewGauge("agent_sessions_active", "Number of active sessions")
	AgentMessagesTotal = reg.NewCounter("agent_messages_total", "Total messages processed")

	AgentSecurityGates = reg.NewCounter("agent_security_gates_triggered", "Total security gate checks")
	AgentSecurityDenied = reg.NewCounter("agent_security_operations_denied", "Total operations denied")

	AgentLLMCostUSD = reg.NewCounter("agent_llm_cost_usd_total", "Total estimated LLM cost in USD (scaled by 10000)")
}
