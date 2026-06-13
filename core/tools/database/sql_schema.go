package database

import (
	"context"
	"fmt"

	"github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/tools"
)

// SQLSchemaTool returns database schema information.
type SQLSchemaTool struct {
	registry *database.Registry
}

// NewSQLSchemaTool creates a new SQL schema tool.
func NewSQLSchemaTool(registry *database.Registry) *SQLSchemaTool {
	return &SQLSchemaTool{registry: registry}
}

func (t *SQLSchemaTool) Name() string        { return "sql_schema" }
func (t *SQLSchemaTool) Description() string { return "Get database schema information (tables, columns, types)" }

func (t *SQLSchemaTool) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "database", Type: "string", Description: "Database name", Required: false},
		{Name: "table", Type: "string", Description: "Specific table name (omit for full schema)", Required: false},
	}
}

func (t *SQLSchemaTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	dbName := t.getDefaultDB(args)
	table, _ := args["table"].(string)

	driver, err := t.registry.GetSQL(dbName)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	var schema string
	if table != "" {
		schema, err = driver.GetSchemaForTable(ctx, table)
	} else {
		schema, err = driver.GetSchema(ctx)
	}

	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Schema error: %v", err), IsError: true}, nil
	}

	return &tools.ToolResult{ForLLM: schema, ForUser: "Schema retrieved"}, nil
}

func (t *SQLSchemaTool) getDefaultDB(args map[string]interface{}) string {
	if db, ok := args["database"].(string); ok && db != "" {
		return db
	}
	return t.registry.Default()
}
