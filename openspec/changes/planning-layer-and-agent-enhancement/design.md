# Design: Cloud-Native AI Agent with Infrastructure & Data Intelligence

## Context

Golem needs to become a cloud-native AI agent that understands the full stack: databases, containers, and orchestration. The user's team uses Docker, K8s, MySQL, PostgreSQL, Redis, and vector databases. Golem should be able to operate on all of these.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      User Request                            │
│  "把 golen 服务部署到 k8s，用 mysql 做后端，redis 做缓存"       │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                     PLANNER                                   │
│                                                               │
│  Step 1: Build Docker image                                   │
│  Step 2: Push to registry                                     │
│  Step 3: Create MySQL database and tables                     │
│  Step 4: Configure Redis connection                           │
│  Step 5: Apply K8s manifests                                  │
│  Step 6: Verify rollout                                       │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                   TOOL SELECTOR                                │
│                                                               │
│  Step 1 → docker_build                                        │
│  Step 2 → docker_push (exec: docker push)                    │
│  Step 3 → sql_query (MySQL)                                   │
│  Step 4 → redis_set                                           │
│  Step 5 → k8s_apply                                           │
│  Step 6 → k8s_get (check pods)                               │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                    TOOL LAYER                                  │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │   Database   │  │   Infra      │  │   Existing       │   │
│  │              │  │              │  │                  │   │
│  │ SQLite       │  │ Docker       │  │ exec, file_read  │   │
│  │ MySQL        │  │ K8s          │  │ file_write       │   │
│  │ PostgreSQL   │  │ Helm         │  │ web_search       │   │
│  │ Redis        │  │              │  │                  │   │
│  │ VectorDB     │  │              │  │                  │   │
│  └──────────────┘  └──────────────┘  └──────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow: Multi-Database Query

```
User: "比较 MySQL 用户表和 SQLite 用户表的数据差异"
  │
  ▼
Planner decomposes:
  Step 1: Query MySQL users
  Step 2: Query SQLite users
  Step 3: Compare results
  │
  ▼
Step 1: ToolSelector → sql_query (mysql)
  LLM generates: SELECT * FROM users
  Execute on MySQL connection → 500 rows
  │
  ▼
Step 2: ToolSelector → sql_query (sqlite)
  LLM generates: SELECT * FROM users
  Execute on SQLite connection → 3 rows
  │
  ▼
Step 3: ToolSelector → (no tool, LLM analyzes)
  LLM compares: "MySQL has 500 users, SQLite has 3.
  MySQL has columns X, Y that SQLite doesn't."
  │
  ▼
Final response with comparison report
```

## Key Interfaces

### Database Abstraction

```go
// core/database/driver.go

// Driver is the interface all database types must implement.
type Driver interface {
    // Name returns the driver name (e.g., "sqlite", "mysql").
    Name() string
    
    // Connect establishes connection to the database.
    Connect(ctx context.Context, config Config) error
    
    // Close closes the connection.
    Close() error
    
    // Query executes a SELECT query.
    Query(ctx context.Context, sql string, args ...interface{}) ([]Row, error)
    
    // Execute executes a non-SELECT query.
    Execute(ctx context.Context, sql string, args ...interface{}) (Result, error)
    
    // GetSchema returns the database schema.
    GetSchema(ctx context.Context) (string, error)
    
    // Ping checks connectivity.
    Ping(ctx context.Context) error
}

type Row map[string]interface{}

type Result struct {
    RowsAffected int64
    LastInsertID  int64
}

type Config struct {
    Type     string            // "sqlite", "mysql", "postgres", "redis", "qdrant"
    DSN      string            // connection string (for SQL databases)
    Host     string            // host (for network databases)
    Port     int               // port
    User     string            // username
    Password string            // password
    Database string            // database name
    Options  map[string]string // extra options
}

// core/database/registry.go

// Registry manages multiple database connections.
type Registry struct {
    drivers map[string]Driver
}

func NewRegistry() *Registry

// Register adds a database driver.
func (r *Registry) Register(name string, driver Driver) error

// Get returns a driver by name.
func (r *Registry) Get(name string) (Driver, error)

// List returns all registered driver names.
func (r *Registry) List() []string
```

