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
	sql := GenerateDeleteRollback("users", "id = ?", []interface{}{1})
	if sql == "" {
		t.Error("expected non-empty rollback SQL")
	}
	if !strings.Contains(sql, "INSERT INTO") {
		t.Error("expected INSERT INTO in rollback SQL")
	}
}

func TestGenerateUpdateRollback(t *testing.T) {
	sql := GenerateUpdateRollback("users", "name = 'bob'", "id = 1", map[string]interface{}{"name": "alice"})
	if sql == "" {
		t.Error("expected non-empty rollback SQL")
	}
	if !strings.Contains(sql, "UPDATE") {
		t.Error("expected UPDATE in rollback SQL")
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
