// Package security provides security gates for database operations.
package security

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// PermissionLevel represents the level of access required.
type PermissionLevel int

const (
	PermRead   PermissionLevel = iota // Read-only operations
	PermWrite                         // Write operations
	PermDelete                        // Delete operations
	PermAdmin                         // Admin operations (DROP, ALTER)
)

// GateResult represents the result of a security gate check.
type GateResult struct {
	Passed          bool     `json:"passed"`
	RequiresConfirm bool     `json:"requires_confirm"`
	AffectedRows    int64    `json:"affected_rows,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	RollbackSQL     string   `json:"rollback_sql,omitempty"`
}

// PermissionChecker validates permissions.
type PermissionChecker struct {
	Level      PermissionLevel
	AllowedDBs []string
	DeniedOps  []string
}

// NewPermissionChecker creates a new permission checker.
func NewPermissionChecker(level PermissionLevel) *PermissionChecker {
	return &PermissionChecker{
		Level:      level,
		AllowedDBs: []string{"*"},
		DeniedOps:  []string{"DROP DATABASE", "TRUNCATE", "DROP TABLE", "ALTER USER"},
	}
}

// Check verifies if an operation is allowed.
func (pc *PermissionChecker) Check(op PermissionLevel, database, operation string) error {
	if op > pc.Level {
		return fmt.Errorf("insufficient permissions: requires level %d, have level %d", op, pc.Level)
	}

	if len(pc.AllowedDBs) > 0 && pc.AllowedDBs[0] != "*" {
		allowed := false
		for _, db := range pc.AllowedDBs {
			if db == database {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("database %q not in allowed list", database)
		}
	}

	upperOp := strings.ToUpper(operation)
	for _, denied := range pc.DeniedOps {
		if strings.Contains(upperOp, strings.ToUpper(denied)) {
			return fmt.Errorf("operation %q is explicitly denied", operation)
		}
	}

	return nil
}

// QualityGate validates SQL operations before execution.
type QualityGate struct {
	MaxAffectedRows int
	RequireWhere    bool
	RequireBackup   bool
}

// NewQualityGate creates a quality gate with sensible defaults.
func NewQualityGate() *QualityGate {
	return &QualityGate{
		MaxAffectedRows: 100,
		RequireWhere:    true,
		RequireBackup:   true,
	}
}

// CheckSQL validates a SQL operation.
func (qg *QualityGate) CheckSQL(ctx context.Context, sql string, affectedRows int64) *GateResult {
	result := &GateResult{Passed: true}

	upperSQL := strings.TrimSpace(strings.ToUpper(sql))

	// Check for WHERE clause on DELETE/UPDATE
	if qg.RequireWhere && (strings.HasPrefix(upperSQL, "DELETE") || strings.HasPrefix(upperSQL, "UPDATE")) {
		if !strings.Contains(upperSQL, "WHERE") {
			result.Passed = false
			result.Warnings = append(result.Warnings, "DELETE/UPDATE without WHERE clause is not allowed")
			return result
		}
	}

	// Check affected rows
	if affectedRows > int64(qg.MaxAffectedRows) {
		result.RequiresConfirm = true
		result.Warnings = append(result.Warnings, fmt.Sprintf("Operation will affect %d rows (threshold: %d)", affectedRows, qg.MaxAffectedRows))
	}

	result.AffectedRows = affectedRows
	return result
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// quoteIdentifier wraps a validated identifier in ANSI double quotes. It is
// only called after the identifier has passed the charset check, so this is
// literal quoting of a known-good name, not free-form escaping.
func quoteIdentifier(name string) string {
	return `"` + name + `"`
}

// escapeLiteral doubles single quotes inside a string value for safe ANSI
// SQL literal embedding (e.g. "Bob's pen" -> "Bob”s pen"). The literal is
// then wrapped in single quotes.
func escapeLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// formatValue renders a Go value as a SQL literal for rollback SET clauses.
// Strings are single-quote-escaped; nil becomes NULL; everything else falls
// back to fmt '%v'. This never interpolates raw user content without escaping.
func formatValue(val interface{}) string {
	switch v := val.(type) {
	case nil:
		return "NULL"
	case string:
		return escapeLiteral(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GenerateDeleteRollback generates rollback SQL for a DELETE operation.
// It returns a non-nil error — and an empty SQL string — when the identifier
// fails the charset check or the WHERE clause is empty, so malformed or
// injectable rollback SQL is never emitted. The `?` placeholders in `where`
// are preserved verbatim so the driver can bind `args`.
func GenerateDeleteRollback(table, where string, args []interface{}) (string, error) {
	if !identifierPattern.MatchString(table) {
		return "", fmt.Errorf("invalid table identifier for rollback")
	}
	if strings.TrimSpace(where) == "" {
		return "", fmt.Errorf("rollback requires a non-empty WHERE clause")
	}
	return fmt.Sprintf("INSERT INTO %s_backup SELECT * FROM %s WHERE %s",
		quoteIdentifier(table), quoteIdentifier(table), where), nil
}

// GenerateUpdateRollback generates rollback SQL for an UPDATE operation by
// rebuilding the SET clause from the captured pre-update column values. It
// refuses to emit a malformed UPDATE (empty SET) when oldValues is nil/empty,
// and rejects identifiers outside the strict charset. String values are
// single-quote-escaped so a value containing "'" cannot break out of the
// literal. The previously-redundant setClause parameter is removed.
func GenerateUpdateRollback(table, where string, oldValues map[string]interface{}) (string, error) {
	if !identifierPattern.MatchString(table) {
		return "", fmt.Errorf("invalid table identifier for rollback")
	}
	if strings.TrimSpace(where) == "" {
		return "", fmt.Errorf("rollback requires a non-empty WHERE clause")
	}
	if len(oldValues) == 0 {
		return "", fmt.Errorf("UPDATE rollback requires captured pre-update values")
	}

	// Sort columns for deterministic output (map iteration order is random).
	cols := make([]string, 0, len(oldValues))
	for col := range oldValues {
		if !identifierPattern.MatchString(col) {
			return "", fmt.Errorf("invalid column identifier in rollback values")
		}
		cols = append(cols, col)
	}
	sort.Strings(cols)

	setParts := make([]string, 0, len(cols))
	for _, col := range cols {
		setParts = append(setParts, fmt.Sprintf("%s = %s", quoteIdentifier(col), formatValue(oldValues[col])))
	}

	return fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		quoteIdentifier(table), strings.Join(setParts, ", "), where), nil
}

// AuditEntry represents a logged operation for audit trail.
type AuditEntry struct {
	Operation    string `json:"operation"`
	Database     string `json:"database"`
	Table        string `json:"table"`
	SQL          string `json:"sql"`
	AffectedRows int64  `json:"affected_rows"`
	RollbackSQL  string `json:"rollback_sql,omitempty"`
	Status       string `json:"status"`
	ExecutedBy   string `json:"executed_by"`
}
