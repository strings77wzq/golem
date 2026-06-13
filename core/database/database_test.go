package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRegistrySQL(t *testing.T) {
	reg := NewRegistry()

	// Create a temp SQLite database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	driver := NewSQLiteDriver("test-sqlite", dbPath)
	if err := driver.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer driver.Close()

	if err := reg.RegisterSQL("test-sqlite", driver); err != nil {
		t.Fatalf("RegisterSQL failed: %v", err)
	}

	// Get the driver back
	got, err := reg.GetSQL("test-sqlite")
	if err != nil {
		t.Fatalf("GetSQL failed: %v", err)
	}
	if got.Name() != "test-sqlite" {
		t.Errorf("Name = %q, want %q", got.Name(), "test-sqlite")
	}

	// List should show it
	list := reg.List()
	if _, ok := list["test-sqlite"]; !ok {
		t.Error("expected test-sqlite in list")
	}
}

func TestRegistrySQLDuplicate(t *testing.T) {
	reg := NewRegistry()
	driver := NewSQLiteDriver("test", ":memory:")

	if err := reg.RegisterSQL("test", driver); err != nil {
		t.Fatal(err)
	}

	if err := reg.RegisterSQL("test", driver); err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestRegistryRedis(t *testing.T) {
	reg := NewRegistry()

	driver := NewRedisDriver("test-redis")
	if err := reg.RegisterRedis("test-redis", driver); err != nil {
		t.Fatal(err)
	}

	got, err := reg.GetRedis("test-redis")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "test-redis" {
		t.Errorf("Name = %q, want %q", got.Name(), "test-redis")
	}
}

func TestRegistryVector(t *testing.T) {
	reg := NewRegistry()

	driver := NewQdrantDriver("test-qdrant", "localhost", 6333)
	if err := reg.RegisterVector("test-qdrant", driver); err != nil {
		t.Fatal(err)
	}

	got, err := reg.GetVector("test-qdrant")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "test-qdrant" {
		t.Errorf("Name = %q, want %q", got.Name(), "test-qdrant")
	}
}

func TestRegistryDefault(t *testing.T) {
	reg := NewRegistry()
	reg.SetDefault("mydb")

	if reg.Default() != "mydb" {
		t.Errorf("Default = %q, want %q", reg.Default(), "mydb")
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.GetSQL("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent SQL driver")
	}

	_, err = reg.GetRedis("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent Redis driver")
	}

	_, err = reg.GetVector("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent Vector driver")
	}
}

func TestRegistryClose(t *testing.T) {
	reg := NewRegistry()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	driver := NewSQLiteDriver("test-sqlite", dbPath)
	if err := driver.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	reg.RegisterSQL("test-sqlite", driver)

	if err := reg.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestSQLiteDriver(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	driver := NewSQLiteDriver("test", dbPath)
	ctx := context.Background()

	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer driver.Close()

	// Create a table
	_, err := driver.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		t.Fatalf("Create table failed: %v", err)
	}

	// Insert data
	_, err = driver.Execute(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", "alice", "alice@example.com")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Query data
	rows, err := driver.Query(ctx, "SELECT * FROM users WHERE name = ?", "alice")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["name"] != "alice" {
		t.Errorf("name = %v, want alice", rows[0]["name"])
	}

	// Get schema
	schema, err := driver.GetSchema(ctx)
	if err != nil {
		t.Fatalf("GetSchema failed: %v", err)
	}
	if schema == "" {
		t.Error("expected non-empty schema")
	}

	// Ping
	if err := driver.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestSQLiteDriverNotConnected(t *testing.T) {
	driver := NewSQLiteDriver("test", ":memory:")
	ctx := context.Background()

	_, err := driver.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("expected error for disconnected driver")
	}
}

func TestQdrantDriverNotConnected(t *testing.T) {
	driver := NewQdrantDriver("test", "localhost", 6333)
	ctx := context.Background()

	_, err := driver.Search(ctx, "test", "query", 5)
	if err == nil {
		t.Error("expected error for disconnected driver")
	}
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
	list := reg.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
}

func TestResultString(t *testing.T) {
	r := SearchResult{ID: "doc-1", Score: 0.95}
	s := r.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestConfigJSON(t *testing.T) {
	cfg := Config{
		Type:     "sqlite",
		DSN:      "/tmp/test.db",
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "pass",
		Database: "test",
	}

	if cfg.Type != "sqlite" {
		t.Errorf("Type = %q, want sqlite", cfg.Type)
	}
	if cfg.Port != 3306 {
		t.Errorf("Port = %d, want 3306", cfg.Port)
	}
}
