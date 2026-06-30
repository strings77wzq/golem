package security

import (
	"context"
	"strings"
	"testing"
)

func TestPermissionCheckerReadAllowed(t *testing.T) {
	pc := NewPermissionChecker(PermRead)
	if err := pc.Check(PermRead, "mydb", "SELECT * FROM users"); err != nil {
		t.Errorf("expected no error for read, got: %v", err)
	}
}

func TestPermissionCheckerWriteBlocked(t *testing.T) {
	pc := NewPermissionChecker(PermRead)
	if err := pc.Check(PermWrite, "mydb", "INSERT INTO users VALUES (1, 'alice')"); err == nil {
		t.Error("expected error for write with read-only permissions")
	}
}

func TestPermissionCheckerWriteAllowed(t *testing.T) {
	pc := NewPermissionChecker(PermWrite)
	if err := pc.Check(PermWrite, "mydb", "INSERT INTO users VALUES (1, 'alice')"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestPermissionCheckerDeleteBlocked(t *testing.T) {
	pc := NewPermissionChecker(PermWrite)
	if err := pc.Check(PermDelete, "mydb", "DELETE FROM users WHERE id = 1"); err == nil {
		t.Error("expected error for delete with write-only permissions")
	}
}

func TestPermissionCheckerDeniedOp(t *testing.T) {
	pc := NewPermissionChecker(PermAdmin)
	if err := pc.Check(PermAdmin, "mydb", "DROP TABLE users"); err == nil {
		t.Error("expected error for denied operation")
	}
}

func TestPermissionCheckerDeniedDB(t *testing.T) {
	pc := NewPermissionChecker(PermRead)
	pc.AllowedDBs = []string{"dev", "staging"}
	if err := pc.Check(PermRead, "production", "SELECT * FROM users"); err == nil {
		t.Error("expected error for denied database")
	}
}

func TestQualityGateNoWhere(t *testing.T) {
	qg := NewQualityGate()
	result := qg.CheckSQL(context.Background(), "DELETE FROM users", 0)
	if result.Passed {
		t.Error("expected failure for DELETE without WHERE")
	}
}

func TestQualityGateWithWhere(t *testing.T) {
	qg := NewQualityGate()
	result := qg.CheckSQL(context.Background(), "DELETE FROM users WHERE id = 1", 1)
	if !result.Passed {
		t.Errorf("expected pass, got warnings: %v", result.Warnings)
	}
}

func TestQualityGateLargeAffected(t *testing.T) {
	qg := NewQualityGate()
	result := qg.CheckSQL(context.Background(), "DELETE FROM users WHERE active = 0", 500)
	if !result.RequiresConfirm {
		t.Error("expected confirmation required for large affected rows")
	}
}

func TestQualityGateUpdateNoWhere(t *testing.T) {
	qg := NewQualityGate()
	result := qg.CheckSQL(context.Background(), "UPDATE users SET name = 'test'", 0)
	if result.Passed {
		t.Error("expected failure for UPDATE without WHERE")
	}
}

func TestGenerateDeleteRollback(t *testing.T) {
	sql, err := GenerateDeleteRollback("users", "id = ?", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sql == "" {
		t.Error("expected non-empty rollback SQL")
	}
	if !strings.Contains(sql, "INSERT INTO") {
		t.Error("expected INSERT INTO in rollback SQL")
	}
	// Where placeholder must be preserved so args can be bound by the driver.
	if !strings.Contains(sql, "WHERE id = ?") {
		t.Errorf("expected preserved WHERE placeholder, got: %s", sql)
	}
	// Identifier must be quoted, never interpolated raw.
	if !strings.Contains(sql, `"users"`) {
		t.Errorf("expected quoted identifier \"users\", got: %s", sql)
	}
}

func TestGenerateDeleteRollbackInjection(t *testing.T) {
	// A malicious table name must be rejected, not interpolated.
	sql, err := GenerateDeleteRollback("users; DROP TABLE users--", "id = ?", nil)
	if err == nil {
		t.Errorf("expected error for injection table, got sql: %s", sql)
	}
	if sql != "" {
		t.Errorf("expected empty SQL on rejection, got: %s", sql)
	}
}

func TestGenerateDeleteRollbackEmptyWhere(t *testing.T) {
	if _, err := GenerateDeleteRollback("users", "", nil); err == nil {
		t.Error("expected error for empty WHERE clause")
	}
}

func TestGenerateDeleteRollbackBadIdentifier(t *testing.T) {
	if _, err := GenerateDeleteRollback("bad name!", "id = ?", nil); err == nil {
		t.Error("expected error for identifier outside charset")
	}
}

func TestGenerateUpdateRollback(t *testing.T) {
	sql, err := GenerateUpdateRollback("users", "id = 1", map[string]interface{}{"name": "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sql == "" {
		t.Error("expected non-empty rollback SQL")
	}
	if !strings.Contains(sql, "UPDATE") {
		t.Error("expected UPDATE in rollback SQL")
	}
	// SET clause must be rebuilt from oldValues, single-quote-escaped.
	if want := `'alice'`; !strings.Contains(sql, want) {
		t.Errorf("expected escaped value %s in SET, got: %s", want, sql)
	}
	if !strings.Contains(sql, "name") {
		t.Errorf("expected column name in SET, got: %s", sql)
	}
	if !strings.Contains(sql, "WHERE id = 1") {
		t.Errorf("expected preserved WHERE, got: %s", sql)
	}
	if !strings.Contains(sql, `"users"`) {
		t.Errorf("expected quoted identifier, got: %s", sql)
	}
}

func TestGenerateUpdateRollbackEscapesSingleQuote(t *testing.T) {
	sql, err := GenerateUpdateRollback("users", "id = 1", map[string]interface{}{"name": "Bob's pen"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// A single quote in the value must be doubled (ANSI escaping), never
	// emitted raw — otherwise the rollback SQL breaks out of the literal.
	if !strings.Contains(sql, `'Bob''s pen'`) {
		t.Errorf("expected single-quote-escaped value, got: %s", sql)
	}
}

func TestGenerateUpdateRollbackNilOldValues(t *testing.T) {
	// Refuse to emit a malformed "UPDATE tbl SET  WHERE …".
	if _, err := GenerateUpdateRollback("users", "id = 1", nil); err == nil {
		t.Error("expected error for nil oldValues")
	}
}

func TestGenerateUpdateRollbackEmptyOldValues(t *testing.T) {
	if _, err := GenerateUpdateRollback("users", "id = 1", map[string]interface{}{}); err == nil {
		t.Error("expected error for empty oldValues")
	}
}

func TestGenerateUpdateRollbackBadIdentifier(t *testing.T) {
	if _, err := GenerateUpdateRollback("users; DROP", "id = 1", map[string]interface{}{"name": "x"}); err == nil {
		t.Error("expected error for injection table")
	}
}

func TestGenerateUpdateRollbackEmptyWhere(t *testing.T) {
	if _, err := GenerateUpdateRollback("users", "", map[string]interface{}{"name": "x"}); err == nil {
		t.Error("expected error for empty WHERE")
	}
}

func TestAuditEntry(t *testing.T) {
	entry := AuditEntry{
		Operation:    "DELETE",
		Database:     "mydb",
		Table:        "users",
		SQL:          "DELETE FROM users WHERE id = 1",
		AffectedRows: 1,
		Status:       "committed",
		ExecutedBy:   "admin",
	}
	if entry.Operation != "DELETE" {
		t.Errorf("Operation = %q, want DELETE", entry.Operation)
	}
}

func TestNewQualityGate(t *testing.T) {
	qg := NewQualityGate()
	if qg.MaxAffectedRows != 100 {
		t.Errorf("MaxAffectedRows = %d, want 100", qg.MaxAffectedRows)
	}
	if !qg.RequireWhere {
		t.Error("expected RequireWhere = true")
	}
	if !qg.RequireBackup {
		t.Error("expected RequireBackup = true")
	}
}
