package helpers

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// SeedDemoDB creates a temporary SQLite database seeded with fixtures/seed.sql
// and returns its path. The database is removed when the test completes.
func SeedDemoDB(t *testing.T) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "demo.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	// Read seed.sql from the fixtures directory
	seedPath := filepath.Join(RepoRoot(t), "tests", "e2e", "fixtures", "seed.sql")
	seed, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatalf("read seed.sql: %v", err)
	}

	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("exec seed.sql: %v", err)
	}

	// Verify 5 rows were inserted
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 users, got %d", count)
	}

	return dbPath
}
