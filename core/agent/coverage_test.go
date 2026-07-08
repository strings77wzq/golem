package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/planner"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/foundation/logger"
)

// --- HandleFork ---

func TestHandleFork(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)

	// Create original session with messages
	sess := session.NewSession("orig")
	sess.AddMessage(providers.Message{Role: providers.RoleSystem, Content: "system"})
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "hello"})
	sess.AddMessage(providers.Message{Role: providers.RoleAssistant, Content: "hi"})
	_ = a.sessionStore.Save(sess)

	// Fork at index 1 (keep system + user, drop assistant)
	newID, err := a.HandleFork(context.Background(), "orig", 1, "new question")
	if err != nil {
		t.Fatalf("HandleFork failed: %v", err)
	}
	if newID == "" {
		t.Fatal("expected non-empty session ID")
	}
	if newID == "orig" {
		t.Error("forked session should have different ID")
	}

	// Verify forked session has correct messages
	// Fork(upToIndex=1) copies messages[0:1] (system) + appends new message = 2 total
	forked, ok := a.sessionStore.Get(newID)
	if !ok {
		t.Fatal("forked session not found")
	}
	msgs := forked.GetMessages()
	if len(msgs) != 2 { // system + new message
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestHandleFork_SessionNotFound(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)

	_, err := a.HandleFork(context.Background(), "nonexistent", 0, "msg")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestHandleFork_IndexExceedsLength(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)

	sess := session.NewSession("short")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "hi"})
	_ = a.sessionStore.Save(sess)

	// Fork with index beyond message count — should clamp
	newID, err := a.HandleFork(context.Background(), "short", 999, "extra")
	if err != nil {
		t.Fatalf("HandleFork failed: %v", err)
	}
	forked, ok := a.sessionStore.Get(newID)
	if !ok {
		t.Fatal("forked session not found")
	}
	msgs := forked.GetMessages()
	if len(msgs) != 2 { // original message + new message
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

// --- HandleMessageWithEvents ---

func TestHandleMessageWithEvents(t *testing.T) {
	a, _, mockProvider, _ := setupTestAgent(t)

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "response with events",
	})

	var events []bus.OutboundMessage
	emit := func(out bus.OutboundMessage) {
		events = append(events, out)
	}

	resp, err := a.HandleMessageWithEvents(context.Background(), "evt-session", "test", emit)
	if err != nil {
		t.Fatalf("HandleMessageWithEvents failed: %v", err)
	}
	if resp != "response with events" {
		t.Errorf("expected 'response with events', got %q", resp)
	}
	if len(events) == 0 {
		t.Error("expected at least one emitted event")
	}
}

// --- summarizeMessages ---

func TestSummarizeMessages(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)

	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "What is Go?"},
		{Role: providers.RoleAssistant, Content: "Go is a programming language."},
		{Role: providers.RoleTool, Content: "tool result here"},
	}

	summary := a.summarizeMessages(messages)
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	if !strings.Contains(summary, "User asked") {
		t.Error("summary should contain 'User asked'")
	}
	if !strings.Contains(summary, "Go is a programming") {
		t.Error("summary should contain assistant response")
	}
	if !strings.Contains(summary, "Tool result") {
		t.Error("summary should contain tool result")
	}
}

func TestSummarizeMessages_LongContent(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)

	longContent := strings.Repeat("x", 300)
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: longContent},
	}

	summary := a.summarizeMessages(messages)
	// Should be truncated to 200 chars + "..."
	if len(summary) > 500 {
		t.Errorf("summary too long: %d chars", len(summary))
	}
}

func TestSummarizeMessages_ToolCallsOnly(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)

	messages := []providers.Message{
		{
			Role:      providers.RoleAssistant,
			Content:   "",
			ToolCalls: []providers.ToolCall{{Name: "sql_query"}, {Name: "web_search"}},
		},
	}

	summary := a.summarizeMessages(messages)
	if !strings.Contains(summary, "sql_query") {
		t.Error("summary should contain tool names")
	}
}

// --- getStepIndex ---

func TestGetStepIndex(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)

	plan := planner.NewPlan("test")
	plan.AddStep("first", "done", nil)
	plan.AddStep("second", "done", nil)
	plan.AddStep("third", "done", nil)

	tests := []struct {
		stepID string
		want   int
	}{
		{"step-1", 0},
		{"step-2", 1},
		{"step-3", 2},
		{"step-99", 0}, // not found returns 0
	}

	for _, tt := range tests {
		got := a.getStepIndex(plan, tt.stepID)
		if got != tt.want {
			t.Errorf("getStepIndex(%q) = %d, want %d", tt.stepID, got, tt.want)
		}
	}
}

// --- resolveProvider (router path) ---

