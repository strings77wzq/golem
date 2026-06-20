package database

import (
	"testing"

	"github.com/strings77wzq/golem/core/security"
)

func TestNormalizeSQL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple select", "SELECT * FROM users", "SELECT * FROM users"},
		{"leading whitespace", "  SELECT 1", "SELECT 1"},
		{"block comment", "/* comment */ DELETE FROM users", "DELETE FROM users"},
		{"inline comment", "SELECT -- comment\n* FROM users", "SELECT * FROM users"},
		{"hash comment", "SELECT # comment\n* FROM users", "SELECT * FROM users"},
		{"multi-statement rejected", "SELECT 1; DROP TABLE users", ""},
		{"empty after strip", "/* only comment */", ""},
		{"whitespace only", "   \n  \t  ", ""},
		{"empty input", "", ""},
		{"CTE query", "WITH cte AS (SELECT 1) SELECT * FROM cte", "WITH cte AS (SELECT 1) SELECT * FROM cte"},
		{"nested block comment", "SELECT /* a /* b */ c */ 1", "SELECT c */ 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSQL(tt.input)
			if got != tt.want {
				t.Errorf("normalizeSQL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestClassifyOperationNormalized(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want security.PermissionLevel
	}{
		{"select", "SELECT * FROM users", security.PermRead},
		{"cte", "WITH cte AS (SELECT 1) SELECT * FROM cte", security.PermRead},
		{"comment then delete", "/* hack */ DELETE FROM users", security.PermDelete},
		{"comment then drop", "-- bypass\nDROP TABLE users", security.PermAdmin},
		{"multi-statement rejected", "SELECT 1; DELETE FROM users", security.PermAdmin},
		{"unknown operation", "GRANT ALL ON *.* TO 'hacker'", security.PermAdmin},
		{"empty input", "", security.PermAdmin},
		{"whitespace only", "   ", security.PermAdmin},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyOperation(tt.sql)
			if got != tt.want {
				t.Errorf("classifyOperation(%q) = %v, want %v", tt.sql, got, tt.want)
			}
		})
	}
}
