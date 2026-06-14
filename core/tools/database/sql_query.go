package database

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/security"
	"github.com/strings77wzq/golem/core/tools"
)

const (
	defaultMaxRows    = 50
	defaultMaxColWidth = 30
	summaryThreshold   = 100
)

// SQLQueryTool executes SQL queries on a database.
type SQLQueryTool struct {
	registry  *database.Registry
	maxRows   int
	permLevel security.PermissionLevel
	auditFn   func(entry security.AuditEntry)
}

// NewSQLQueryTool creates a new SQL query tool (read-only by default).
func NewSQLQueryTool(registry *database.Registry) *SQLQueryTool {
	return &SQLQueryTool{registry: registry, maxRows: defaultMaxRows, permLevel: security.PermRead}
}

// NewSQLQueryToolWithMaxRows creates a SQL query tool with custom max rows.
func NewSQLQueryToolWithMaxRows(registry *database.Registry, maxRows int) *SQLQueryTool {
	return &SQLQueryTool{registry: registry, maxRows: maxRows, permLevel: security.PermRead}
}

// NewSQLQueryToolWithPermission creates a SQL query tool with custom permission level.
func NewSQLQueryToolWithPermission(registry *database.Registry, permLevel security.PermissionLevel) *SQLQueryTool {
	return &SQLQueryTool{registry: registry, maxRows: defaultMaxRows, permLevel: permLevel}
}

// SetAuditFunc sets the audit callback function.
func (t *SQLQueryTool) SetAuditFunc(fn func(entry security.AuditEntry)) {
	t.auditFn = fn
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

	// Security gate: classify operation and check permissions
	opLevel := classifyOperation(sqlQuery)
	checker := security.NewPermissionChecker(t.permLevel)
	if err := checker.Check(opLevel, dbName, sqlQuery); err != nil {
		return &tools.ToolResult{
			ForLLM:  fmt.Sprintf("Security: %v", err),
			ForUser: "Operation not permitted",
			IsError: true,
		}, nil
	}

	// Quality gate: check WHERE clause for write operations
	if opLevel >= security.PermWrite {
		gate := security.NewQualityGate()
		gateResult := gate.CheckSQL(ctx, sqlQuery, 0)
		if !gateResult.Passed {
			return &tools.ToolResult{
				ForLLM:  fmt.Sprintf("Safety check failed: %v", gateResult.Warnings),
				ForUser: "Operation blocked by safety gate",
				IsError: true,
			}, nil
		}
	}

	driver, err := t.registry.GetSQL(dbName)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Error: %v", err), IsError: true}, nil
	}

	queryArgs := t.parseArgs(args)

	// For read operations, use Query
	if opLevel <= security.PermRead {
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

	// For write operations, use Execute
	res, err := driver.Execute(ctx, sqlQuery, queryArgs...)
	if err != nil {
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Execute error: %v", err), IsError: true}, nil
	}

	// Generate rollback SQL for DELETE/UPDATE operations
	var rollbackSQL string
	upper := strings.TrimSpace(strings.ToUpper(sqlQuery))
	if strings.HasPrefix(upper, "DELETE") {
		rollbackSQL = security.GenerateDeleteRollback(extractTableName(sqlQuery), extractWhereClause(sqlQuery), queryArgs)
	} else if strings.HasPrefix(upper, "UPDATE") {
		rollbackSQL = security.GenerateUpdateRollback(extractTableName(sqlQuery), extractSetClause(sqlQuery), extractWhereClause(sqlQuery), nil)
	}

	// Audit logging
	if t.auditFn != nil {
		t.auditFn(security.AuditEntry{
			Operation:    classifyOpName(sqlQuery),
			Database:     dbName,
			Table:        extractTableName(sqlQuery),
			SQL:          sqlQuery,
			AffectedRows: res.RowsAffected,
			RollbackSQL:  rollbackSQL,
			Status:       "success",
		})
	}

	result := fmt.Sprintf("OK: %d rows affected", res.RowsAffected)
	if rollbackSQL != "" {
		result += fmt.Sprintf("\nRollback SQL: %s", rollbackSQL)
	}

	return &tools.ToolResult{
		ForLLM:  result,
		ForUser: fmt.Sprintf("Executed: %d rows affected", res.RowsAffected),
	}, nil
}

// classifyOperation determines the permission level needed for a SQL operation.
func classifyOperation(sql string) security.PermissionLevel {
	upper := strings.TrimSpace(strings.ToUpper(sql))
	switch {
	case strings.HasPrefix(upper, "SELECT"):
		return security.PermRead
	case strings.HasPrefix(upper, "INSERT"):
		return security.PermWrite
	case strings.HasPrefix(upper, "UPDATE"):
		return security.PermWrite
	case strings.HasPrefix(upper, "DELETE"):
		return security.PermDelete
	case strings.HasPrefix(upper, "DROP"), strings.HasPrefix(upper, "ALTER"), strings.HasPrefix(upper, "TRUNCATE"):
		return security.PermAdmin
	default:
		return security.PermAdmin
	}
}

// classifyOpName returns the operation name for audit logging.
func classifyOpName(sql string) string {
	upper := strings.TrimSpace(strings.ToUpper(sql))
	switch {
	case strings.HasPrefix(upper, "SELECT"):
		return "SELECT"
	case strings.HasPrefix(upper, "INSERT"):
		return "INSERT"
	case strings.HasPrefix(upper, "UPDATE"):
		return "UPDATE"
	case strings.HasPrefix(upper, "DELETE"):
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}

// extractTableName extracts the table name from a SQL statement.
func extractTableName(sql string) string {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	switch {
	case strings.HasPrefix(upper, "DELETE FROM"):
		rest := strings.TrimSpace(sql[11:])
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			return parts[0]
		}
	case strings.HasPrefix(upper, "UPDATE"):
		rest := strings.TrimSpace(sql[6:])
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			return parts[0]
		}
	case strings.HasPrefix(upper, "INSERT INTO"):
		rest := strings.TrimSpace(sql[11:])
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return "unknown"
}

// extractWhereClause extracts the WHERE clause from a SQL statement.
func extractWhereClause(sql string) string {
	upper := strings.ToUpper(sql)
	idx := strings.Index(upper, "WHERE")
	if idx < 0 {
		return "1=1"
	}
	return strings.TrimSpace(sql[idx+5:])
}

// extractSetClause extracts the SET clause from an UPDATE statement.
func extractSetClause(sql string) string {
	upper := strings.ToUpper(sql)
	idx := strings.Index(upper, "SET")
	if idx < 0 {
		return ""
	}
	whereIdx := strings.Index(upper, "WHERE")
	if whereIdx < 0 {
		return strings.TrimSpace(sql[idx+3:])
	}
	return strings.TrimSpace(sql[idx+3 : whereIdx])
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
