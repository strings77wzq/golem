package metrics

import (
	"testing"
)

func TestLookupByToolName_RAG(t *testing.T) {
	m := LookupByToolName("rag_retrieve")
	if m == nil {
		t.Fatal("expected non-nil metrics for rag_retrieve")
	}
	if m.Calls == nil {
		t.Error("expected non-nil Calls counter")
	}
	if m.Latency == nil {
		t.Error("expected non-nil Latency histogram")
	}
	if m.Errors == nil {
		t.Error("expected non-nil Errors counter")
	}
}

func TestLookupByToolName_Memory(t *testing.T) {
	m := LookupByToolName("memory_recall")
	if m == nil {
		t.Fatal("expected non-nil metrics for memory_recall")
	}
}

func TestLookupByToolName_MCP(t *testing.T) {
	m := LookupByToolName("mcp_some_tool")
	if m == nil {
		t.Fatal("expected non-nil metrics for mcp_some_tool")
	}
}

func TestLookupByToolName_Skills(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
	}{
		{"skill prefix", "skill_summarize"},
		{"summarize prefix", "summarize_doc"},
		{"code_review prefix", "code_review_go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := LookupByToolName(tt.toolName)
			if m == nil {
				t.Fatalf("expected non-nil metrics for %s", tt.toolName)
			}
		})
	}
}

func TestLookupByToolName_Unknown(t *testing.T) {
	m := LookupByToolName("sql_query")
	if m != nil {
		t.Errorf("expected nil for unknown tool, got %+v", m)
	}
}

func TestLookupByToolName_Empty(t *testing.T) {
	m := LookupByToolName("")
	if m != nil {
		t.Errorf("expected nil for empty tool name, got %+v", m)
	}
}

func TestLookupByToolName_ShortToolName(t *testing.T) {
	// Tool name shorter than prefix should not match
	m := LookupByToolName("ra")
	if m != nil {
		t.Errorf("expected nil for short tool name, got %+v", m)
	}
}

func TestEnsureInit_Idempotent(t *testing.T) {
	// ensureInit uses sync.Once, calling it multiple times should be safe
	ensureInit()
	ensureInit()
	ensureInit()

	// Verify metrics are still functional after multiple init calls
	m := LookupByToolName("rag_test")
	if m == nil {
		t.Fatal("expected non-nil after multiple ensureInit calls")
	}
}
