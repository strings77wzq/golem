package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteDriver implements SQLDriver for SQLite databases.
type SQLiteDriver struct {
	db          *sql.DB
	name        string
	dsn         string
	schemaCache string
	schemaTime  time.Time
	schemaTTL   time.Duration
}

// NewSQLiteDriver creates a new SQLite driver.
func NewSQLiteDriver(name, dsn string) *SQLiteDriver {
	return &SQLiteDriver{
		name:      name,
		dsn:       dsn,
		schemaTTL: 5 * time.Minute,
	}
}

// Name returns the driver name.
func (d *SQLiteDriver) Name() string {
	return d.name
}

// Connect opens the SQLite database.
func (d *SQLiteDriver) Connect(ctx context.Context) error {
	db, err := sql.Open("sqlite", d.dsn)
	if err != nil {
		return fmt.Errorf("opening sqlite: %w", err)
	}

	// Enable WAL mode for concurrent reads
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return fmt.Errorf("setting WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return fmt.Errorf("enabling foreign keys: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite supports one writer at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	d.db = db
	return nil
}

// Query executes a SELECT query.
func (d *SQLiteDriver) Query(ctx context.Context, query string, args ...interface{}) ([]Row, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("getting columns: %w", err)
	}

	var result []Row
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		row := make(Row, len(columns))
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for easier handling
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

// Execute runs a non-SELECT query.
func (d *SQLiteDriver) Execute(ctx context.Context, query string, args ...interface{}) (Result, error) {
	if d.db == nil {
		return Result{}, fmt.Errorf("database not connected")
	}

	res, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return Result{}, fmt.Errorf("execute failed: %w", err)
	}

	affected, _ := res.RowsAffected()
	lastID, _ := res.LastInsertId()

	return Result{
		RowsAffected: affected,
		LastInsertID: lastID,
	}, nil
}

// GetSchema returns the full database schema.
func (d *SQLiteDriver) GetSchema(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", fmt.Errorf("database not connected")
	}

	// Check cache
	if d.schemaCache != "" && time.Since(d.schemaTime) < d.schemaTTL {
		return d.schemaCache, nil
	}

	// Get all tables
	tables, err := d.Query(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return "", fmt.Errorf("listing tables: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Database: SQLite\n\nTables:\n")

	for _, table := range tables {
		name := fmt.Sprintf("%v", table["name"])
		tableInfo, err := d.GetSchemaForTable(ctx, name)
		if err != nil {
			sb.WriteString(fmt.Sprintf("- %s (error: %v)\n", name, err))
			continue
		}
		sb.WriteString(tableInfo)
		sb.WriteString("\n")
	}

	d.schemaCache = sb.String()
	d.schemaTime = time.Now()

	return d.schemaCache, nil
}

// GetSchemaForTable returns the schema for a specific table.
func (d *SQLiteDriver) GetSchemaForTable(ctx context.Context, table string) (string, error) {
	if d.db == nil {
		return "", fmt.Errorf("database not connected")
	}

	rows, err := d.Query(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return "", fmt.Errorf("getting table info: %w", err)
	}

	if len(rows) == 0 {
		return "", fmt.Errorf("table %q not found", table)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- %s (", table))

	var parts []string
	for _, row := range rows {
		col := fmt.Sprintf("%v", row["name"])
		typ := fmt.Sprintf("%v", row["type"])
		pk, _ := row["pk"].(int64)
		notnull, _ := row["notnull"].(int64)

		part := fmt.Sprintf("%s %s", col, typ)
		if pk == 1 {
			part += " PK"
		}
		if notnull == 1 {
			part += " NOT NULL"
		}
		parts = append(parts, part)
	}

	sb.WriteString(strings.Join(parts, ", "))
	sb.WriteString(")")

	return sb.String(), nil
}

// Ping checks the connection.
func (d *SQLiteDriver) Ping(ctx context.Context) error {
	if d.db == nil {
		return fmt.Errorf("database not connected")
	}
	return d.db.PingContext(ctx)
}

// Close closes the connection.
func (d *SQLiteDriver) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