func TestResolveProvider_WithRouter(t *testing.T) {
	a, _, mockProvider, _ := setupTestAgent(t)

	// Queue a response for the router's provider
	mockProvider.AddResponse(&providers.LLMResponse{Content: "router response"})

	// Set up a router
	router := &mockRouter{provider: mockProvider}
	a.router = router

	resp, provider, modelName, err := a.resolveProvider(
		context.Background(),
		"test-model",
		[]providers.Message{{Role: providers.RoleUser, Content: "hi"}},
		nil,
	)
	if err != nil {
		t.Fatalf("resolveProvider failed: %v", err)
	}
	if resp == nil {
		t.Error("expected non-nil response from router")
	}
	if provider != nil {
		t.Error("expected nil provider when router is set")
	}
	if modelName != "test-model" {
		t.Errorf("expected model 'test-model', got %q", modelName)
	}
}

func TestResolveProvider_WithoutRouter(t *testing.T) {
	a, _, mockProvider, _ := setupTestAgent(t)

	// No router set — should use factory
	_, provider, modelName, err := a.resolveProvider(
		context.Background(),
		"mock/test-model",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("resolveProvider failed: %v", err)
	}
	if provider == nil {
		t.Error("expected non-nil provider from factory")
	}
	_ = mockProvider // used indirectly via factory
	_ = modelName
}

// --- preProcess (planning disabled, no compactor) ---

func TestPreProcess_NoPlanning(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)
	a.planEnabled = false
	a.compactor = nil

	sess := session.NewSession("pre")
	sess.AddMessage(providers.Message{Role: providers.RoleUser, Content: "test"})

	content, usage, handled, err := a.preProcess(
		context.Background(),
		bus.InboundMessage{SessionID: "pre", Content: "test"},
		sess,
		"mock/test-model",
		nil,
		false, nil, nil,
	)
	if err != nil {
		t.Fatalf("preProcess failed: %v", err)
	}
	if handled {
		t.Error("expected handled=false when planning is disabled")
	}
	if content != "" {
		t.Errorf("expected empty content, got %q", content)
	}
	_ = usage
}

// --- buildToolErrorMessage ---

func TestBuildToolErrorMessage(t *testing.T) {
	a, _, _, registry := setupTestAgent(t)

	// Register a tool so the error message can list available tools
	registry.Register(&coverageMockTool{name: "sql_query"})

	tc := providers.ToolCall{
		Name:      "nonexistent_tool",
		Arguments: map[string]interface{}{"key": "value"},
	}

	msg := a.buildToolErrorMessage(tc, fmt.Errorf("tool not found"))
	if !strings.Contains(msg, "nonexistent_tool") {
		t.Error("error message should contain tool name")
	}
	if !strings.Contains(msg, "sql_query") {
		t.Error("error message should list available tools")
	}
	if !strings.Contains(msg, "Suggested actions") {
		t.Error("error message should contain suggested actions")
	}
}

func TestBuildToolErrorMessage_ConnectionError(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)

	tc := providers.ToolCall{Name: "db_tool", Arguments: nil}
	msg := a.buildToolErrorMessage(tc, fmt.Errorf("connection timeout"))
	if !strings.Contains(msg, "temporary issue") {
		t.Error("connection error should suggest retry")
	}
}

func TestBuildToolErrorMessage_PermissionError(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)

	tc := providers.ToolCall{Name: "exec", Arguments: nil}
	msg := a.buildToolErrorMessage(tc, fmt.Errorf("permission denied"))
	if !strings.Contains(msg, "don't have permission") {
		t.Error("permission error should mention permission")
	}
}

func TestBuildToolErrorMessage_SQLError(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)

	tc := providers.ToolCall{Name: "sql", Arguments: nil}
	msg := a.buildToolErrorMessage(tc, fmt.Errorf("sql syntax error"))
	if !strings.Contains(msg, "SQL syntax") {
		t.Error("SQL error should mention syntax")
	}
}

// --- HandleMessageStreamWithProgress ---

func TestHandleMessageStreamWithProgress(t *testing.T) {
	a, _, mockProvider, _ := setupTestAgent(t)

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "response",
	})

	tokens := make(chan string, 64)
	progress := make(chan bus.OutboundMessage, 64)

	err := a.HandleMessageStreamWithProgress(
		context.Background(), "progress-session", "hello", tokens, progress,
	)
	if err != nil {
		t.Fatalf("HandleMessageStreamWithProgress failed: %v", err)
	}

	// Collect all tokens (non-streaming provider delivers content as single chunk)
	var collected string
	for tok := range tokens {
		collected += tok
	}
	if !strings.Contains(collected, "response") {
		t.Errorf("expected tokens to contain 'response', got %q", collected)
	}
}

// --- helpers ---

type mockRouter struct {
	provider providers.LLMProvider
}

func (r *mockRouter) Chat(ctx context.Context, model string, messages []providers.Message, toolDefs []tools.ToolDefinition, opts *providers.ChatOptions) (*providers.LLMResponse, error) {
	return r.provider.Chat(ctx, messages, toolDefs, model, opts)
}

type coverageMockTool struct {
	name string
}

