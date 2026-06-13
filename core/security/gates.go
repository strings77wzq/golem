package security

import (
	"context"
	"fmt"
	"strings"
)

// PermissionLevel represents the level of access required.
type PermissionLevel int

const (
	PermRead    PermissionLevel = iota // Read-only operations
	PermWrite                          // Write operations
	PermDelete                         // Delete operations
	PermAdmin                          // Admin operations (DROP, ALTER)
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

// GenerateDeleteRollback generates rollback SQL for a DELETE operation.
func GenerateDeleteRollback(table, where string, args []interface{}) string {
	// Build INSERT INTO ... SELECT * FROM ... WHERE ... query
	argPlaceholders := make([]string, len(args))
	for i := range args {
		argPlaceholders[i] = fmt.Sprintf("%v", args[i])
	}

	return fmt.Sprintf("INSERT INTO %s_backup SELECT * FROM %s WHERE %s",
		table, table, where)
}

// GenerateUpdateRollback generates rollback SQL for an UPDATE operation.
func GenerateUpdateRollback(table, setClause, where string, oldValues map[string]interface{}) string {
	var setParts []string
	for col, val := range oldValues {
		setParts = append(setParts, fmt.Sprintf("%s = '%v'", col, val))
	}

	return fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		table, strings.Join(setParts, ", "), where)
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
