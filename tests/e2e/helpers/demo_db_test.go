package helpers

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSeedDemoDB_CreatesValidDB(t *testing.T) {
	dbPath := SeedDemoDB(t)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open seeded db: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 users, got %d", count)
	}

	var name string
	if err := db.QueryRow("SELECT name FROM users WHERE email = ?", "alice@example.com").Scan(&name); err != nil {
		t.Fatalf("query alice: %v", err)
	}
	if name != "Alice" {
		t.Fatalf("expected Alice, got %s", name)
	}
}
