package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCreateDemoDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "demo.db")

	// Create a smaller test DB instead of the full demo
	if err := createTestDB(dbPath, 100, 500, 10, 200); err != nil {
		t.Fatalf("createTestDB failed: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	var count int

	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if count != 100 {
		t.Errorf("users: expected 100, got %d", count)
	}

	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM products").Scan(&count)
	if count != 10 {
		t.Errorf("products: expected 10, got %d", count)
	}

	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders").Scan(&count)
	if count != 500 {
		t.Errorf("orders: expected 500, got %d", count)
	}

	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM order_items").Scan(&count)
	if count != 200 {
		t.Errorf("order_items: expected 200, got %d", count)
	}
}

// createTestDB creates a small test database for unit tests.
func createTestDB(path string, users, orders, products, orderItems int) error {
	os.Remove(path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA foreign_keys=ON")

	ctx := context.Background()
	db.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT UNIQUE, role TEXT DEFAULT 'user')`)
	db.ExecContext(ctx, `CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL, category TEXT, stock INTEGER DEFAULT 0)`)
	db.ExecContext(ctx, `CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER, total REAL, status TEXT, created_at DATETIME, FOREIGN KEY (user_id) REFERENCES users(id))`)
	db.ExecContext(ctx, `CREATE TABLE order_items (id INTEGER PRIMARY KEY, order_id INTEGER, product_id INTEGER, quantity INTEGER, price REAL, FOREIGN KEY (order_id) REFERENCES orders(id))`)

	for i := 0; i < users; i++ {
		db.ExecContext(ctx, "INSERT INTO users (name, email, role) VALUES (?, ?, ?)",
			fmt.Sprintf("user_%d", i), fmt.Sprintf("user_%d@test.com", i), "user")
	}
	for i := 0; i < products; i++ {
		db.ExecContext(ctx, "INSERT INTO products (name, price, category) VALUES (?, ?, ?)",
			fmt.Sprintf("product_%d", i), float64(i)*1.5, "general")
	}
	for i := 0; i < orders; i++ {
		db.ExecContext(ctx, "INSERT INTO orders (user_id, total, status) VALUES (?, ?, ?)",
			i%users+1, float64(i)*2.5, "completed")
	}
	for i := 0; i < orderItems; i++ {
		db.ExecContext(ctx, "INSERT INTO order_items (order_id, product_id, quantity, price) VALUES (?, ?, ?, ?)",
			i%orders+1, i%products+1, 1, 1.5)
	}
	return nil
}
