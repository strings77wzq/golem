// Package database provides SQL tools for database operations.
package database

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/strings77wzq/golem/core/database"
	"github.com/strings77wzq/golem/core/security"
	"github.com/strings77wzq/golem/core/tools"
	"github.com/strings77wzq/golem/foundation/logger"
)

const (
	defaultMaxRows     = 50
	defaultMaxColWidth = 30
	summaryThreshold   = 100
)

// SQLQueryTool executes SQL queries on a database.
type SQLQueryTool struct {
	registry   *database.Registry
	maxRows    int
	permLevel  security.PermissionLevel
	auditFn    func(entry security.AuditEntry)
	secHandler security.SecurityEventHandler
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

// auditError records a failed Query/Execute so success and failure rates can
// be reconciled from the audit log.
func (t *SQLQueryTool) auditError(ctx context.Context, op, dbName, table, sql string, execErr error) {
	if t.auditFn == nil {
		return
	}
	t.auditFn(security.AuditEntry{
		Operation:  op,
		Database:   dbName,
		Table:      table,
		SQL:        sql,
		Status:     "error",
		ExecutedBy: fmt.Sprintf("error: %v", execErr),
		TraceID:    logger.TraceIDFromContext(ctx),
	})
}

// SetSecurityEventHandler sets the security event handler for metrics.
func (t *SQLQueryTool) SetSecurityEventHandler(h security.SecurityEventHandler) {
	t.secHandler = h
}

func (t *SQLQueryTool) Name() string { return "sql_query" }
func (t *SQLQueryTool) Description() string {
	return "Execute SQL SELECT query and return results (auto-truncated to 50 rows)"
}

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
		if t.auditFn != nil {
			t.auditFn(security.AuditEntry{
				Operation: classifyOpName(sqlQuery),
				Database:  dbName,
				Table:     extractTableName(sqlQuery),
				SQL:       sqlQuery,
				Status:    "denied",
				TraceID:   logger.TraceIDFromContext(ctx),
			})
		}
		if t.secHandler != nil {
			t.secHandler(security.EventSQLDenied, map[string]string{
				"operation": classifyOpName(sqlQuery),
				"database":  dbName,
				"reason":    err.Error(),
			})
		}
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
			if t.auditFn != nil {
				t.auditFn(security.AuditEntry{
					Operation: classifyOpName(sqlQuery),
					Database:  dbName,
					Table:     extractTableName(sqlQuery),
					SQL:       sqlQuery,
					Status:    "denied",
					TraceID:   logger.TraceIDFromContext(ctx),
				})
			}
			if t.secHandler != nil {
				t.secHandler(security.EventSQLDenied, map[string]string{
					"operation": classifyOpName(sqlQuery),
					"database":  dbName,
					"reason":    "quality gate",
				})
			}
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
			t.auditError(ctx, classifyOpName(sqlQuery), dbName, extractTableName(sqlQuery), sqlQuery, err)
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
	upper := strings.TrimSpace(strings.ToUpper(sqlQuery))

	// Pre-update snapshot: capture old values BEFORE executing the UPDATE
	// so we can generate a proper rollback SQL.
	var preSnapshot map[string]interface{}
	if strings.HasPrefix(upper, "UPDATE") {
		table := extractTableName(sqlQuery)
		whereClause := extractWhereClause(sqlQuery)
		sanitizedWhere, whereArgs := sanitizeWhereClause(whereClause)
		snapshotSQL := fmt.Sprintf(`SELECT * FROM "%s" WHERE %s LIMIT 1`, table, sanitizedWhere)
		if snapshotRows, snapErr := driver.Query(ctx, snapshotSQL, whereArgs...); snapErr == nil && len(snapshotRows) > 0 {
			preSnapshot = snapshotRows[0]
		}
		// Snapshot failure is non-fatal — we'll generate rollback as best-effort
	}

	res, err := driver.Execute(ctx, sqlQuery, queryArgs...)
	if err != nil {
		t.auditError(ctx, classifyOpName(sqlQuery), dbName, extractTableName(sqlQuery), sqlQuery, err)
		return &tools.ToolResult{ForLLM: fmt.Sprintf("Execute error: %v", err), IsError: true}, nil
	}

	// Generate rollback SQL for DELETE/UPDATE operations. The new generators
	// return (sql, error) so malformed/injectable rollback is never emitted;
	// on error we record a documented "rollback unavailable" note in the audit
	// trail rather than shipping broken SQL (e.g. UPDATE … SET  WHERE …).
	rollbackSQL, rollbackErr := "", error(nil)
	switch {
	case strings.HasPrefix(upper, "DELETE"):
		whereClause := extractWhereClause(sqlQuery)
		sanitizedWhere, whereArgs := sanitizeWhereClause(whereClause)
		rollbackSQL, rollbackErr = security.GenerateDeleteRollback(extractTableName(sqlQuery), sanitizedWhere, whereArgs)
	case strings.HasPrefix(upper, "UPDATE"):
		table := extractTableName(sqlQuery)
		whereClause := extractWhereClause(sqlQuery)
		sanitizedWhere, _ := sanitizeWhereClause(whereClause)
		if preSnapshot != nil {
			rollbackSQL, rollbackErr = security.GenerateUpdateRollback(table, sanitizedWhere, preSnapshot)
		} else {
			rollbackErr = fmt.Errorf("UPDATE rollback unavailable: pre-update snapshot failed")
		}
	}

	// Audit logging
	auditRollback := rollbackSQL
	if rollbackErr != nil {
		auditRollback = fmt.Sprintf("-- rollback unavailable: %v", rollbackErr)
	}
	if t.auditFn != nil {
		t.auditFn(security.AuditEntry{
			Operation:    classifyOpName(sqlQuery),
			Database:     dbName,
			Table:        extractTableName(sqlQuery),
			SQL:          sqlQuery,
			AffectedRows: res.RowsAffected,
			RollbackSQL:  auditRollback,
			Status:       "success",
			TraceID:      logger.TraceIDFromContext(ctx),
		})
	}

	result := fmt.Sprintf("OK: %d rows affected", res.RowsAffected)
	if rollbackSQL != "" {
		result += fmt.Sprintf("\nRollback SQL: %s", rollbackSQL)
		// Warn if multi-row UPDATE but snapshot only captured one row
		if res.RowsAffected > 1 && rollbackErr == nil {
			result += "\n⚠️  Note: Rollback only restores 1 row (snapshot used LIMIT 1). For full rollback, use WHERE clause to target specific rows."
		}
	}

	return &tools.ToolResult{
		ForLLM:  result,
		ForUser: fmt.Sprintf("Executed: %d rows affected", res.RowsAffected),
	}, nil
}

