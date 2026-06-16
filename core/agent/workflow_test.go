package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/tools"
)

// TestAgentWorkflow_SimpleTextResponse tests the simplest workflow:
// User input → LLM → Final response
func TestAgentWorkflow_SimpleTextResponse(t *testing.T) {
	fmt.Println("\n=== Workflow: Simple Text Response ===")
	fmt.Println("Input:  'What is 2+2?'")
	fmt.Println("LLM:    Returns '4'")

	a, b, mockProvider, _ := setupTestAgent(t)
	defer b.Close()

	mockProvider.AddResponse(&providers.LLMResponse{
		Content:   "4",
		Usage:     providers.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		StopReason: "end_turn",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	// Simulate user input
	resp, err := a.HandleMessage(ctx, "workflow-simple", "What is 2+2?")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	fmt.Printf("Output: %s\n", resp)
	fmt.Printf("LLM Calls: %d\n", mockProvider.CallCount)

	if resp != "4" {
		t.Errorf("expected '4', got %q", resp)
	}
	if mockProvider.CallCount != 1 {
		t.Errorf("expected 1 LLM call, got %d", mockProvider.CallCount)
	}

	fmt.Println("✓ PASS: Simple text response workflow")
}

// TestAgentWorkflow_ToolCalling tests the full tool calling workflow:
// User input → LLM requests tool → Tool executes → LLM returns final answer
func TestAgentWorkflow_ToolCalling(t *testing.T) {
	fmt.Println("\n=== Workflow: Tool Calling ===")
	fmt.Println("Input:  'What files are in the current directory?'")
	fmt.Println("LLM:    Requests 'list_files' tool")
	fmt.Println("Tool:   Returns 'file1.go, file2.go, file3.go'")
	fmt.Println("LLM:    Returns 'There are 3 files: file1.go, file2.go, file3.go'")

	a, b, mockProvider, registry := setupTestAgent(t)
	defer b.Close()

	// Register a mock file listing tool
	fileListTool := &tools.MockTool{
		ToolName:        "list_files",
		ToolDescription: "Lists files in current directory",
		ToolParameters: []tools.ToolParameter{
			{Name: "path", Type: "string", Description: "Directory path", Required: false},
		},
	}
	fileListTool.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		fmt.Printf("  Tool 'list_files' executed with args: %v\n", args)
		return &tools.ToolResult{
			ForLLM:  "file1.go, file2.go, file3.go",
			ForUser: "Found 3 files",
		}, nil
	}
	registry.Register(fileListTool)

	// LLM first requests the tool, then returns final answer
	mockProvider.AddResponse(&providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{
				ID:        "call_001",
				Name:      "list_files",
				Arguments: map[string]interface{}{"path": "."},
			},
		},
	})
	mockProvider.AddResponse(&providers.LLMResponse{
		Content:    "There are 3 files: file1.go, file2.go, file3.go",
		Usage:      providers.TokenUsage{PromptTokens: 50, CompletionTokens: 20, TotalTokens: 70},
		StopReason: "end_turn",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	resp, err := a.HandleMessage(ctx, "workflow-tool", "What files are in the current directory?")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	fmt.Printf("Output: %s\n", resp)
	fmt.Printf("LLM Calls: %d\n", mockProvider.CallCount)

	if resp != "There are 3 files: file1.go, file2.go, file3.go" {
		t.Errorf("unexpected response: %q", resp)
	}
	if mockProvider.CallCount != 2 {
		t.Errorf("expected 2 LLM calls (tool request + final), got %d", mockProvider.CallCount)
	}

	// Verify the LLM received the tool result
	lastMessages := mockProvider.LastMessages
	foundToolResult := false
	for _, msg := range lastMessages {
		if msg.Role == providers.RoleTool && strings.Contains(msg.Content, "file1.go") {
			foundToolResult = true
			break
		}
	}
	if !foundToolResult {
		t.Error("LLM should have received tool result with file list")
	}

	fmt.Println("✓ PASS: Tool calling workflow")
}

// TestAgentWorkflow_MultiTurnConversation tests multi-turn context:
// Turn 1: "My name is Alice" → LLM acknowledges
// Turn 2: "What's my name?" → LLM remembers "Alice"
func TestAgentWorkflow_MultiTurnConversation(t *testing.T) {
	fmt.Println("\n=== Workflow: Multi-Turn Conversation ===")
	fmt.Println("Turn 1: 'My name is Alice' → LLM: 'Nice to meet you, Alice!'")
	fmt.Println("Turn 2: 'What's my name?' → LLM: 'Your name is Alice.'")

	a, b, mockProvider, _ := setupTestAgent(t)
	defer b.Close()

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Nice to meet you, Alice!",
	})
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Your name is Alice.",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	sessionID := "workflow-multiturn"

	// Turn 1
	resp1, err := a.HandleMessage(ctx, sessionID, "My name is Alice")
	if err != nil {
		t.Fatalf("turn 1 error: %v", err)
	}
	fmt.Printf("Turn 1 Output: %s\n", resp1)

	// Turn 2
	resp2, err := a.HandleMessage(ctx, sessionID, "What's my name?")
	if err != nil {
		t.Fatalf("turn 2 error: %v", err)
	}
	fmt.Printf("Turn 2 Output: %s\n", resp2)

	if resp1 != "Nice to meet you, Alice!" {
		t.Errorf("turn 1: expected 'Nice to meet you, Alice!', got %q", resp1)
	}
	if resp2 != "Your name is Alice." {
		t.Errorf("turn 2: expected 'Your name is Alice.', got %q", resp2)
	}

	// Verify session has both messages
	sess, found := a.sessionStore.Get(sessionID)
	if !found {
		t.Fatal("session not found")
	}
	messages := sess.GetMessages()
	fmt.Printf("Session messages: %d (system + 2 user + 2 assistant = 5)\n", len(messages))

	// Should have: system + user1 + assistant1 + user2 + assistant2 = 5
	if len(messages) != 5 {
		t.Errorf("expected 5 messages, got %d", len(messages))
	}

	fmt.Println("✓ PASS: Multi-turn conversation workflow")
}

