package tui

import (
	"strings"
	"testing"
)

func TestParseMarkdownTable(t *testing.T) {
	input := "| id | name | email |\n|----|------|-------|\n| 1 | Alice | alice@example.com |\n| 2 | Bob | bob@example.com |"
	rows := parseMarkdownTable(input)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (header + 2 data), got %d", len(rows))
	}
	// rows[0] is header
	if rows[0][0] != "id" {
		t.Errorf("expected 'id', got %q", rows[0][0])
	}
	// rows[1] is first data row
	if rows[1][1] != "Alice" {
		t.Errorf("expected 'Alice', got %q", rows[1][1])
	}
}

func TestParseMarkdownTableEmpty(t *testing.T) {
	rows := parseMarkdownTable("not a table")
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

func TestComputeWidths(t *testing.T) {
	rows := [][]string{
		{"id", "name", "email"},
		{"1", "Alice", "alice@example.com"},
	}
	widths := computeWidths(rows)
	if widths[0] != 2 { // "id" = 2 chars
		t.Errorf("width[0] = %d, want 2", widths[0])
	}
	if widths[2] != 17 { // "alice@example.com" = 17 chars
		t.Errorf("width[2] = %d, want 17", widths[2])
	}
}

func TestRenderTable(t *testing.T) {
	input := "| id | name |\n|----|------|\n| 1 | Alice |\n| 2 | Bob |"
	output := renderTable(input)
	if !strings.Contains(output, "┌") {
		t.Error("expected Unicode table border")
	}
	if !strings.Contains(output, "Alice") {
		t.Error("expected Alice in output")
	}
	if !strings.Contains(output, "Bob") {
		t.Error("expected Bob in output")
	}
}

func TestRenderTableEmpty(t *testing.T) {
	output := renderTable("not a table")
	if output != "not a table" {
		t.Errorf("expected original text for non-table, got %q", output)
	}
}

func TestRenderTableTruncation(t *testing.T) {
	longName := strings.Repeat("x", 50)
	input := "| id | name |\n|----|------|\n| 1 | " + longName + " |"
	output := renderTable(input)
	if !strings.Contains(output, "...") {
		t.Error("expected truncation indicator")
	}
}

func TestIsMarkdownTable(t *testing.T) {
	if !isMarkdownTable("| a | b |\n|---|---|\n| 1 | 2 |") {
		t.Error("expected true for valid table")
	}
	if isMarkdownTable("just text") {
		t.Error("expected false for non-table")
	}
	if isMarkdownTable("") {
		t.Error("expected false for empty string")
	}
}

func TestRenderTableRowCount(t *testing.T) {
	input := "| id | name |\n|----|------|\n| 1 | Alice |\n| 2 | Bob |"
	output := renderTable(input)
	// Should show "(3 rows)" — header + 2 data rows
	if !strings.Contains(output, "(3 rows)") {
		t.Errorf("expected '(3 rows)' in output, got:\n%s", output)
	}
}

func TestRenderTableRowCountTruncated(t *testing.T) {
	// Create table with >20 rows to trigger truncation
	var sb strings.Builder
	sb.WriteString("| id | name |\n|----|------|\n")
	for i := 0; i < 25; i++ {
		sb.WriteString("| 1 | data |\n")
	}
	output := renderTable(sb.String())
	// Should show "(showing 20 of 26 rows)" — 1 header + 25 data = 26 total, show 20
	if !strings.Contains(output, "showing 20 of 26 rows") {
		t.Errorf("expected '(showing 20 of 26 rows)' in output, got:\n%s", output)
	}
}
