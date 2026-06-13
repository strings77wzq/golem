package database

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/tools"
)

const (
	defaultMaxRows    = 50
	defaultMaxColWidth = 30
	summaryThreshold   = 100
)

// SQLQueryTool executes SQL queries on a database.
type SQLQueryTool struct {
	registry *database.Registry
	maxRows  int
}

// NewSQLQueryTool creates a new SQL query tool.
func NewSQLQueryTool(registry *database.Registry) *SQLQueryTool {
	return &SQLQueryTool{registry: registry, maxRows: defaultMaxRows}
}

// NewSQLQueryToolWithMaxRows creates a SQL query tool with custom max rows.
func NewSQLQueryToolWithMaxRows(registry *database.Registry, maxRows int) *SQLQueryTool {
	return &SQLQueryTool{registry: registry, maxRows: maxRows}
}

func (t *SQLQueryTool) Name() string        { return "sql_query" }
func (t *SQLQueryTool) Description() string { return "Execute SQL SELECT query and return results (auto-truncated to 50 rows)" }

func (t *SQLQueryTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "database", Type: "string", Description: "Database name (defaults to default database)", Required: false},
		{Name: "sql", Type: "string", Description: "SQL query to execute", Required: true},
		{Name: "args", Type: "array", Description: "Query arguments for parameterized query", Required: false},
	}
}

func (t *SQLQueryTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	dbName := t.getDefaultDB(args)
	sqlQuery, _ := args["sql"].(string)
	if sqlQuery == "" {
		return &tools.ToolResult{ForLLM: "Error: sql parameter is required", IsError: true}, nil
	}

	// Safety: only allow SELECT
	trimmed := strings.TrimSpace(strings.ToUpper(sqlQuery))
	if !strings.HasPrefix(trimmed, "SELECT") {
		return &tools.ToolResult{
			ForLLM:  "Error: only SELECT queries are allowed by default. Use --allow-writes for INSERT/UPDATE/DELETE.",
			ForUser: "Write operations require --allow-writes flag",
			IsError: true,
		}, nil
	}

	driver, err := t.registry.GetSQL(dbName)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	queryArgs := t.parseArgs(args)
	rows, err := driver.Query(ctx, sqlQuery, queryArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Query error: %v", err), IsError: true}, nil
	}

	if len(rows) == 0 {
		return &tools.ToolResult{ForLLM: "Query returned 0 rows.", ForUser: "No results found."}, nil
	}

	totalRows := len(rows)

	// Smart aggregation for large result sets
	if totalRows > summaryThreshold {
		summary := t.formatSummary(rows, totalRows, sqlQuery)
		return &tools.ToolResult{
			ForLLM:  summary,
			ForUser: fmt.Sprintf("Found %d rows (showing summary)", totalRows),
		}, nil
	}

	// Truncate if over maxRows
	truncated := false
	if totalRows > t.maxRows {
		rows = rows[:t.maxRows]
		truncated = true
	}

	result := t.formatRows(rows, truncated, totalRows)
	return &tools.ToolResult{
		ForLLM:  result,
		ForUser: fmt.Sprintf("Found %d rows%s", totalRows, t.truncationNote(truncated, totalRows)),
	}, nil
}

func (t *SQLQueryTool) getDefaultDB(args map[string]interface{}) string {
	if db, ok := args["database"].(string); ok && db != "" {
		return db
	}
	return t.registry.Default()
}

func (t *SQLQueryTool) parseArgs(args map[string]interface{}) []interface{} {
	argList, ok := args["args"].([]interface{})
	if !ok {
		return nil
	}
	return argList
}

func (t *SQLQueryTool) truncationNote(truncated bool, total int) string {
	if truncated {
		return fmt.Sprintf(" (showing first %d of %d rows, use WHERE to narrow results)", t.maxRows, total)
	}
	return ""
}

func (t *SQLQueryTool) formatRows(rows []database.Row, truncated bool, total int) string {
	if len(rows) == 0 {
		return "Empty result set"
	}

	var columns []string
	for col := range rows[0] {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	var sb strings.Builder

	// Header
	sb.WriteString("| ")
	for _, col := range columns {
		displayCol := col
		if len(displayCol) > defaultMaxColWidth {
			displayCol = displayCol[:defaultMaxColWidth-3] + "..."
		}
		sb.WriteString(fmt.Sprintf("%-*s | ", defaultMaxColWidth, displayCol))
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString("|")
	for range columns {
		for i := 0; i < defaultMaxColWidth+2; i++ {
			sb.WriteString("-")
		}
		sb.WriteString("|")
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range rows {
		sb.WriteString("| ")
		for _, col := range columns {
			val := row[col]
			s := fmt.Sprintf("%v", val)
			if len(s) > defaultMaxColWidth {
				s = s[:defaultMaxColWidth-3] + "..."
			}
			sb.WriteString(fmt.Sprintf("%-*s | ", defaultMaxColWidth, s))
		}
		sb.WriteString("\n")
	}

	if truncated {
		sb.WriteString(fmt.Sprintf("\n... (showing %d of %d rows)\n", len(rows), total))
		sb.WriteString("Use WHERE clause or LIMIT to narrow results.\n")
	}

	return sb.String()
}

func (t *SQLQueryTool) formatSummary(rows []database.Row, total int, sqlQuery string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Large result set: %d rows total.\n\n", total))

	// Column analysis
	if len(rows) > 0 {
		var columns []string
		for col := range rows[0] {
			columns = append(columns, col)
		}
		sort.Strings(columns)

		sb.WriteString("Column summary (from first 100 rows):\n")
		sampleSize := 100
		if len(rows) < sampleSize {
			sampleSize = len(rows)
		}

		for _, col := range columns {
			// Count distinct values
			seen := make(map[string]bool)
			nullCount := 0
			for i := 0; i < sampleSize; i++ {
				val := fmt.Sprintf("%v", rows[i][col])
				if val == "<nil>" || val == "" {
					nullCount++
				} else {
					seen[val] = true
				}
			}
			sb.WriteString(fmt.Sprintf("  - %s: %d distinct values", col, len(seen)))
			if nullCount > 0 {
				sb.WriteString(fmt.Sprintf(", %d nulls", nullCount))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString(fmt.Sprintf("\nTotal rows: %d\n", total))
	sb.WriteString("Use WHERE clause, GROUP BY, or LIMIT to narrow results.\n")
	sb.WriteString("Example: SELECT * FROM table WHERE column = 'value' LIMIT 50\n")

	return sb.String()
}
