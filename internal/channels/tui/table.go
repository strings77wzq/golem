package tui

import (
	"strings"
)

const maxColWidth = 30
const maxTableRows = 20

// isMarkdownTable checks if a string is a Markdown table.
func isMarkdownTable(s string) bool {
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return false
	}
	// First line should start with |
	if !strings.HasPrefix(strings.TrimSpace(lines[0]), "|") {
		return false
	}
	// Second line should be separator (|---|---|)
	if !strings.Contains(lines[1], "---") {
		return false
	}
	return true
}

// parseMarkdownTable parses a Markdown table into rows.
func parseMarkdownTable(s string) [][]string {
	if !isMarkdownTable(s) {
		return nil
	}

	lines := strings.Split(s, "\n")
	var rows [][]string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "---") {
			continue
		}
		if !strings.HasPrefix(line, "|") {
			continue
		}

		// Split by | and trim spaces
		parts := strings.Split(line, "|")
		var row []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				row = append(row, trimmed)
			}
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}

	return rows
}

// computeWidths calculates the width for each column.
func computeWidths(rows [][]string) []int {
	if len(rows) == 0 {
		return nil
	}

	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			w := len(cell)
			if w > maxColWidth {
				w = maxColWidth
			}
			if w > widths[i] {
				widths[i] = w
			}
		}
	}
	return widths
}

// renderTable renders a Markdown table as a Unicode box-drawing table.
func renderTable(raw string) string {
	rows := parseMarkdownTable(raw)
	if len(rows) == 0 {
		return raw
	}

	// Truncate if too many rows
	totalRows := len(rows)
	showRows := rows
	truncated := false
	if totalRows > maxTableRows {
		showRows = rows[:maxTableRows]
		truncated = true
	}

	widths := computeWidths(rows)
	var sb strings.Builder

	// Top border
	sb.WriteString("┌")
	for i, w := range widths {
		for j := 0; j < w+2; j++ {
			sb.WriteString("─")
		}
		if i < len(widths)-1 {
			sb.WriteString("┬")
		}
	}
	sb.WriteString("┐\n")

	// Header row (first row)
	if len(showRows) > 0 {
		sb.WriteString("│")
		for i, cell := range showRows[0] {
			padded := padRight(cell, widths[i])
			sb.WriteString(" " + padded + " │")
		}
		sb.WriteString("\n")

		// Separator
		sb.WriteString("├")
		for i, w := range widths {
			for j := 0; j < w+2; j++ {
				sb.WriteString("─")
			}
			if i < len(widths)-1 {
				sb.WriteString("┼")
			}
		}
		sb.WriteString("┤\n")
	}

	// Data rows
	for _, row := range showRows[1:] {
		sb.WriteString("│")
		for i := 0; i < len(widths); i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			padded := padRight(cell, widths[i])
			sb.WriteString(" " + padded + " │")
		}
		sb.WriteString("\n")
	}

	// Bottom border
	sb.WriteString("└")
	for i, w := range widths {
		for j := 0; j < w+2; j++ {
			sb.WriteString("─")
		}
		if i < len(widths)-1 {
			sb.WriteString("┴")
		}
	}
	sb.WriteString("┘\n")

	// Row count
	sb.WriteString("(")
	if truncated {
		sb.WriteString("showing ")
	}
	sb.WriteString(strings.Replace(string(rune('0'+totalRows)), string(rune('0'+totalRows)), "", -1))
	sb.WriteString(")")

	return sb.String()
}

// padRight pads a string to the right with spaces, truncating if too long.
func padRight(s string, width int) string {
	if len(s) > width {
		return s[:width-3] + "..."
	}
	return s + strings.Repeat(" ", width-len(s))
}
