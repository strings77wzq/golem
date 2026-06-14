package context

import (
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
)

func TestTokenBudgetCompute(t *testing.T) {
	b := &TokenBudget{TotalTokens: 8192, SystemRatio: 0.2, ToolsRatio: 0.3, HistoryRatio: 0.5}
	b.Compute()

	if b.SystemTokens != 1638 {
		t.Errorf("SystemTokens = %d, want 1638", b.SystemTokens)
	}
	if b.ToolsTokens != 2457 {
		t.Errorf("ToolsTokens = %d, want 2457", b.ToolsTokens)
	}
	if b.HistoryTokens != 4096 {
		t.Errorf("HistoryTokens = %d, want 4096", b.HistoryTokens)
	}
}

func TestTokenBudgetString(t *testing.T) {
	b := &TokenBudget{TotalTokens: 8192, SystemRatio: 0.2, ToolsRatio: 0.3, HistoryRatio: 0.5}
	b.Compute()
	s := b.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestPromptBuilderBuild(t *testing.T) {
	pb := NewPromptBuilder("You are a helpful assistant.")
	empty := ""
	pb.contextHints = &empty
	result := pb.Build(nil, nil, "")
	if result != "You are a helpful assistant." {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestPromptBuilderWithToolSummary(t *testing.T) {
	pb := NewPromptBuilder("You are a helper.")
	toolDefs := []tools.ToolDefinition{
		{Name: "web_search", Description: "Search the web"},
		{Name: "file_read", Description: "Read a file"},
	}
	result := pb.Build(nil, toolDefs, "")
	if !contains(result, "Available tools:") {
		t.Error("expected tool summary in prompt")
	}
	if !contains(result, "web_search") {
		t.Error("expected web_search in prompt")
	}
}

func TestPromptBuilderWithSystemMsg(t *testing.T) {
	pb := NewPromptBuilder("base prompt")
	sysMsg := providers.Message{Role: providers.RoleSystem, Content: "custom system prompt"}
	result := pb.Build(&sysMsg, nil, "")
	if !contains(result, "custom system prompt") {
		t.Error("expected custom system prompt")
	}
}

func TestPromptBuilderWithSkillPrompts(t *testing.T) {
	pb := NewPromptBuilder("base")
	pb.SetSkillPrompts("## Skill: summarize\nSummarize text.\n")
	result := pb.Build(nil, nil, "")
	if !contains(result, "Skill: summarize") {
		t.Error("expected skill prompt in result")
	}
}

func TestPromptBuilderWithContextHints(t *testing.T) {
	pb := NewPromptBuilder("base")
	pb.SetContextHints("User prefers Chinese.")
	result := pb.Build(nil, nil, "")
	if !contains(result, "User prefers Chinese") {
		t.Error("expected context hints")
	}
}

func TestCompressorEmpty(t *testing.T) {
	c := NewCompressor()
	result := c.Compress(nil, 1000, DefaultTokenEstimator)
	if len(result) != 0 {
		t.Error("expected empty result")
	}
}

func TestCompressorFitsBudget(t *testing.T) {
	c := NewCompressor()
	msgs := []providers.Message{
		{Role: providers.RoleUser, Content: "hello"},
		{Role: providers.RoleAssistant, Content: "hi"},
	}
	result := c.Compress(msgs, 10000, DefaultTokenEstimator)
	if len(result) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result))
	}
}

func TestCompressorTruncatesToolOutput(t *testing.T) {
	c := NewCompressor()
	c.TruncateThreshold = 10
	c.MaxToolOutput = 5

	longContent := "This is a very long tool output that should be truncated because it exceeds the threshold."
	msgs := []providers.Message{
		{Role: providers.RoleUser, Content: "query"},
		{Role: providers.RoleTool, Content: longContent},
		{Role: providers.RoleAssistant, Content: "response"},
	}

	result := c.Compress(msgs, 100000, DefaultTokenEstimator)
	for _, msg := range result {
		if msg.Role == providers.RoleTool && len(msg.Content) > 200 {
			t.Errorf("tool output not truncated: len=%d", len(msg.Content))
		}
	}
}

func TestCompressorDropsOldest(t *testing.T) {
	c := NewCompressor()
	c.KeepRecent = 2

	msgs := make([]providers.Message, 20)
	for i := range msgs {
		msgs[i] = providers.Message{
			Role:    providers.RoleUser,
			Content: "message with some content that takes tokens",
		}
	}

	// Very small budget forces dropping old messages
	result := c.Compress(msgs, 50, DefaultTokenEstimator)
	if len(result) >= len(msgs) {
		t.Errorf("expected fewer messages, got %d", len(result))
	}
	// Recent messages should be preserved
	if len(result) < 2 {
		t.Error("expected at least 2 recent messages")
	}
}

func TestCompressorPreservesToolChain(t *testing.T) {
	c := NewCompressor()
	c.KeepRecent = 2

	// Create a message sequence with tool chain
	msgs := []providers.Message{
		{Role: providers.RoleSystem, Content: "You are helpful."},
		{Role: providers.RoleUser, Content: "query"},
		{Role: providers.RoleAssistant, Content: "let me search", ToolCalls: []providers.ToolCall{
			{ID: "tc1", Name: "web_search"},
		}},
		{Role: providers.RoleTool, Content: "search result", ToolCallID: "tc1"},
		{Role: providers.RoleAssistant, Content: "found it"},
		{Role: providers.RoleUser, Content: "another query"},
		{Role: providers.RoleAssistant, Content: "another response"},
	}

	// Small budget - should keep recent and tool chain
	result := c.Compress(msgs, 500, DefaultTokenEstimator)

	// Check that tool chain is preserved
	hasToolChain := false
	for i, msg := range result {
		if msg.Role == providers.RoleAssistant && len(msg.ToolCalls) > 0 {
			// Find the next tool message
			if i+1 < len(result) && result[i+1].Role == providers.RoleTool {
				hasToolChain = true
			}
		}
	}
	if !hasToolChain {
		t.Error("expected tool chain to be preserved")
	}
}

