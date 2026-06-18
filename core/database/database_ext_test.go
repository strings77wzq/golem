package database

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// Test SQLite schema caching
func TestSQLiteDriver_SchemaCache(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	if err := driver.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer driver.Close()

	// Create a table
	driver.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")

	// First call - should query database
	schema1, err := driver.GetSchema(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if schema1 == "" {
		t.Error("expected non-empty schema")
	}

	// Second call - should use cache
	schema2, err := driver.GetSchema(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if schema1 != schema2 {
		t.Error("schema should be cached")
	}
}

// Test SQLite schema cache expiration
func TestSQLiteDriver_SchemaCacheExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	driver.schemaTTL = 0 // Force expiration
	ctx := context.Background()

	if err := driver.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer driver.Close()

	driver.Execute(ctx, "CREATE TABLE t1 (id INTEGER)")
	schema1, _ := driver.GetSchema(ctx)

	// Add another table
	driver.Execute(ctx, "CREATE TABLE t2 (id INTEGER)")
	time.Sleep(10 * time.Millisecond) // Ensure cache expires

	schema2, _ := driver.GetSchema(ctx)
	if schema1 == schema2 {
		t.Error("schema should be refreshed after cache expiration")
	}
}

// Test SQLite GetSchemaForTable not found
func TestSQLiteDriver_GetSchemaForTable_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	_, err := driver.GetSchemaForTable(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent table")
	}
}

// Test SQLite multiple operations
func TestSQLiteDriver_MultipleOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	// Create table
	driver.Execute(ctx, "CREATE TABLE items (id INTEGER PRIMARY KEY, value TEXT)")

	// Insert multiple rows
	for i := 0; i < 10; i++ {
		_, err := driver.Execute(ctx, "INSERT INTO items (value) VALUES (?)", fmt.Sprintf("item_%d", i))
		if err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
	}

	// Query all
	rows, err := driver.Query(ctx, "SELECT * FROM items")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 10 {
		t.Errorf("expected 10 rows, got %d", len(rows))
	}

	// Query with limit
	rows, err = driver.Query(ctx, "SELECT * FROM items LIMIT 5")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(rows))
	}
}

// Test SQLite concurrent reads
func TestSQLiteDriver_ConcurrentReads(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	driver.Execute(ctx, "CREATE TABLE t (id INTEGER)")
	driver.Execute(ctx, "INSERT INTO t VALUES (1)")

	// Concurrent reads should work
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			_, err := driver.Query(ctx, "SELECT * FROM t")
			if err != nil {
				t.Errorf("concurrent query failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

// Test SQLite empty result
func TestSQLiteDriver_EmptyResult(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	driver.Execute(ctx, "CREATE TABLE empty (id INTEGER)")

	rows, err := driver.Query(ctx, "SELECT * FROM empty")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

// Test SQLite NULL values
func TestSQLiteDriver_NullValues(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	driver.Execute(ctx, "CREATE TABLE t (id INTEGER, val TEXT)")
	driver.Execute(ctx, "INSERT INTO t (id) VALUES (1)") // val is NULL

	rows, err := driver.Query(ctx, "SELECT * FROM t")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatal("expected 1 row")
	}
	// NULL values should be handled (not panic)
	if rows[0]["id"] != int64(1) {
		t.Errorf("expected id=1, got %v", rows[0]["id"])
	}
}

// Test registry thread safety
func TestRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewRegistry()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	driver := NewSQLiteDriver("test", dbPath)
	driver.Connect(context.Background())
	defer driver.Close()

	// Concurrent register and get
	done := make(chan bool, 10)
	for i := 0; i < 5; i++ {
		go func() {
			reg.RegisterSQL("test", driver)
			done <- true
		}()
		go func() {
			reg.GetSQL("test")
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
