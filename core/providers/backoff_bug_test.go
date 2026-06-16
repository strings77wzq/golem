package providers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/tools"
)

// Bug: RetryProvider waits for backoff delay even after context is cancelled.
func TestRetryProvider_ContextCancelDuringBackoff(t *testing.T) {
	mock := &cancelTestProvider{
		failCount: 10,
		errMsg:    "server error: 500",
	}

	retry := NewRetryProvider(mock, RetryConfig{
		MaxRetries: 5,
		BaseDelay:  5 * time.Second, // long delay to test cancellation
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := retry.Chat(ctx, nil, nil, "test", nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error")
	}

	// Should return quickly after context cancellation
	if elapsed > 500*time.Millisecond {
		t.Errorf("took too long (%v), context cancellation not respected during backoff", elapsed)
	}

	if mock.callCount < 1 {
		t.Errorf("expected at least 1 call, got %d", mock.callCount)
	}
}

// Bug: isClientError incorrectly matches "rate limit" as client error.
func TestRetryProvider_RateLimitShouldRetry(t *testing.T) {
	mock := &cancelTestProvider{
		failCount: 2,
		errMsg:    "rate limit exceeded",
		response:  &LLMResponse{Content: "success"},
	}

	retry := NewRetryProvider(mock, RetryConfig{
		MaxRetries: 3,
		BaseDelay:  10 * time.Millisecond,
	})

	resp, err := retry.Chat(context.Background(), nil, nil, "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "success" {
		t.Errorf("expected 'success', got %q", resp.Content)
	}
	if mock.callCount != 3 {
		t.Errorf("expected 3 calls (rate limit should retry), got %d", mock.callCount)
	}
}

// cancelTestProvider is a test provider for cancellation and rate limit tests.
type cancelTestProvider struct {
	failCount int
	callCount int
	response  *LLMResponse
	errMsg    string
}

func (p *cancelTestProvider) Name() string { return "cancel-test" }

func (p *cancelTestProvider) Chat(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition, model string, opts *ChatOptions) (*LLMResponse, error) {
	p.callCount++
	if p.callCount <= p.failCount {
		return nil, errors.New(p.errMsg)
	}
	return p.response, nil
}