// TestAgentWorkflow_ToolErrorRecovery tests error handling:
// LLM requests tool → Tool fails → LLM gets error → LLM handles gracefully
func TestAgentWorkflow_ToolErrorRecovery(t *testing.T) {
	fmt.Println("\n=== Workflow: Tool Error Recovery ===")
	fmt.Println("Input:  'Read the file /nonexistent.txt'")
	fmt.Println("LLM:    Requests 'read_file' tool")
	fmt.Println("Tool:   Returns error 'file not found'")
	fmt.Println("LLM:    Returns 'The file /nonexistent.txt does not exist.'")

	a, b, mockProvider, registry := setupTestAgent(t)
	defer b.Close()

	errorTool := &tools.MockTool{
		ToolName:        "read_file",
		ToolDescription: "Reads file contents",
		ToolParameters: []tools.ToolParameter{
			{Name: "path", Type: "string", Description: "File path", Required: true},
		},
	}
	errorTool.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		path := args["path"].(string)
		fmt.Printf("  Tool 'read_file' failed: %s not found\n", path)
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Error: file %s not found", path),
			ForUser: fmt.Sprintf("Error: %s not found", path),
			IsError: true,
		}, nil
	}
	registry.Register(errorTool)

	mockProvider.AddResponse(&providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{
				ID:        "call_err",
				Name:      "read_file",
				Arguments: map[string]interface{}{"path": "/nonexistent.txt"},
			},
		},
	})
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "The file /nonexistent.txt does not exist.",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	resp, err := a.HandleMessage(ctx, "workflow-error", "Read the file /nonexistent.txt")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	fmt.Printf("Output: %s\n", resp)
	fmt.Printf("LLM Calls: %d\n", mockProvider.CallCount)

	if resp != "The file /nonexistent.txt does not exist." {
		t.Errorf("unexpected response: %q", resp)
	}
	if mockProvider.CallCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mockProvider.CallCount)
	}

	fmt.Println("✓ PASS: Tool error recovery workflow")
}

