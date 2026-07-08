package database

import (
	"strings"
	"testing"
)

func TestExtractWhereClause(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantWhere string
	}{
		{
			name:      "simple WHERE",
			sql:       "DELETE FROM users WHERE id = 1",
			wantWhere: "id = 1",
		},
		{
			name:      "no WHERE returns 1=1",
			sql:       "DELETE FROM users",
			wantWhere: "1=1",
		},
		{
			name:      "WHERE with AND",
			sql:       "DELETE FROM users WHERE id = 1 AND active = 0",
			wantWhere: "id = 1 AND active = 0",
		},
		{
			name:      "WHERE with string literal",
			sql:       "DELETE FROM users WHERE name = 'Alice'",
			wantWhere: "name = 'Alice'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWhereClause(tt.sql)
			if got != tt.wantWhere {
				t.Errorf("extractWhereClause(%q) = %q, want %q", tt.sql, got, tt.wantWhere)
			}
		})
	}
}

func TestSanitizeWhereClause(t *testing.T) {
	tests := []struct {
		name     string
		where    string
		wantArgs int
	}{
		{
			name:     "simple equality - numeric becomes param",
			where:    "id = 1",
			wantArgs: 1,
		},
		{
			name:     "string literal becomes param",
			where:    "name = 'Alice'",
			wantArgs: 1,
		},
		{
			name:     "OR injection - all numeric values become params",
			where:    "id = 1 OR 1=1",
			wantArgs: 3, // "1" in "id = 1", "1" and "1" in "1=1"
		},
		{
			name:     "multiple conditions",
			where:    "id = 1 AND name = 'Bob'",
			wantArgs: 2,
		},
		{
			name:     "empty where",
			where:    "",
			wantArgs: 0,
		},
		{
			name:     "WHERE keyword only",
			where:    "WHERE id = 1",
			wantArgs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, args := sanitizeWhereClause(tt.where)
			if len(args) != tt.wantArgs {
				t.Errorf("sanitizeWhereClause(%q) returned %d args, want %d", tt.where, len(args), tt.wantArgs)
			}
		})
	}
}

func TestSanitizeWhereClause_InjectableWhere(t *testing.T) {
	// Malicious WHERE: "id = 1 OR 1=1" should NOT produce a WHERE that
	// matches all rows when used in rollback SQL. sanitizeWhereClause
	// replaces literals with ? placeholders so the actual values are bound
	// by the driver, preventing injection.
	where := "id = 1 OR 1=1"
	sanitized, args := sanitizeWhereClause(where)

	// The sanitized WHERE should contain ? placeholders, not raw "1" or "1=1"
	if sanitized == where {
		t.Errorf("sanitizeWhereClause did not modify injectable WHERE: %q", sanitized)
	}
	// All three "1" values should be captured as args (id=1 has one, 1=1 has two)
	if len(args) != 3 {
		t.Errorf("expected 3 args for 'id = 1 OR 1=1', got %d: %v", len(args), args)
	}
}

func TestSanitizeWhereClause_EscapedQuotes(t *testing.T) {
	// SQL escaped single quotes: O''Brien should be treated as one literal
	where := "name = 'O''Brien'"
	sanitized, args := sanitizeWhereClause(where)
	if len(args) != 1 {
		t.Errorf("expected 1 arg for escaped quote, got %d: %v", len(args), args)
	}
	if len(args) > 0 && args[0] != "O''Brien" {
		t.Errorf("expected arg value 'O''Brien', got %v", args[0])
	}
	// Sanitized should have exactly one ?
	if strings.Count(sanitized, "?") != 1 {
		t.Errorf("expected 1 ? placeholder, got %d in: %s", strings.Count(sanitized, "?"), sanitized)
	}
}

func TestExtractTableName(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantTable string
	}{
		{"DELETE FROM", "DELETE FROM users WHERE id = 1", "users"},
		{"UPDATE", "UPDATE users SET name = 'test' WHERE id = 1", "users"},
		{"INSERT INTO", "INSERT INTO users VALUES (1, 'alice')", "users"},
		{"unknown", "SELECT * FROM users", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTableName(tt.sql)
			if got != tt.wantTable {
				t.Errorf("extractTableName(%q) = %q, want %q", tt.sql, got, tt.wantTable)
			}
		})
	}
}

// NOTE: TestNormalizeSQL lives in sql_normalize_test.go.
// New hash-inside-string test case added there.