// classifyOperation determines the permission level needed for a SQL operation.
func classifyOperation(sql string) security.PermissionLevel {
	normalized := normalizeSQL(sql)
	if normalized == "" {
		return security.PermAdmin
	}
	upper := strings.ToUpper(normalized)
	switch {
	case strings.HasPrefix(upper, "SELECT"), strings.HasPrefix(upper, "WITH"):
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

// normalizeSQL strips comments, collapses whitespace, and rejects multi-statement SQL.
// Conservative: if we can't confidently classify it, we deny it.
func normalizeSQL(sql string) string {
	s := sql

	// Strip block comments /* ... */ (iterate to handle nesting)
	for {
		start := strings.Index(s, "/*")
		if start == -1 {
			break
		}
		end := strings.Index(s[start+2:], "*/")
		if end == -1 {
			s = s[:start]
			break
		}
		s = s[:start] + s[start+2+end+2:]
	}

	// Strip single-line comments -- ... and # ...
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		// Strip -- comments first
		if idx := strings.Index(line, "--"); idx != -1 {
			line = line[:idx]
		}
		// Only treat # as comment when it appears at the start of a line
		// (after optional leading whitespace). This prevents stripping #
		// inside string literals like 'O''Brien #test'.
		// NOTE: reads from local `line` (already -- stripped), not the
		// original loop variable, to avoid cascading bugs.
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "#") {
			offset := len(line) - len(trimmed)
			line = line[:offset]
		}
		lines[i] = line
	}
	s = strings.Join(lines, " ")

	// Collapse whitespace
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)

	if s == "" {
		return ""
	}

	// Reject multi-statement (semicolon indicates injection attempt)
	if strings.Contains(s, ";") {
		return ""
	}

	return s
}

// classifyOpName returns the operation name for audit logging.
func classifyOpName(sql string) string {
	normalized := normalizeSQL(sql)
	if normalized == "" {
		return "UNKNOWN"
	}
	upper := strings.ToUpper(normalized)
	switch {
	case strings.HasPrefix(upper, "SELECT"), strings.HasPrefix(upper, "WITH"):
		return "SELECT"
	case strings.HasPrefix(upper, "INSERT"):
		return "INSERT"
	case strings.HasPrefix(upper, "UPDATE"):
		return "UPDATE"
	case strings.HasPrefix(upper, "DELETE"):
		return "DELETE"
	case strings.HasPrefix(upper, "DROP"), strings.HasPrefix(upper, "ALTER"), strings.HasPrefix(upper, "TRUNCATE"):
		return "ADMIN"
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
// It operates on the normalized SQL (comments stripped, whitespace collapsed)
// to prevent comment-based bypass of the QualityGate.
func extractWhereClause(sql string) string {
	normalized := normalizeSQL(sql)
	if normalized == "" {
		return "1=1"
	}
	upper := strings.ToUpper(normalized)
	idx := strings.Index(upper, "WHERE")
	if idx < 0 {
		return "1=1"
	}
	return strings.TrimSpace(normalized[idx+5:])
}

var (
	// reStringLiteral matches single-quoted string literals including escaped quotes:
	// 'value', 'O''Brien', 'it''s a test'. The '' sequence is SQL's escape for a
	// literal single quote inside a string.
	reStringLiteral = regexp.MustCompile(`'[^']*(?:''[^']*)*'`)
	// reNumericLiteral matches integer and decimal numbers
	reNumericLiteral = regexp.MustCompile(`\b\d+(?:\.\d+)?\b`)
)

// sanitizeWhereClause replaces string and numeric literals in a WHERE clause
// with ? placeholders, collecting the original values as args. This prevents
// SQL injection when the WHERE clause is used in rollback SQL generation.
// Keywords (AND, OR, IN, BETWEEN, LIKE, IS, NULL, =, >, <, etc.) are preserved.
func sanitizeWhereClause(where string) (string, []interface{}) {
	if where == "" {
		return where, nil
	}

	var args []interface{}

	// Replace string literals with ?
	result := reStringLiteral.ReplaceAllStringFunc(where, func(match string) string {
		// Strip surrounding quotes
		args = append(args, match[1:len(match)-1])
		return "?"
	})

	// Replace numeric literals with ? (only those not already replaced)
	result = reNumericLiteral.ReplaceAllStringFunc(result, func(match string) string {
		if f, err := strconv.ParseFloat(match, 64); err == nil {
			args = append(args, f)
		} else {
			args = append(args, match)
		}
		return "?"
	})

	return result, args
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

func (t *SQLQueryTool) formatSummary(rows []database.Row, total int, _ string) string {
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