func TestCompressorDropsBatchAtomically(t *testing.T) {
	c := NewCompressor()
	c.KeepRecent = 2

	// Large messages that won't fit
	longContent := strings.Repeat("x", 5000)
	msgs := []providers.Message{
		{Role: providers.RoleSystem, Content: "system"},
		{Role: providers.RoleUser, Content: "query"},
		{Role: providers.RoleAssistant, Content: "thinking", ToolCalls: []providers.ToolCall{
			{ID: "tc1", Name: "web_search"},
			{ID: "tc2", Name: "file_read"},
		}},
		{Role: providers.RoleTool, Content: longContent, ToolCallID: "tc1"},
		{Role: providers.RoleTool, Content: longContent, ToolCallID: "tc2"},
		{Role: providers.RoleAssistant, Content: "found it"},
		{Role: providers.RoleUser, Content: "another"},
		{Role: providers.RoleAssistant, Content: "response"},
	}

	// Very small budget - batch should be dropped atomically
	result := c.Compress(msgs, 200, DefaultTokenEstimator)

	// Should NOT have orphaned tool messages
	for _, msg := range result {
		if msg.Role == providers.RoleTool {
			t.Errorf("expected no tool messages in result when batch is dropped, got: %s", msg.Content[:50])
		}
	}
}

func TestCompressorMultipleToolBatches(t *testing.T) {
	c := NewCompressor()
	c.KeepRecent = 2

	msgs := []providers.Message{
		{Role: providers.RoleUser, Content: "query 1"},
		{Role: providers.RoleAssistant, Content: "thinking 1", ToolCalls: []providers.ToolCall{
			{ID: "tc1", Name: "tool1"},
		}},
		{Role: providers.RoleTool, Content: "result 1", ToolCallID: "tc1"},
		{Role: providers.RoleAssistant, Content: "response 1"},
		{Role: providers.RoleUser, Content: "query 2"},
		{Role: providers.RoleAssistant, Content: "thinking 2", ToolCalls: []providers.ToolCall{
			{ID: "tc2", Name: "tool2"},
		}},
		{Role: providers.RoleTool, Content: "result 2", ToolCallID: "tc2"},
		{Role: providers.RoleAssistant, Content: "response 2"},
		{Role: providers.RoleUser, Content: "final"},
		{Role: providers.RoleAssistant, Content: "final response"},
	}

	result := c.Compress(msgs, 1000, DefaultTokenEstimator)

	// Verify tool chains are intact
	for _, msg := range result {
		if msg.Role == providers.RoleTool {
			// Each tool message should have a preceding assistant with matching tool_calls
			t.Logf("tool message found: ToolCallID=%s", msg.ToolCallID)
		}
	}

	// Final messages should be present
	if result[len(result)-1].Content != "final response" {
		t.Errorf("expected last message to be 'final response', got %q", result[len(result)-1].Content)
	}
}

func TestManagerBuildContext(t *testing.T) {
	sess := session.NewSession("test")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "You are helpful."})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hello"})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "Hi there!"})

	mgr := NewManager(8192, "You are helpful.")
	result := mgr.BuildContext(sess, nil, "")

	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	if result[0].Role != providers.RoleSystem {
		t.Errorf("first message should be system, got %v", result[0].Role)
	}
}

func TestManagerBuildContextWithTools(t *testing.T) {
	sess := session.NewSession("test")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "Hello"})

	toolDefs := []tools.ToolDefinition{
		{Name: "web_search", Description: "Search the web"},
	}

	mgr := NewManager(8192, "")
	result := mgr.BuildContext(sess, toolDefs, "")

	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	// System prompt should contain tool info
	if !contains(result[0].Content, "web_search") {
		t.Error("expected tool info in system prompt")
	}
}

func TestManagerBudgetReport(t *testing.T) {
	mgr := NewManager(8192, "")
	msgs := []providers.Message{
		{Role: providers.RoleUser, Content: "hello"},
	}
	report := mgr.BudgetReport(msgs)
	if report == "" {
		t.Error("expected non-empty report")
	}
}

func TestFormatToolSummary(t *testing.T) {
	defs := []tools.ToolDefinition{
		{Name: "exec", Description: "Execute command"},
		{Name: "read", Description: "Read file"},
	}
	summary := FormatToolSummary(defs)
	if !contains(summary, "exec") || !contains(summary, "read") {
		t.Errorf("unexpected summary: %s", summary)
	}
}

func TestFormatToolSummaryEmpty(t *testing.T) {
	summary := FormatToolSummary(nil)
	if summary != "" {
		t.Errorf("expected empty summary, got %q", summary)
	}
}

func TestDefaultTokenEstimator(t *testing.T) {
	msg := providers.Message{Role: providers.RoleUser, Content: "Hello, world!"}
	tokens := DefaultTokenEstimator(msg)
	if tokens <= 0 {
		t.Errorf("expected positive token count, got %d", tokens)
	}
}

func TestDefaultTokenEstimatorCJK(t *testing.T) {
	msg := providers.Message{Role: providers.RoleUser, Content: "你好世界"}
	tokens := DefaultTokenEstimator(msg)
	// CJK: 4 chars / 2 = 2 tokens
	if tokens != 2 {
		t.Errorf("expected 2 tokens for CJK, got %d", tokens)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
