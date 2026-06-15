// Package database provides multi-database support for the AI agent.
// It supports SQL databases (SQLite, MySQL, PostgreSQL), Redis, and VectorDB
// through separate driver types, each with their most natural API.
package database

import (
	"context"
	"fmt"
	"time"
)

// Row represents a single database row as a map of column names to values.
type Row map[string]interface{}

// Result represents the result of a non-SELECT query.
type Result struct {
	RowsAffected int64
	LastInsertID int64
}

// Config holds database connection configuration.
type Config struct {
	Type     string            `json:"type"`               // "sqlite", "mysql", "postgres", "redis", "qdrant"
	DSN      string            `json:"dsn,omitempty"`      // connection string (for SQL)
	Host     string            `json:"host,omitempty"`     // host
	Port     int               `json:"port,omitempty"`     // port
	User     string            `json:"user,omitempty"`     // username
	Password string            `json:"password,omitempty"` // password
	Database string            `json:"database,omitempty"` // database name
	Options  map[string]string `json:"options,omitempty"`  // extra options
}

// SQLDriver is the interface for SQL databases (SQLite, MySQL, PostgreSQL).
type SQLDriver interface {
	// Name returns the driver name.
	Name() string
	// Query executes a SELECT query and returns rows.
	Query(ctx context.Context, sql string, args ...interface{}) ([]Row, error)
	// Execute executes a non-SELECT query (INSERT/UPDATE/DELETE).
	Execute(ctx context.Context, sql string, args ...interface{}) (Result, error)
	// GetSchema returns the database schema as a human-readable string.
	GetSchema(ctx context.Context) (string, error)
	// GetSchemaForTable returns the schema for a specific table.
	GetSchemaForTable(ctx context.Context, table string) (string, error)
	// Ping checks the connection.
	Ping(ctx context.Context) error
	// Close closes the connection.
	Close() error
}

// RedisDriver is the interface for Redis operations.
type RedisDriver interface {
	// Name returns the driver name.
	Name() string
	// Get retrieves a value by key.
	Get(ctx context.Context, key string) (string, error)
	// Set sets a key-value pair with optional TTL.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	// Del deletes one or more keys.
	Del(ctx context.Context, keys ...string) (int64, error)
	// Keys returns keys matching a pattern.
	Keys(ctx context.Context, pattern string) ([]string, error)
	// HGetAll returns all fields of a hash.
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	// LRange returns a range of elements from a list.
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	// Info returns Redis server information.
	Info(ctx context.Context) (string, error)
	// GetSchema returns a description of the Redis data.
	GetSchema(ctx context.Context) (string, error)
	// Ping checks the connection.
	Ping(ctx context.Context) error
	// Close closes the connection.
	Close() error
}

// VectorDriver is the interface for vector database operations.
type VectorDriver interface {
	// Name returns the driver name.
	Name() string
	// Search performs a semantic search.
	Search(ctx context.Context, collection string, query string, topK int) ([]SearchResult, error)
	// Insert adds a vector with metadata.
	Insert(ctx context.Context, collection string, id string, metadata map[string]interface{}) error
	// Delete removes a vector by ID.
	Delete(ctx context.Context, collection string, id string) error
	// Collections lists all collections.
	Collections(ctx context.Context) ([]string, error)
	// GetSchema returns collection info.
	GetSchema(ctx context.Context) (string, error)
	// Ping checks the connection.
	Ping(ctx context.Context) error
	// Close closes the connection.
	Close() error
}

// SearchResult represents a single result from a vector search.
type SearchResult struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// String returns a formatted string representation of search results.
func (r SearchResult) String() string {
	return fmt.Sprintf("ID: %s, Score: %.4f", r.ID, r.Score)
}
