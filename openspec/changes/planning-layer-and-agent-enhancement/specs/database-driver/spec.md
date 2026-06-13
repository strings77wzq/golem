# Spec: Database Driver Abstraction

## [S1] Driver Interface

All database types implement the `Driver` interface:

```go
type Driver interface {
    Name() string
    Connect(ctx context.Context, config Config) error
    Close() error
    Query(ctx context.Context, sql string, args ...interface{}) ([]Row, error)
    Execute(ctx context.Context, sql string, args ...interface{}) (Result, error)
    GetSchema(ctx context.Context) (string, error)
    Ping(ctx context.Context) error
}
```

## [S2] SQLite Driver

Pure Go implementation via `modernc.org/sqlite`.

**Config:**
```go
Config{Type: "sqlite", DSN: "~/.golem/data.db"}
```

**Features:**
- Auto-create database file if not exists
- WAL mode for concurrent reads
- Foreign keys enabled by default
- Schema introspection via `sqlite_master`

## [S3] MySQL Driver

Uses `go-sql-driver/mysql` (already in Go ecosystem, pure Go).

**Config:**
```go
Config{Type: "mysql", Host: "localhost", Port: 3306, User: "root", Password: "***", Database: "myapp"}
```

**DSN format:** `user:password@tcp(host:port)/database`

**Features:**
- Connection pooling (max 10 connections)
- Query timeout (5 seconds default)
- Schema introspection via `INFORMATION_SCHEMA`

## [S4] PostgreSQL Driver

Uses `lib/pq` or `pgx` (pure Go).

**Config:**
```go
Config{Type: "postgres", Host: "localhost", Port: 5432, User: "postgres", Password: "***", Database: "myapp"}
```

**DSN format:** `postgres://user:password@host:port/database?sslmode=disable`

**Features:**
- Connection pooling
- Schema introspection via `information_schema`
- Support for JSON/JSONB columns

## [S5] Redis Driver

Uses `go-redis/redis` (pure Go).

**Config:**
```go
Config{Type: "redis", Host: "localhost", Port: 6379, Password: "***"}
```

**Operations:**
- `GET key` → value
- `SET key value` → OK
- `DEL key` → count
- `KEYS pattern` → list
- `HGETALL hash` → map
- `LRANGE list 0 -1` → list
- `INFO` → server info

**Schema:** No traditional schema. GetSchema returns "Redis key-value store" with key patterns.

## [S6] Vector DB Driver

Support for Qdrant, Milvus, or pgvector.

**Config (Qdrant):**
```go
Config{Type: "qdrant", Host: "localhost", Port: 6333, Options: map[string]string{"collection": "documents"}}
```

**Operations:**
- `search query top_k` → semantic search results
- `insert id text vector` → add vector
- `delete id` → remove vector
- `collections` → list collections

**Schema:** Returns collection info with vector dimensions and point count.

## [S7] Driver Registry

```go
type Registry struct {
    drivers map[string]Driver
    configs map[string]Config
}

func NewRegistry() *Registry

// Register adds a driver for a database type.
func (r *Registry) Register(name string, driver Driver) error

// Connect connects to a database using the given config.
func (r *Registry) Connect(ctx context.Context, name string, config Config) error

// Get returns a connected driver by name.
func (r *Registry) Get(name string) (Driver, error)

// List returns all connected database names.
func (r *Registry) List() []string

// Close closes all connections.
func (r *Registry) Close() error
```

## [S8] Safety Rules

- All SQL queries use parameterized arguments (no string interpolation)
- Read-only operations (SELECT) always allowed
- Write operations (INSERT/UPDATE/DELETE) require `--allow-writes` flag
- DELETE requires `--confirm-delete` flag
- Connection timeout: 5 seconds
- Query timeout: 5 seconds