### Database Tools

```go
// core/tools/database/sql_query.go

type SQLQueryTool struct {
    registry *database.Registry
    defaultDB string  // default database name
}

// Execute runs SQL on the specified database.
// Input: {"database": "mysql", "sql": "SELECT ...", "args": [...]}
// If database is omitted, uses default.

// core/tools/database/sql_schema.go

type SQLSchemaTool struct {
    registry *database.Registry
}

// Execute returns schema for a database.
// Input: {"database": "sqlite"} or {"database": "mysql", "table": "users"}

// core/tools/database/sql_analyze.go

type SQLAnalyzeTool struct {
    registry *database.Registry
}

// Execute analyzes data distribution.
// Input: {"database": "mysql", "table": "users"}
```

### Infrastructure Tools

```go
// core/tools/infra/docker.go

type DockerTool struct {
    executor command.Executor
}

// Execute runs docker commands.
// Input: {"action": "build", "context": ".", "tag": "golem:latest"}
// Input: {"action": "ps", "filter": "status=running"}
// Input: {"action": "logs", "container": "golem-1", "tail": 100}

// core/tools/infra/kubectl.go

type KubectlTool struct {
    executor command.Executor
}

// Execute runs kubectl commands.
// Input: {"action": "get", "resource": "pods", "namespace": "default"}
// Input: {"action": "apply", "file": "deployment.yaml"}
// Input: {"action": "scale", "deployment": "golem", "replicas": 3}

// core/tools/infra/helm.go

type HelmTool struct {
    executor command.Executor
}

// Execute runs helm commands.
// Input: {"action": "install", "release": "golem", "chart": "./helm/golem"}
```

## Decision Points

### D1. Database Connection Strategy

**Option A: Connection per query** — Open/close connection for each query.
- Pro: Simple, no connection management
- Con: Slow for repeated queries

**Option B: Persistent connections** — Keep connections open.
- Pro: Fast
- Con: Connection pool management, cleanup

**Decision: Option B with lazy connection.** Connect on first use, keep alive, close on agent shutdown. Connection pool managed by the driver.

### D2. Multi-Database Routing

How does the agent know which database to query?

**Option A: Explicit specification** — User specifies database name in each query.
- Pro: Clear, no ambiguity
- Con: Verbose

**Option B: Default database** — Agent uses default database unless specified.
- Pro: Convenient for single-database users
- Con: May query wrong database

**Decision: Option B with explicit override.** Default database set via `--db` flag. Agent can specify `database` parameter in tool calls to query other databases.

### D3. Infrastructure Tool Safety

**Option A: Allow all commands** — Agent can run any docker/kubectl command.
- Pro: Maximum flexibility
- Con: Dangerous (can delete production resources)

**Option B: Restricted commands** — Only allow safe read operations.
- Pro: Safe
- Con: Very limited

**Decision: Tiered safety.**
- Read-only operations (ps, get, logs, describe): always allowed
- Write operations (build, apply, scale): require `--allow-infra` flag
- Destructive operations (delete, rm): require `--allow-infra` AND `--confirm-destructive`

## Risks

- **Multi-database complexity**: 5 database drivers × tools = significant code. Mitigation: start with SQLite + MySQL, add others incrementally.
- **Infrastructure safety**: Docker/K8s commands can be destructive. Mitigation: tiered safety with explicit flags.
- **Connection management**: Multiple database connections consume resources. Mitigation: lazy connection + connection pool + cleanup on shutdown.
- **Token overhead**: Multiple database schemas in system prompt. Mitigation: inject only default database schema, others on request.
