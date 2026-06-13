package database

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/tools"
)

// SQLQueryTool executes SQL queries on a database.
type SQLQueryTool struct {
	registry *database.Registry
}

// NewSQLQueryTool creates a new SQL query tool.
func NewSQLQueryTool(registry *database.Registry) *SQLQueryTool {
	return &SQLQueryTool{registry: registry}
}

func (t *SQLQueryTool) Name() string        { return "sql_query" }
func (t *SQLQueryTool) Description() string { return "Execute SQL SELECT query and return results" }

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

	// Format as table
	result := t.formatRows(rows)
	return &tools.ToolResult{ForLLM: result, ForUser: fmt.Sprintf("Found %d rows", len(rows))}, nil
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

func (t *SQLQueryTool) formatRows(rows []database.Row) string {
	if len(rows) == 0 {
		return "Empty result set"
	}

	// Get columns from first row
	var columns []string
	for col := range rows[0] {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	var sb strings.Builder

	// Header
	sb.WriteString("| ")
	for _, col := range columns {
		sb.WriteString(fmt.Sprintf("%-20s | ", col))
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString("|")
	for range columns {
		sb.WriteString("----------------------|")
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range rows {
		sb.WriteString("| ")
		for _, col := range columns {
			val := row[col]
			s := fmt.Sprintf("%v", val)
			if len(s) > 20 {
				s = s[:17] + "..."
			}
			sb.WriteString(fmt.Sprintf("%-20s | ", s))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
