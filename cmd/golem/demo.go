package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

func newDemoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "demo-db",
		Short: "Create a demo ecommerce database for testing",
		Long:  "Create a SQLite database with sample data (10K users, 100K orders, 500 products, 300K order_items) for testing Golem's database analysis capabilities.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ".golem-demo.db"
			if len(args) > 0 {
				path = args[0]
			}
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			fmt.Println("Creating demo ecommerce database...")
			if err := createDemoDB(path); err != nil {
				return err
			}
			fmt.Printf("Done! Use with: golem agent --db %s\n", path)
			return nil
		},
	}
}

func createDemoDB(path string) error {
	os.Remove(path)

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	_, _ = db.Exec("PRAGMA journal_mode=WAL")
	_, _ = db.Exec("PRAGMA foreign_keys=ON")

	ctx := context.Background()

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		role TEXT DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		price DECIMAL(10,2) NOT NULL,
		category TEXT,
		stock INTEGER DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL,
		total DECIMAL(10,2) NOT NULL,
		status TEXT DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
	CREATE TABLE IF NOT EXISTS order_items (
		id INTEGER PRIMARY KEY,
		order_id INTEGER NOT NULL,
		product_id INTEGER NOT NULL,
		quantity INTEGER NOT NULL,
		price DECIMAL(10,2) NOT NULL,
		FOREIGN KEY (order_id) REFERENCES orders(id),
		FOREIGN KEY (product_id) REFERENCES products(id)
	);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("creating tables: %w", err)
	}

	// Insert data
	rng := rand.New(rand.NewSource(42))

	fmt.Print("  Users... ")
	tx, _ := db.BeginTx(ctx, nil)
	stmt, _ := tx.PrepareContext(ctx, "INSERT INTO users (name, email, role) VALUES (?, ?, ?)")
	for i := 0; i < 10000; i++ {
		_, _ = stmt.ExecContext(ctx,
			fmt.Sprintf("user_%d", i),
			fmt.Sprintf("user_%d@example.com", i),
			[]string{"user", "admin", "moderator"}[rng.Intn(3)])
	}
	stmt.Close()
	_ = tx.Commit()
	fmt.Println("10K")

	categories := []string{"electronics", "clothing", "food", "books", "home"}
	fmt.Print("  Products... ")
	tx, _ = db.BeginTx(ctx, nil)
	stmt, _ = tx.PrepareContext(ctx, "INSERT INTO products (name, price, category, stock) VALUES (?, ?, ?, ?)")
	for i := 0; i < 500; i++ {
		_, _ = stmt.ExecContext(ctx,
			fmt.Sprintf("product_%d", i),
			float64(rng.Intn(10000))/100.0,
			categories[rng.Intn(len(categories))],
			rng.Intn(1000))
	}
	stmt.Close()
	_ = tx.Commit()
	fmt.Println("500")

	statuses := []string{"pending", "completed", "cancelled", "shipped"}
	fmt.Print("  Orders... ")
	tx, _ = db.BeginTx(ctx, nil)
	stmt, _ = tx.PrepareContext(ctx, "INSERT INTO orders (user_id, total, status, created_at) VALUES (?, ?, ?, ?)")
	for i := 0; i < 100000; i++ {
		_, _ = stmt.ExecContext(ctx,
			rng.Intn(10000)+1,
			float64(rng.Intn(50000))/100.0,
			statuses[rng.Intn(len(statuses))],
			time.Now().Add(-time.Duration(rng.Intn(365*2*24))*time.Hour))
	}
	stmt.Close()
	_ = tx.Commit()
	fmt.Println("100K")

	fmt.Print("  Order items... ")
	tx, _ = db.BeginTx(ctx, nil)
	stmt, _ = tx.PrepareContext(ctx, "INSERT INTO order_items (order_id, product_id, quantity, price) VALUES (?, ?, ?, ?)")
	for i := 0; i < 300000; i++ {
		_, _ = stmt.ExecContext(ctx,
			rng.Intn(100000)+1,
			rng.Intn(500)+1,
			rng.Intn(10)+1,
			float64(rng.Intn(10000))/100.0)
	}
	stmt.Close()
	_ = tx.Commit()
	fmt.Println("300K")

	fmt.Printf("  Database: %s\n", path)
	return nil
}
