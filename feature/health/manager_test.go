package health_test

import (
	"context"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/feature/health"
	"github.com/strings77wzq/golem/foundation/logger"
)

type mockHealthChecker struct {
	name    string
	status  string
	latency int64
	err     string
}

func (m *mockHealthChecker) HealthCheck(ctx context.Context) (*providers.HealthStatus, error) {
	return &providers.HealthStatus{
		Provider:  m.name,
		Status:    m.status,
		Latency:   m.latency,
		Error:     m.err,
		CheckedAt: time.Now().Unix(),
	}, nil
}

func TestNew(t *testing.T) {
	log := logger.NopLogger()
	m := health.New(log)
	if m == nil {
		t.Fatal("expected non-nil Manager")
	}
}

func TestNewWithInterval(t *testing.T) {
	log := logger.NopLogger()
	m := health.New(log, health.WithInterval(1*time.Second))
	if m == nil {
		t.Fatal("expected non-nil Manager")
	}
}

func TestRegisterAndGetStatus(t *testing.T) {
	log := logger.NopLogger()
	m := health.New(log)

	mock := &mockHealthChecker{name: "test", status: "healthy", latency: 100}
	m.Register(mock)

	ctx := context.Background()
	m.CheckNow(ctx)

	status, ok := m.GetStatus("test")
	if !ok {
		t.Fatal("expected status for 'test'")
	}
	if status.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", status.Status)
	}
	if status.Latency != 100 {
		t.Errorf("expected latency 100, got %d", status.Latency)
	}
}

func TestGetAllStatuses(t *testing.T) {
	log := logger.NopLogger()
	m := health.New(log)

	m.Register(&mockHealthChecker{name: "openai", status: "healthy", latency: 50})
	m.Register(&mockHealthChecker{name: "anthropic", status: "degraded", latency: 2500})

	ctx := context.Background()
	m.CheckNow(ctx)

	statuses := m.GetAllStatuses()
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}
	if statuses["openai"].Status != "healthy" {
		t.Errorf("expected openai status 'healthy', got %q", statuses["openai"].Status)
	}
	if statuses["anthropic"].Status != "degraded" {
		t.Errorf("expected anthropic status 'degraded', got %q", statuses["anthropic"].Status)
	}
}

func TestGetStatusNotFound(t *testing.T) {
	log := logger.NopLogger()
	m := health.New(log)

	_, ok := m.GetStatus("nonexistent")
	if ok {
		t.Error("expected false for nonexistent provider")
	}
}

func TestStartAndStop(t *testing.T) {
	log := logger.NopLogger()
	m := health.New(log, health.WithInterval(100*time.Millisecond))

	m.Register(&mockHealthChecker{name: "test", status: "healthy", latency: 10})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	m.Stop()

	status, ok := m.GetStatus("test")
	if !ok {
		t.Fatal("expected status for 'test'")
	}
	if status.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", status.Status)
	}
}
