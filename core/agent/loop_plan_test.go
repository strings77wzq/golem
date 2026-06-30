package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/strings77wzq/golem/core/bus"
	"github.com/strings77wzq/golem/core/config"
	"github.com/strings77wzq/golem/core/providers"
	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/foundation/logger"
)

// setupPlannedTestAgent is like setupTestAgent but enables planning mode so
// processWithPlan is the active path for complex (long) messages.
func setupPlannedTestAgent(t *testing.T) (*Agent, *providers.MockProvider) {
	t.Helper()

	b := bus.New()
	registry := tools.NewRegistry()
	factory := providers.NewFactory()
	store := session.NewMemoryStore()
	log := logger.NopLogger()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "mock/test-model"

	mockProvider := providers.NewMockProvider("mock")
	factory.Register("mock", mockProvider)

	a := New(b, registry, factory, store, log, cfg, WithPlanner(mockProvider, "mock/test-model"))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go a.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	return a, mockProvider
}

// TestProcessWithPlanProviderErrorSurfacesFailure verifies F3: a provider
// returning an error during step execution must NOT be turned into a success
// answer string of the form "Error: …"/"LLM error: …". The step must be
// marked failed and the final result must represent a genuine failure
// distinct from a normal answer.
func TestProcessWithPlanProviderErrorSurfacesFailure(t *testing.T) {
	a, mockProvider := setupPlannedTestAgent(t)

	// First queued response is consumed by planner.Decompose (the plan JSON).
	// No further responses are queued → executeStep's Chat call receives the
	// mock's "no more responses queued" error, reproducing a real provider
	// failure in the step-execution path.
	mockProvider.AddResponse(&providers.LLMResponse{
		Content: `{"steps":[{"description":"query the users table","tool_hints":["sql_query"],"expected_outcome":"a list of users"}]}`,
	})

	// >30 words to trip isComplexTask into the processWithPlan path.
	longMsg := strings.Repeat("please query the database for every active user record now and return the complete list of matching rows back to me ", 4)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	content, err := a.HandleMessage(ctx, "plan-err-session", longMsg)

	// The failure must NOT be surfaced as a fake success answer matching the
	// historical "Error: %v" / "LLM error: %v" success-string pattern.
	if strings.HasPrefix(content, "Error: ") || strings.HasPrefix(content, "LLM error: ") {
		t.Errorf("provider error surfaced as success-string answer %q — should be a real failure, not \"Error: …\"", content)
	}

	// Either a Go error is returned OR content carries a user-visible failure
	// marker distinct from a normal answer. Both are acceptable; an empty
	// string with nil error is NOT (that would silently swallow the failure).
	if err == nil && !strings.Contains(content, "fail") && !strings.Contains(content, "Fail") && !strings.Contains(content, "error") {
		t.Errorf("failure was swallowed: content=%q err=%v — expected a visible failure marker or a non-nil error", content, err)
	}
}