func (m *coverageMockTool) Name() string        { return m.name }
func (m *coverageMockTool) Description() string { return "mock tool" }
func (m *coverageMockTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{{Name: "arg", Type: "string", Description: "test"}}
}
func (m *coverageMockTool) Execute(_ context.Context, _ map[string]interface{}) (*tools.ToolResult, error) {
	return &tools.ToolResult{ForLLM: "ok"}, nil
}

func newTestPlan() *planner.Plan {
	return planner.NewPlan("test goal")
}

// --- Option functions ---

func TestWithMaxToolIterations(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)
	WithMaxToolIterations(5)(a)
	if a.maxToolIterations != 5 {
		t.Errorf("expected maxToolIterations=5, got %d", a.maxToolIterations)
	}
}

func TestWithCompactor(t *testing.T) {
	a, _, mockProvider, _ := setupTestAgent(t)
	c := NewCompactor(mockProvider, "test-model")
	WithCompactor(c)(a)
	if a.compactor == nil {
		t.Error("expected non-nil compactor")
	}
}

func TestWithRouter(t *testing.T) {
	a, _, mockProvider, _ := setupTestAgent(t)
	router := &mockRouter{provider: mockProvider}
	WithRouter(router)(a)
	if a.router == nil {
		t.Error("expected non-nil router")
	}
}

func TestWithTracker(t *testing.T) {
	a, _, _, _ := setupTestAgent(t)
	// Tracker can be nil — just testing the option sets the field
	WithTracker(nil)(a)
	// No assertion needed, just ensuring no panic
}

// --- Embedded functions ---

func TestChat(t *testing.T) {
	a, _, mockProvider, _ := setupTestAgent(t)
	mockProvider.AddResponse(&providers.LLMResponse{Content: "chat response"})

	resp, err := a.Chat(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp != "chat response" {
		t.Errorf("expected 'chat response', got %q", resp)
	}
}

// --- Compactor internals ---

func TestBuildSummaryPrompt(t *testing.T) {
	c := &Compactor{}
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "What is Go?"},
		{Role: providers.RoleAssistant, Content: "Go is a language."},
		{Role: providers.RoleAssistant, Content: ""},                  // empty assistant
		{Role: providers.RoleTool, Content: strings.Repeat("x", 300)}, // long tool output
	}
	prompt := c.buildSummaryPrompt(messages)
	if !strings.Contains(prompt, "User: What is Go?") {
		t.Error("prompt should contain user message")
	}
	if !strings.Contains(prompt, "Assistant: Go is a language.") {
		t.Error("prompt should contain assistant message")
	}
	if !strings.Contains(prompt, "Tool: ") {
		t.Error("prompt should contain tool result")
	}
	// Tool output should be truncated to 200 chars
	if strings.Count(prompt, "x") > 210 {
		t.Error("tool output should be truncated")
	}
}

func TestSimpleSummary(t *testing.T) {
	c := &Compactor{}
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: "hello"},
		{Role: providers.RoleAssistant, Content: "hi", ToolCalls: []providers.ToolCall{{Name: "sql_query"}}},
		{Role: providers.RoleTool, Content: "result"},
		{Role: providers.RoleUser, Content: strings.Repeat("x", 200)}, // long user msg
	}
	summary := c.simpleSummary(messages)
	if !strings.Contains(summary, "User asked") {
		t.Error("summary should contain user messages")
	}
	if !strings.Contains(summary, "sql_query") {
		t.Error("summary should contain tool names")
	}
	if !strings.Contains(summary, "Tool result") {
		t.Error("summary should contain tool results")
	}
}

// --- Hooks error paths ---

func TestRunAfterMessage_WithError(t *testing.T) {
	h := &Hooks{
		AfterMessage: func(ctx context.Context, sessionID, response string) error {
			return fmt.Errorf("hook failed")
		},
	}
	hc := NewHookChain(func() *Hooks { return h }, func() logger.Logger { return logger.NopLogger() })
	// Should not panic — error is logged
	hc.RunAfterMessage(context.Background(), "sess", "response")
}

func TestRunBeforeLLM_WithError(t *testing.T) {
	h := &Hooks{
		BeforeLLM: func(ctx context.Context, msgs []providers.Message) error {
			return fmt.Errorf("hook failed")
		},
	}
	hc := NewHookChain(func() *Hooks { return h }, func() logger.Logger { return logger.NopLogger() })
	hc.RunBeforeLLM(context.Background(), nil)
}

func TestRunOnError_WithError(t *testing.T) {
	h := &Hooks{
		OnError: func(ctx context.Context, err error) error {
			return fmt.Errorf("hook also failed")
		},
	}
	hc := NewHookChain(func() *Hooks { return h }, func() logger.Logger { return logger.NopLogger() })
	hc.RunOnError(context.Background(), fmt.Errorf("original error"))
}

// Ensure imports are used
var _ = config.DefaultConfig
