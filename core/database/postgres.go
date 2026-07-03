package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PostgresDriver implements SQLDriver for PostgreSQL databases.
type PostgresDriver struct {
	db          *sql.DB
	name        string
	dsn         string
	schemaCache string
	schemaTime  time.Time
	schemaTTL   time.Duration
}

// NewPostgresDriver creates a new PostgreSQL driver.
func NewPostgresDriver(name, dsn string) *PostgresDriver {
	return &PostgresDriver{
		name:      name,
		dsn:       dsn,
		schemaTTL: 5 * time.Minute,
	}
}

// Name returns the driver name.
func (d *PostgresDriver) Name() string {
	return d.name
}

// Connect opens the PostgreSQL database.
func (d *PostgresDriver) Connect(ctx context.Context) error {
	db, err := sql.Open("postgres", d.dsn)
	if err != nil {
		return fmt.Errorf("opening postgres: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close() //nolint:errcheck
		return fmt.Errorf("pinging postgres: %w", err)
	}

	d.db = db
	return nil
}

// Query executes a SELECT query.
func (d *PostgresDriver) Query(ctx context.Context, query string, args ...interface{}) ([]Row, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close() //nolint:errcheck

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
func (d *PostgresDriver) Execute(ctx context.Context, query string, args ...interface{}) (Result, error) {
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
func (d *PostgresDriver) GetSchema(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", fmt.Errorf("database not connected")
	}

	if d.schemaCache != "" && time.Since(d.schemaTime) < d.schemaTTL {
		return d.schemaCache, nil
	}

	tables, err := d.Query(ctx,
		"SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name")
	if err != nil {
		return "", fmt.Errorf("listing tables: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Database: PostgreSQL\n\nTables:\n")

	for _, table := range tables {
		name := fmt.Sprintf("%v", table["table_name"])
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
func (d *PostgresDriver) GetSchemaForTable(ctx context.Context, table string) (string, error) {
	if d.db == nil {
		return "", fmt.Errorf("database not connected")
	}

	rows, err := d.Query(ctx,
		`SELECT column_name, data_type, is_nullable, column_default
		 FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name = $1
		 ORDER BY ordinal_position`, table)
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
		col := fmt.Sprintf("%v", row["column_name"])
		typ := fmt.Sprintf("%v", row["data_type"])
		nullable := fmt.Sprintf("%v", row["is_nullable"])

		part := fmt.Sprintf("%s %s", col, typ)
		if nullable == "NO" {
			part += " NOT NULL"
		}
		parts = append(parts, part)
	}

	sb.WriteString(strings.Join(parts, ", "))
	sb.WriteString(")")

	return sb.String(), nil
}

// Ping checks the connection.
func (d *PostgresDriver) Ping(ctx context.Context) error {
	if d.db == nil {
		return fmt.Errorf("database not connected")
	}
	return d.db.PingContext(ctx)
}

// Close closes the connection.
func (d *PostgresDriver) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
