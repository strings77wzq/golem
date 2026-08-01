package providers

import (
	"context"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"

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

// calculateDelay computes exponential backoff with jitter using
// cenkalti/backoff's ExponentialBackOff scheduler. The retry loop owns the
// attempt budget (MaxElapsedTime is disabled), and a fresh scheduler is
// constructed per call so concurrent Chat invocations stay race-free.
func (r *RetryProvider) calculateDelay(attempt int) time.Duration {
	b := backoff.NewExponentialBackOff()
	// Cap the initial interval so a misconfigured BaseDelay > MaxDelay
	// still respects the cap from the first attempt (matches the previous
	// formula, which clamped before jitter).
	b.InitialInterval = min(r.config.BaseDelay, r.config.MaxDelay)
	b.MaxInterval = r.config.MaxDelay
	b.Multiplier = 2.0
	b.RandomizationFactor = 0.25 // ±25% jitter, mirrors the previous formula
	b.MaxElapsedTime = 0         // attempt budget is managed by the retry loop

	var delay time.Duration
	for i := 0; i <= attempt; i++ {
		delay = b.NextBackOff()
	}
	return delay
}

// isClientError checks if the error is a client error (4xx) that should not be retried.
func isClientError(err error) bool {
	msg := strings.ToLower(err.Error())
	// These patterns indicate client errors that won't succeed on retry
	// Note: "rate limit" is NOT included — 429 errors should be retried with backoff
	clientPatterns := []string{
		"invalid request",
		"bad request",
		"unauthorized",
		"forbidden",
		"not found",
		"invalid api key",
		"invalid model",
		"context length exceeded",
	}
	for _, pattern := range clientPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}
