package providers

import (
	"context"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/strings77wzq/golem/core/tools"
)

// RetryConfig configures the retry behavior.
type RetryConfig struct {
	MaxRetries int           // Maximum number of retries (default: 3)
	BaseDelay  time.Duration // Base delay between retries (default: 1s)
	MaxDelay   time.Duration // Maximum delay cap (default: 30s)
}

func (c RetryConfig) withDefaults() RetryConfig {
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.BaseDelay == 0 {
		c.BaseDelay = 1 * time.Second
	}
	if c.MaxDelay == 0 {
		c.MaxDelay = 30 * time.Second
	}
	return c
}

// RetryProvider wraps an LLMProvider with exponential backoff retry logic.
type RetryProvider struct {
	inner  LLMProvider
	config RetryConfig
}

// NewRetryProvider creates a new retry wrapper around the given provider.
func NewRetryProvider(inner LLMProvider, config RetryConfig) *RetryProvider {
	return &RetryProvider{
		inner:  inner,
		config: config.withDefaults(),
	}
}

// Name returns the inner provider's name.
func (r *RetryProvider) Name() string {
	return r.inner.Name()
}

// Chat calls the inner provider with retry logic.
func (r *RetryProvider) Chat(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition, model string, opts *ChatOptions) (*LLMResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		resp, err := r.inner.Chat(ctx, messages, toolDefs, model, opts)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on context cancellation or client errors (4xx)
		if ctx.Err() != nil {
			return nil, err
		}
		if isClientError(err) {
			return nil, err
		}

		// Don't retry on last attempt
		if attempt == r.config.MaxRetries {
			break
		}

		// Wait with exponential backoff + jitter
		delay := r.calculateDelay(attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	return nil, lastErr
}

// calculateDelay computes exponential backoff with jitter.
func (r *RetryProvider) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(r.config.BaseDelay) * math.Pow(2, float64(attempt)))
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}
	// Add jitter: ±25%
	jitter := float64(delay) * 0.25
	delay = time.Duration(float64(delay) + (rand.Float64()*2-1)*jitter)
	return delay
}

// isClientError checks if the error is a client error (4xx) that should not be retried.
func isClientError(err error) bool {
	msg := strings.ToLower(err.Error())
	// These patterns indicate client errors that won't succeed on retry
	clientPatterns := []string{
		"invalid request",
		"bad request",
		"unauthorized",
		"forbidden",
		"not found",
		"invalid api key",
		"invalid model",
		"context length exceeded",
		"rate limit",
	}
	for _, pattern := range clientPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}
