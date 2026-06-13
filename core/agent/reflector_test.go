package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/strings77wzq/golem/core/planner"
	"github.com/strings77wzq/golem/core/tools"
)

type mockTool struct {
	name string
	desc string
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return m.desc }
func (m *mockTool) Parameters() []tools.ToolParameter {
	return nil
}
func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	return &tools.ToolResult{ForLLM: "ok"}, nil
}

func setupTestRegistry() *tools.Registry {
	reg := tools.NewRegistry()
	reg.Register(&mockTool{name: "sql_query", desc: "Execute SQL queries"})
	reg.Register(&mockTool{name: "sql_schema", desc: "Get database schema"})
	reg.Register(&mockTool{name: "redis_get", desc: "Get value from Redis"})
	reg.Register(&mockTool{name: "docker_ps", desc: "List Docker containers"})
	reg.Register(&mockTool{name: "file_read", desc: "Read file contents"})
	reg.Register(&mockTool{name: "web_search", desc: "Search the web"})
	return reg
}

func TestToolSelectorBasic(t *testing.T) {
	reg := setupTestRegistry()
	ts := NewToolSelector(reg)

	result := ts.Select("query the database for users", nil, 3)
	if len(result) == 0 {
		t.Fatal("expected at least 1 tool")
	}
	if len(result) > 3 {
		t.Errorf("expected at most 3 tools, got %d", len(result))
	}
}

func TestToolSelectorWithHints(t *testing.T) {
	reg := setupTestRegistry()
	ts := NewToolSelector(reg)

	result := ts.Select("do something", []string{"sql_query"}, 5)
	if len(result) == 0 {
		t.Fatal("expected at least 1 tool")
	}
	// sql_query should be first due to hint
	if result[0].Name != "sql_query" {
		t.Errorf("expected sql_query first, got %q", result[0].Name)
	}
}

func TestToolSelectorMaxTools(t *testing.T) {
	reg := setupTestRegistry()
	ts := NewToolSelector(reg)

	result := ts.Select("use all tools", nil, 2)
	if len(result) > 2 {
		t.Errorf("expected at most 2 tools, got %d", len(result))
	}
}

func TestToolSelectorEmptyRegistry(t *testing.T) {
	reg := tools.NewRegistry()
	ts := NewToolSelector(reg)

	result := ts.Select("anything", nil, 5)
	if len(result) != 0 {
		t.Errorf("expected 0 tools, got %d", len(result))
	}
}

func TestReflectorSuccess(t *testing.T) {
	r := NewReflector()
	step := &planner.Step{
		Description: "query users",
		ExpectedOut: "list of users found",
	}

	result := r.Evaluate(step, "Found 5 users in the database", nil)
	if !result.Success {
		t.Errorf("expected success, got: %v", result)
	}
}

func TestReflectorFailureOnError(t *testing.T) {
	r := NewReflector()
	step := &planner.Step{Description: "query users"}

	result := r.Evaluate(step, "", errors.New("connection refused"))
	if result.Success {
		t.Error("expected failure on error")
	}
	if !result.ShouldRevise {
		t.Error("expected ShouldRevise on error")
	}
}

func TestReflectorFailureOnEmpty(t *testing.T) {
	r := NewReflector()
	step := &planner.Step{Description: "query users"}

	result := r.Evaluate(step, "", nil)
	if result.Success {
		t.Error("expected failure on empty result")
	}
}

func TestReflectorOptimisticDefault(t *testing.T) {
	r := NewReflector()
	step := &planner.Step{
		Description: "do something",
		ExpectedOut: "",
	}

	result := r.Evaluate(step, "some output here", nil)
	if !result.Success {
		t.Error("expected optimistic success")
	}
}

func TestExtractKeywords(t *testing.T) {
	keywords := extractKeywords("Find all users in the database")
	if len(keywords) == 0 {
		t.Error("expected keywords")
	}
	// Should filter stop words
	for _, kw := range keywords {
		if kw == "the" || kw == "in" {
			t.Errorf("stop word not filtered: %q", kw)
		}
	}
}
