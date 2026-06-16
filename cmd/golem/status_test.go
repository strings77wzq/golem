package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/tools"
)

func TestStatus_ShowsToolCount(t *testing.T) {
	registry := setupTestRegistry()
	buf := new(bytes.Buffer)

	showStatusTools(buf, registry)

	output := buf.String()
	if !strings.Contains(output, "2 tools registered") {
		t.Errorf("output should show tool count, got: %s", output)
	}
}

func TestStatus_ShowsToolNames(t *testing.T) {
	registry := setupTestRegistry()
	buf := new(bytes.Buffer)

	showStatusTools(buf, registry)

	output := buf.String()
	if !strings.Contains(output, "another_tool") {
		t.Errorf("output should contain 'another_tool', got: %s", output)
	}
	if !strings.Contains(output, "test_tool") {
		t.Errorf("output should contain 'test_tool', got: %s", output)
	}
}

func TestStatus_EmptyRegistry(t *testing.T) {
	registry := tools.NewRegistry()
	buf := new(bytes.Buffer)

	showStatusTools(buf, registry)

	output := buf.String()
	if !strings.Contains(output, "0 tools registered") {
		t.Errorf("output should show 0 tools, got: %s", output)
	}
}

func TestStatus_ShowsFeatures(t *testing.T) {
	features := FeatureStatus{
		MCPEnabled:   true,
		MCPServers:   2,
		RAGEnabled:   false,
		MemoryEnabled: true,
		SkillsEnabled: true,
		SkillsCount:  5,
	}
	buf := new(bytes.Buffer)

	showStatusFeatures(buf, features)

	output := buf.String()
	if !strings.Contains(output, "MCP:        enabled (2 servers)") {
		t.Errorf("MCP status incorrect, got: %s", output)
	}
	if !strings.Contains(output, "RAG:        disabled") {
		t.Errorf("RAG status incorrect, got: %s", output)
	}
	if !strings.Contains(output, "Memory:     enabled") {
		t.Errorf("Memory status incorrect, got: %s", output)
	}
	if !strings.Contains(output, "Skills:     enabled (5)") {
		t.Errorf("Skills status incorrect, got: %s", output)
	}
}

func TestStatus_AllFeaturesDisabled(t *testing.T) {
	features := FeatureStatus{}
	buf := new(bytes.Buffer)

	showStatusFeatures(buf, features)

	output := buf.String()
	if !strings.Contains(output, "MCP:        disabled") {
		t.Errorf("MCP should be disabled, got: %s", output)
	}
	if !strings.Contains(output, "RAG:        disabled") {
		t.Errorf("RAG should be disabled, got: %s", output)
	}
	if !strings.Contains(output, "Memory:     disabled") {
		t.Errorf("Memory should be disabled, got: %s", output)
	}
	if !strings.Contains(output, "Skills:     disabled") {
		t.Errorf("Skills should be disabled, got: %s", output)
	}
}
