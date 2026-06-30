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

// Test SQLite GetSchemaForTable found
func TestSQLiteDriver_GetSchemaForTable_Found(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	driver.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")

	schema, err := driver.GetSchemaForTable(ctx, "users")
	if err != nil {
		t.Fatalf("GetSchemaForTable failed: %v", err)
	}
	if schema == "" {
		t.Error("expected non-empty schema")
	}
}

// Test SQLite multiple tables schema
func TestSQLiteDriver_GetSchema_MultipleTables(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	driver.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	driver.Execute(ctx, "CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT)")

	schema, err := driver.GetSchema(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if schema == "" {
		t.Error("expected non-empty schema")
	}
}

// Test SQLite different data types
func TestSQLiteDriver_DifferentDataTypes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	driver.Execute(ctx, "CREATE TABLE types (id INTEGER PRIMARY KEY, int_val INTEGER, real_val REAL, text_val TEXT, blob_val BLOB)")

	// Insert different types
	_, err := driver.Execute(ctx, "INSERT INTO types (int_val, real_val, text_val, blob_val) VALUES (?, ?, ?, ?)",
		42, 3.14, "hello", []byte{0x01, 0x02})
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Query
	rows, err := driver.Query(ctx, "SELECT * FROM types")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
}

// Test SQLite query with parameters
func TestSQLiteDriver_QueryWithParameters(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	driver.Execute(ctx, "CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")
	driver.Execute(ctx, "INSERT INTO t (name, age) VALUES (?, ?)", "Alice", 30)
	driver.Execute(ctx, "INSERT INTO t (name, age) VALUES (?, ?)", "Bob", 25)

	// Query with multiple parameters
	rows, err := driver.Query(ctx, "SELECT * FROM t WHERE age > ?", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}

	// Query with AND condition
	rows, err = driver.Query(ctx, "SELECT * FROM t WHERE name = ? AND age > ?", "Alice", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
}

// Test SQLite error handling
func TestSQLiteDriver_ErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	// Invalid SQL
	_, err := driver.Execute(ctx, "INVALID SQL")
	if err == nil {
		t.Error("expected error for invalid SQL")
	}

	// Query with invalid SQL
	_, err = driver.Query(ctx, "INVALID SQL")
	if err == nil {
		t.Error("expected error for invalid SQL")
	}
}

// Test SQLite schema cache invalidation
func TestSQLiteDriver_SchemaCacheInvalidation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	// Create table and get schema
	driver.Execute(ctx, "CREATE TABLE t1 (id INTEGER)")
	schema1, _ := driver.GetSchema(ctx)

	// Add another table and force cache refresh
	driver.Execute(ctx, "CREATE TABLE t2 (id INTEGER)")
	driver.schemaTime = time.Time{} // Force expiration

	schema2, _ := driver.GetSchema(ctx)
	if schema1 == schema2 {
		t.Error("schema should be refreshed")
	}
}

// Test SQLite large result set
func TestSQLiteDriver_LargeResultSet(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	driver.Connect(ctx)
	defer driver.Close()

	driver.Execute(ctx, "CREATE TABLE t (id INTEGER PRIMARY KEY, value TEXT)")

	// Insert 100 rows
	for i := 0; i < 100; i++ {
		driver.Execute(ctx, "INSERT INTO t (value) VALUES (?)", fmt.Sprintf("value_%d", i))
	}

	// Query all
	rows, err := driver.Query(ctx, "SELECT * FROM t")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 100 {
		t.Errorf("expected 100 rows, got %d", len(rows))
	}
}
