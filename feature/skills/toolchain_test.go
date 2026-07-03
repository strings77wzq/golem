package skills

import (
	"context"
	"testing"

	"github.com/strings77wzq/golem/core/tools"
)

type mockTool struct {
	name   string
	output string
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return "mock tool" }
func (m *mockTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{{Name: "input", Type: "string", Description: "input"}}
}
func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	return &tools.ToolResult{ForLLM: m.output, ForUser: m.output}, nil
}

func TestSkillHasSteps(t *testing.T) {
	s := &Skill{Name: "test", Steps: []Step{{Tool: "foo"}}}
	if !s.HasSteps() {
		t.Error("expected HasSteps() true")
	}
	s2 := &Skill{Name: "test2"}
	if s2.HasSteps() {
		t.Error("expected HasSteps() false")
	}
}

func TestSkillHasCondition(t *testing.T) {
	s := &Skill{Name: "test", Condition: "var:file_path"}
	if !s.HasCondition() {
		t.Error("expected HasCondition() true")
	}
	s2 := &Skill{Name: "test2"}
	if s2.HasCondition() {
		t.Error("expected HasCondition() false")
	}
}

func TestToolChainExecute(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&mockTool{name: "reader", output: "file contents here"})
	tc := NewToolChain(registry)

	skill := &Skill{
		Name: "test-skill",
		Steps: []Step{
			{Tool: "reader", Input: map[string]string{"path": "{{file_path}}"}, OutputVar: "content"},
		},
	}

	result, err := tc.Execute(context.Background(), skill, map[string]interface{}{
		"file_path": "/tmp/test.txt",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result != "file contents here" {
		t.Errorf("expected 'file contents here', got %q", result)
	}
}

func TestToolChainNoSteps(t *testing.T) {
	registry := tools.NewRegistry()
	tc := NewToolChain(registry)
	skill := &Skill{Name: "empty"}

	_, err := tc.Execute(context.Background(), skill, nil)
	if err == nil {
		t.Error("expected error for no steps")
	}
}

func TestToolChainMissingTool(t *testing.T) {
	registry := tools.NewRegistry()
	tc := NewToolChain(registry)
	skill := &Skill{
		Name:  "test",
		Steps: []Step{{Tool: "nonexistent"}},
	}

	_, err := tc.Execute(context.Background(), skill, nil)
	if err == nil {
		t.Error("expected error for missing tool")
	}
}

func TestToolChainConditionSkip(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&mockTool{name: "reader", output: "skipped"})
	tc := NewToolChain(registry)

	skill := &Skill{
		Name: "conditional",
		Steps: []Step{
			{Tool: "reader", Condition: "var:file_path"},
		},
	}

	result, err := tc.Execute(context.Background(), skill, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result when condition false, got %q", result)
	}
}

func TestResolveArgs(t *testing.T) {
	vars := map[string]interface{}{
		"name": "golem",
		"port": "8080",
	}

	template := map[string]string{
		"greeting": "hello {{name}}",
		"url":      "http://localhost:{{port}}/api",
		"literal":  "no interpolation",
	}

	args := resolveArgs(template, vars)
	if args["greeting"] != "hello golem" {
		t.Errorf("expected 'hello golem', got %v", args["greeting"])
	}
	if args["url"] != "http://localhost:8080/api" {
		t.Errorf("expected 'http://localhost:8080/api', got %v", args["url"])
	}
	if args["literal"] != "no interpolation" {
		t.Errorf("expected 'no interpolation', got %v", args["literal"])
	}
}

func TestInterpolateStringSelfReferential(t *testing.T) {
	// Self-referential variable should not cause infinite loop
	vars := map[string]interface{}{
		"cmd": "echo {{cmd}}",
	}
	result := interpolateString("run {{cmd}}", vars)
	// Should terminate within maxInterpolationDepth iterations
	// The exact output depends on depth, but it must not hang
	_ = result // just verify it terminates
}

func TestInterpolateStringIndirectCycle(t *testing.T) {
	// Indirect cycle: A references B, B references A
	vars := map[string]interface{}{
		"a": "{{b}}",
		"b": "{{a}}",
	}
	result := interpolateString("start {{a}}", vars)
	_ = result // must terminate
}

func TestInterpolateStringMaxDepth(t *testing.T) {
	// Deeply nested but valid interpolation (no cycle)
	vars := map[string]interface{}{
		"l1": "{{l2}}",
		"l2": "{{l3}}",
		"l3": "final",
	}
	result := interpolateString("{{l1}}", vars)
	if result != "final" {
		t.Errorf("expected 'final', got %q", result)
	}
}

func TestEvaluateCondition(t *testing.T) {
	vars := map[string]interface{}{
		"present": "yes",
		"empty":   "",
		"zero":    "0",
	}

	if !evaluateCondition("var:present", vars) {
		t.Error("expected true for present var")
	}
	if evaluateCondition("var:missing", vars) {
		t.Error("expected false for missing var")
	}
	if evaluateCondition("var:empty", vars) {
		t.Error("expected false for empty var")
	}
	if evaluateCondition("var:zero", vars) {
		t.Error("expected false for zero var")
	}
}
