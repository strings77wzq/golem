// Package health provides a health check scheduler for LLM providers.
// It periodically checks all registered providers and caches their health status.
package health

import (
	"context"
	"sync"
	"time"

	coreproviders "github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/foundation/logger"
)

// Manager manages health checks for all providers.
type Manager struct {
	mu        sync.RWMutex
	providers []coreproviders.HealthChecker
	statuses  map[string]*coreproviders.HealthStatus
	interval  time.Duration
	log       logger.Logger
	stopCh    chan struct{}
	stopOnce  sync.Once
}

// Option configures the health Manager.
type Option func(*Manager)

// WithInterval sets the health check interval.
func WithInterval(d time.Duration) Option {
	return func(m *Manager) {
		m.interval = d
	}
}

// New creates a new health check Manager.
func New(log logger.Logger, opts ...Option) *Manager {
	m := &Manager{
		statuses: make(map[string]*coreproviders.HealthStatus),
		interval: 5 * time.Minute,
		log:      log,
		stopCh:   make(chan struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Register adds a provider for health checking.
func (m *Manager) Register(p coreproviders.HealthChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers = append(m.providers, p)
}

// Start begins the health check loop.
func (m *Manager) Start(ctx context.Context) {
	go m.loop(ctx)
}

// Stop stops the health check loop. Safe to call multiple times.
func (m *Manager) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
	})
}

// GetStatus returns the cached health status for a provider.
func (m *Manager) GetStatus(provider string) (*coreproviders.HealthStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status, ok := m.statuses[provider]
	return status, ok
}

// GetAllStatuses returns all cached health statuses.
func (m *Manager) GetAllStatuses() map[string]*coreproviders.HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]*coreproviders.HealthStatus, len(m.statuses))
	for k, v := range m.statuses {
		result[k] = v
	}
	return result
}

// CheckNow performs an immediate health check on all providers.
func (m *Manager) CheckNow(ctx context.Context) map[string]*coreproviders.HealthStatus {
	m.mu.RLock()
	providers := make([]coreproviders.HealthChecker, len(m.providers))
	copy(providers, m.providers)
	m.mu.RUnlock()

	results := make(map[string]*coreproviders.HealthStatus, len(providers))
	for _, p := range providers {
		status, err := p.HealthCheck(ctx)
		if err != nil {
			m.log.Error("health check failed", "error", err)
			continue
		}
		results[status.Provider] = status
	}

	m.mu.Lock()
	for k, v := range results {
		m.statuses[k] = v
	}
	m.mu.Unlock()

	return results
}

func (m *Manager) loop(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	m.CheckNow(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.CheckNow(ctx)
		}
	}
}
