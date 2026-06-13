package context

import "fmt"

// TokenBudget manages token allocation across system prompt, tool definitions,
// and conversation history. Ratios determine what fraction of the total
// context window each component gets.
type TokenBudget struct {
	TotalTokens   int
	SystemRatio   float64
	ToolsRatio    float64
	HistoryRatio  float64

	SystemTokens  int
	ToolsTokens   int
	HistoryTokens int
}

// Compute calculates the token allocation from ratios.
func (b *TokenBudget) Compute() {
	b.SystemTokens = int(float64(b.TotalTokens) * b.SystemRatio)
	b.ToolsTokens = int(float64(b.TotalTokens) * b.ToolsRatio)
	b.HistoryTokens = int(float64(b.TotalTokens) * b.HistoryRatio)
}

// String returns a human-readable budget summary.
func (b *TokenBudget) String() string {
	return fmt.Sprintf("total=%d system=%d(%.0f%%) tools=%d(%.0f%%) history=%d(%.0f%%)",
		b.TotalTokens,
		b.SystemTokens, b.SystemRatio*100,
		b.ToolsTokens, b.ToolsRatio*100,
		b.HistoryTokens, b.HistoryRatio*100)
}

// WithSystemRatio sets the system prompt ratio and recomputes.
func (b *TokenBudget) WithSystemRatio(r float64) *TokenBudget {
	b.SystemRatio = r
	b.Compute()
	return b
}

// WithToolsRatio sets the tools ratio and recomputes.
func (b *TokenBudget) WithToolsRatio(r float64) *TokenBudget {
	b.ToolsRatio = r
	b.Compute()
	return b
}

// WithHistoryRatio sets the history ratio and recomputes.
func (b *TokenBudget) WithHistoryRatio(r float64) *TokenBudget {
	b.HistoryRatio = r
	b.Compute()
	return b
}
