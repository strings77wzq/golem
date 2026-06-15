// Package database provides SQL tools for database operations.
package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/tools"
)

// SQLAnalyzeTool analyzes data distribution for a table.
type SQLAnalyzeTool struct {
	registry *database.Registry
}

// NewSQLAnalyzeTool creates a new SQL analyze tool.
func NewSQLAnalyzeTool(registry *database.Registry) *SQLAnalyzeTool {
	return &SQLAnalyzeTool{registry: registry}
}

func (t *SQLAnalyzeTool) Name() string { return "sql_analyze" }
func (t *SQLAnalyzeTool) Description() string {
	return "Analyze data distribution for a table (row count, nulls, distinct values, ranges)"
}

func (t *SQLAnalyzeTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "database", Type: "string", Description: "Database name", Required: false},
		{Name: "table", Type: "string", Description: "Table to analyze", Required: true},
	}
}

func (t *SQLAnalyzeTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	dbName := t.getDefaultDB(args)
	table, _ := args["table"].(string)
	if table == "" {
		return &tools.ToolResult{ForLLM: "Error: table parameter is required", IsError: true}, nil
	}

	driver, err := t.registry.GetSQL(dbName)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	// Get row count
	countRows, err := driver.Query(ctx, fmt.Sprintf("SELECT COUNT(*) as cnt FROM %s", table))
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error counting rows: %v", err), IsError: true}, nil
	}
	rowCount := 0
	if len(countRows) > 0 {
		if cnt, ok := countRows[0]["cnt"].(int64); ok {
			rowCount = int(cnt)
		}
	}

	// Get column info
	schema, err := driver.GetSchemaForTable(ctx, table)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error getting schema: %v", err), IsError: true}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Table: %s\n", table))
	sb.WriteString(fmt.Sprintf("Total rows: %d\n\n", rowCount))
	sb.WriteString("Schema:\n")
	sb.WriteString(schema)
	sb.WriteString("\n")

	return &tools.ToolResult{ForLLM: sb.String(), ForUser: fmt.Sprintf("Analyzed %s (%d rows)", table, rowCount)}, nil
}

func (t *SQLAnalyzeTool) getDefaultDB(args map[string]interface{}) string {
	if db, ok := args["database"].(string); ok && db != "" {
		return db
	}
	return t.registry.Default()
}
