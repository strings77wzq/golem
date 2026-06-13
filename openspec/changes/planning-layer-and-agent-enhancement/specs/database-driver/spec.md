# Spec: Database Drivers — 按类型分层

## [S1] 驱动架构决策

**决策：不使用统一 Driver 接口。按数据库类型分层。**

```
SQL 层: SQLite, MySQL, PostgreSQL
  → 共用 database/sql 接口
  → 各自实现 driver/sql/driver.Driver

NoSQL 层: Redis
  → 专用 go-redis 接口

VectorDB 层: Qdrant, Milvus
  → 专用 HTTP/gRPC 接口
```

## [S2] SQL 数据库

所有 SQL 数据库通过 `database/sql` 标准接口连接。

### SQLite
- 驱动: `modernc.org/sqlite` (纯 Go，无 CGO)
- 连接: `sql.Open("sqlite", "~/.golem/data.db")`
- Schema 查询: `sqlite_master` 表

### MySQL
- 驱动: `github.com/go-sql-driver/mysql` (纯 Go)
- 连接: `sql.Open("mysql", "user:pass@tcp(host:port)/db")`
- Schema 查询: `INFORMATION_SCHEMA`

### PostgreSQL
- 驱动: `github.com/lib/pq` 或 `github.com/jackc/pgx` (纯 Go)
- 连接: `sql.Open("postgres", "postgres://user:pass@host:port/db?sslmode=disable")`
- Schema 查询: `information_schema`

### SQL 共用能力
```go
type SQLDriver struct {
    db     *sql.DB
    name   string
    schema *SchemaCache
}

func (d *SQLDriver) Query(ctx, sql, args) ([]Row, error)      // SELECT
func (d *SQLDriver) Execute(ctx, sql, args) (Result, error)    // INSERT/UPDATE/DELETE
func (d *SQLDriver) GetSchema(ctx) (string, error)             // Schema introspection
func (d *SQLDriver) Ping(ctx) error                            // 连接检查
func (d *SQLDriver) Close() error                              // 关闭连接
```

## [S3] Redis

专用驱动，不用 SQL 接口。

```go
type RedisDriver struct {
    client *redis.Client
    name   string
}

func (d *RedisDriver) Get(ctx, key) (string, error)
func (d *RedisDriver) Set(ctx, key, value, ttl) error
func (d *RedisDriver) Del(ctx, keys ...string) (int64, error)
func (d *RedisDriver) Keys(ctx, pattern) ([]string, error)
func (d *RedisDriver) HGetAll(ctx, key) (map[string]string, error)
func (d *RedisDriver) LRange(ctx, key, start, stop) ([]string, error)
func (d *RedisDriver) Info(ctx) (string, error)
func (d *RedisDriver) GetSchema(ctx) (string, error)  // 返回 key patterns
func (d *RedisDriver) Ping(ctx) error
func (d *RedisDriver) Close() error
```

## [S4] VectorDB

专用驱动，各 DB 独立实现。

### Qdrant
```go
type QdrantDriver struct {
    host   string
    port   int
    client *http.Client
}

func (d *QdrantDriver) Search(ctx, collection, query, topK) ([]SearchResult, error)
func (d *QdrantDriver) Insert(ctx, collection, id, vector, payload) error
func (d *QdrantDriver) Delete(ctx, collection, id) error
func (d *QdrantDriver) Collections(ctx) ([]string, error)
func (d *QdrantDriver) GetSchema(ctx) (string, error)  // collection info
func (d *QdrantDriver) Ping(ctx) error
func (d *QdrantDriver) Close() error
```

## [S5] 驱动注册表

```go
type Registry struct {
    sqlDrivers    map[string]*SQLDriver
    redisDrivers  map[string]*RedisDriver
    qdrantDrivers map[string]*QdrantDriver
}

func NewRegistry() *Registry

// SQL 数据库
func (r *Registry) RegisterSQL(name string, driver *SQLDriver) error
func (r *Registry) GetSQL(name string) (*SQLDriver, error)

// Redis
func (r *Registry) RegisterRedis(name string, driver *RedisDriver) error
func (r *Registry) GetRedis(name string) (*RedisDriver, error)

// VectorDB
func (r *Registry) RegisterQdrant(name string, driver *QdrantDriver) error
func (r *Registry) GetQdrant(name string) (*QdrantDriver, error)

// 通用
func (r *Registry) List() []string
func (r *Registry) Close() error
```

## [S6] 安全规则

- 所有 SQL 使用参数化查询（防注入）
- 读操作始终允许
- 写操作需 `--allow-writes`
- 删除需 `--confirm-delete`
- 连接超时: 5 秒
- 查询超时: 5 秒
