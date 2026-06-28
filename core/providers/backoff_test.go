package providers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/tools"
)

func TestRetryProvider_SucceedsFirstTry(t *testing.T) {
	mock := NewMockProvider("test")
	mock.AddResponse(&LLMResponse{Content: "hello"})
	retry := NewRetryProvider(mock, RetryConfig{MaxRetries: 3})

	resp, err := retry.Chat(context.Background(), nil, nil, "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "hello" {
		t.Errorf("expected 'hello', got %q", resp.Content)
	}
}

func TestRetryProvider_RetriesOnRetryableError(t *testing.T) {
	mock := &failingProvider{
		failCount: 2,
		response:  &LLMResponse{Content: "success after retry"},
	}
	retry := NewRetryProvider(mock, RetryConfig{
		MaxRetries: 3,
		BaseDelay:  10 * time.Millisecond,
	})

	resp, err := retry.Chat(context.Background(), nil, nil, "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "success after retry" {
		t.Errorf("expected 'success after retry', got %q", resp.Content)
	}
	if mock.callCount != 3 {
		t.Errorf("expected 3 calls, got %d", mock.callCount)
	}
}

func TestRetryProvider_NoRetryOnClientError(t *testing.T) {
	mock := &failingProvider{
		failCount: 10,
		errMsg:    "invalid request: bad parameters",
	}
	retry := NewRetryProvider(mock, RetryConfig{MaxRetries: 3})

	_, err := retry.Chat(context.Background(), nil, nil, "test", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if mock.callCount != 1 {
		t.Errorf("expected 1 call (no retry on client error), got %d", mock.callCount)
	}
}

func TestRetryProvider_ExhaustsRetries(t *testing.T) {
	mock := &failingProvider{
		failCount: 10,
		errMsg:    "server error: 500",
	}
	retry := NewRetryProvider(mock, RetryConfig{
		MaxRetries: 2,
		BaseDelay:  10 * time.Millisecond,
	})

	_, err := retry.Chat(context.Background(), nil, nil, "test", nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	// 1 initial + 2 retries = 3 calls
	if mock.callCount != 3 {
		t.Errorf("expected 3 calls, got %d", mock.callCount)
	}
}

func TestRetryProvider_RespectsContextCancellation(t *testing.T) {
	mock := &failingProvider{
		failCount: 10,
		errMsg:    "timeout",
	}
	retry := NewRetryProvider(mock, RetryConfig{
		MaxRetries: 5,
		BaseDelay:  1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := retry.Chat(ctx, nil, nil, "test", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRetryProvider_Name(t *testing.T) {
	mock := NewMockProvider("openai")
	retry := NewRetryProvider(mock, RetryConfig{})

	if name := retry.Name(); name != "openai" {
		t.Errorf("expected 'openai', got %q", name)
	}
}

// failingProvider returns errors for the first N calls, then succeeds.
type failingProvider struct {
	failCount int
	callCount int
	response  *LLMResponse
	errMsg    string
}

func (f *failingProvider) Name() string { return "failing" }

func (f *failingProvider) Chat(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition, model string, opts *ChatOptions) (*LLMResponse, error) {
	f.callCount++
	if f.callCount <= f.failCount {
		msg := f.errMsg
		if msg == "" {
			msg = "connection timeout"
		}
		return nil, errors.New(msg)
	}
	return f.response, nil
}