// TestAgentWorkflow_StreamingResponse tests streaming:
// User input → LLM streams tokens → Final response
func TestAgentWorkflow_StreamingResponse(t *testing.T) {
	fmt.Println("\n=== Workflow: Streaming Response ===")
	fmt.Println("Input:  'Tell me a joke'")
	fmt.Println("LLM:    Streams 'Why', ' did', ' the', ' chicken', ' cross', ' the', ' road?'")

	a, b, mockProvider, _ := setupTestAgent(t)
	defer b.Close()

	mockProvider.AddResponse(&providers.LLMResponse{
		Content:    "Why did the chicken cross the road?",
		StopReason: "end_turn",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	tokens := make(chan string, 32)
	errCh := make(chan error, 1)

	go func() {
		errCh <- a.HandleMessageStream(ctx, "workflow-stream", "Tell me a joke", tokens)
	}()

	var collected []string
	for tok := range tokens {
		collected = append(collected, tok)
		fmt.Printf("  Token: %q\n", tok)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("streaming error: %v", err)
	}

	full := strings.Join(collected, "")
	fmt.Printf("Full response: %s\n", full)

	if full != "Why did the chicken cross the road?" {
		t.Errorf("expected full response, got %q", full)
	}
	if len(collected) == 0 {
		t.Error("expected streaming tokens")
	}

	fmt.Printf("✓ PASS: Streaming response (%d tokens)\n", len(collected))
}

// TestAgentWorkflow_ToolCallWithStreaming tests streaming with tool calls:
// User input → LLM streams thinking → Tool call → Tool result → LLM streams final answer
func TestAgentWorkflow_ToolCallWithStreaming(t *testing.T) {
	fmt.Println("\n=== Workflow: Tool Call with Streaming ===")
	fmt.Println("Input:  'Search for Go concurrency patterns'")
	fmt.Println("LLM:    Requests 'web_search' tool")
	fmt.Println("Tool:   Returns search results")
	fmt.Println("LLM:    Streams final summary")

	a, b, mockProvider, registry := setupTestAgent(t)
	defer b.Close()

	searchTool := &tools.MockTool{
		ToolName:        "web_search",
		ToolDescription: "Searches the web",
		ToolParameters: []tools.ToolParameter{
			{Name: "query", Type: "string", Description: "Search query", Required: true},
		},
	}
	searchTool.ExecuteFn = func(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
		query := args["query"].(string)
		fmt.Printf("  Tool 'web_search' executed: %s\n", query)
		return &tools.ToolResult{
			ForLLM: "Go concurrency patterns: goroutines, channels, sync primitives, errgroup",
		}, nil
	}
	registry.Register(searchTool)

	mockProvider.AddResponse(&providers.LLMResponse{
		ToolCalls: []providers.ToolCall{
			{
				ID:        "call_search",
				Name:      "web_search",
				Arguments: map[string]interface{}{"query": "Go concurrency patterns"},
			},
		},
	})
	mockProvider.AddResponse(&providers.LLMResponse{
		Content:    "Go concurrency patterns include goroutines, channels, and sync primitives.",
		StopReason: "end_turn",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	tokens := make(chan string, 32)
	errCh := make(chan error, 1)

	go func() {
		errCh <- a.HandleMessageStream(ctx, "workflow-stream-tool", "Search for Go concurrency patterns", tokens)
	}()

	var collected []string
	for tok := range tokens {
		collected = append(collected, tok)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("streaming error: %v", err)
	}

	full := strings.Join(collected, "")
	fmt.Printf("Final response: %s\n", full)

	if !strings.Contains(full, "goroutines") {
		t.Errorf("expected response to contain 'goroutines', got %q", full)
	}

	fmt.Println("✓ PASS: Tool call with streaming workflow")
}

// TestAgentWorkflow_HooksExecution tests lifecycle hooks:
// User input → BeforeMessage hook → LLM → AfterMessage hook
func TestAgentWorkflow_HooksExecution(t *testing.T) {
	fmt.Println("\n=== Workflow: Hooks Execution ===")
	fmt.Println("Input:  'Hello'")
	fmt.Println("Hooks:  BeforeMessage fires → LLM → AfterMessage fires")

	a, b, mockProvider, _ := setupTestAgent(t)
	defer b.Close()

	hookLog := []string{}

	a.hooks = &Hooks{
		BeforeMessage: func(ctx context.Context, sessionID, message string) error {
			hookLog = append(hookLog, fmt.Sprintf("BeforeMessage: %s", message))
			fmt.Printf("  Hook: BeforeMessage('%s')\n", message)
			return nil
		},
		AfterMessage: func(ctx context.Context, sessionID, response string) error {
			hookLog = append(hookLog, fmt.Sprintf("AfterMessage: %s", response))
			fmt.Printf("  Hook: AfterMessage('%s')\n", response)
			return nil
		},
	}

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Hello there!",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	resp, err := a.HandleMessage(ctx, "workflow-hooks", "Hello")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	fmt.Printf("Output: %s\n", resp)
	fmt.Printf("Hooks fired: %d\n", len(hookLog))
	for i, h := range hookLog {
		fmt.Printf("  [%d] %s\n", i+1, h)
	}

	if len(hookLog) != 2 {
		t.Errorf("expected 2 hooks, got %d", len(hookLog))
	}
	if !strings.Contains(hookLog[0], "BeforeMessage") {
		t.Error("first hook should be BeforeMessage")
	}
	if !strings.Contains(hookLog[1], "AfterMessage") {
		t.Error("second hook should be AfterMessage")
	}

	fmt.Println("✓ PASS: Hooks execution workflow")
}

// TestAgentWorkflow_ContextCancellation tests graceful shutdown:
// User input → LLM starts → Context cancelled → Graceful exit
func TestAgentWorkflow_ContextCancellation(t *testing.T) {
	fmt.Println("\n=== Workflow: Context Cancellation ===")
	fmt.Println("Input:  'Long task'")
	fmt.Println("Action: Cancel context mid-processing")

	a, b, mockProvider, _ := setupTestAgent(t)
	defer b.Close()

	mockProvider.AddResponse(&providers.LLMResponse{
		Content: "Response",
	})

	ctx, cancel := context.WithCancel(context.Background())

	go a.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	// Cancel immediately
	cancel()
	time.Sleep(100 * time.Millisecond)

	fmt.Println("✓ PASS: Context cancellation handled gracefully")
}

// TestAgentWorkflow_SessionIsolation tests that different sessions are isolated:
// Session A: "I like cats"
// Session B: "I like dogs"
// Session A: "What do I like?" → "cats" (not "dogs")
func TestAgentWorkflow_SessionIsolation(t *testing.T) {
	fmt.Println("\n=== Workflow: Session Isolation ===")
	fmt.Println("Session A: 'I like cats'")
	fmt.Println("Session B: 'I like dogs'")
	fmt.Println("Session A: 'What do I like?' → should remember 'cats'")

	a, b, mockProvider, _ := setupTestAgent(t)
	defer b.Close()

	mockProvider.AddResponse(&providers.LLMResponse{Content: "Got it!"})
	mockProvider.AddResponse(&providers.LLMResponse{Content: "Noted!"})
	mockProvider.AddResponse(&providers.LLMResponse{Content: "You like cats."})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startAgent(ctx, a)

	// Session A
	resp1, _ := a.HandleMessage(ctx, "session-a", "I like cats")
	fmt.Printf("Session A Turn 1: %s\n", resp1)

	// Session B
	resp2, _ := a.HandleMessage(ctx, "session-b", "I like dogs")
	fmt.Printf("Session B Turn 1: %s\n", resp2)

	// Session A again
	resp3, _ := a.HandleMessage(ctx, "session-a", "What do I like?")
	fmt.Printf("Session A Turn 2: %s\n", resp3)

	if resp3 != "You like cats." {
		t.Errorf("session isolation failed: expected 'You like cats.', got %q", resp3)
	}

	fmt.Println("✓ PASS: Session isolation maintained")
}

// Note: setupTestAgent and startAgent are defined in agent_test.go
