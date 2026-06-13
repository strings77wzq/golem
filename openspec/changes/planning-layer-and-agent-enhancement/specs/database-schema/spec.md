# Spec: Database Schema

## [S1] Schema Discovery (NOT Hardcoded Tables)

The agent does NOT have predefined tables. Instead, it **auto-discovers** whatever schema exists in the connected database.

**Flow:**
1. Agent connects to database (SQLite/MySQL/PostgreSQL/Redis/VectorDB)
2. Calls `GetSchema()` to discover all tables
3. Reads column names, types, constraints, indexes
4. Injects schema summary into system prompt
5. Agent generates SQL based on ACTUAL schema

**This means:**
- Connect to your existing MySQL → agent sees YOUR tables
- Connect to your existing PostgreSQL → agent sees YOUR schema
- Connect to SQLite → agent sees whatever tables exist

## [S2] Schema Introspection

The `SchemaManager` reads database metadata dynamically:

### SQL Databases (SQLite, MySQL, PostgreSQL)

Query `INFORMATION_SCHEMA` or equivalent:

```sql
-- Get all tables
SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';

-- Get columns for a table
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'users';

-- Get primary keys
SELECT column_name FROM information_schema.key_column_usage
WHERE table_name = 'users' AND constraint_type = 'PRIMARY KEY';

-- Get foreign keys
SELECT column_name, referenced_table_name, referenced_column_name
FROM information_schema.key_column_usage
WHERE table_name = 'orders' AND referenced_table_name IS NOT NULL;
```

### Redis

Redis has no schema. `GetSchema()` returns:
```
Redis key-value store
Sample keys: user:1001, session:abc, cache:home
Key patterns: user:*, session:*, cache:*
```

### Vector DB (Qdrant, Milvus)

```go
// Qdrant
GET /collections/{name} → {vectors: {size: 1536, distance: "Cosine"}}

// Milvus
GET /collections/{name} → {fields: [{name: "text", type: "VARCHAR"}, ...]}
```

## [S3] Schema Summary Format

The schema is summarized for system prompt injection:

```
Database: MySQL (myapp)

Tables:
- users (id INT PK, name VARCHAR, email VARCHAR UNIQUE, created_at TIMESTAMP)
- orders (id INT PK, user_id INT FK→users.id, amount DECIMAL, status VARCHAR, created_at TIMESTAMP)
- products (id INT PK, name VARCHAR, price DECIMAL, category VARCHAR)
- order_items (id INT PK, order_id INT FK→orders.id, product_id INT FK→products.id, quantity INT)

Relationships:
- orders.user_id → users.id
- order_items.order_id → orders.id
- order_items.product_id → products.id
```

**Token cost:** ~50-100 tokens per table (depends on column count). Acceptable.

## [S4] Schema Refresh

Schema can change (tables added/dropped/altered). The agent should:

1. Cache schema on first read
2. Re-read schema if SQL query fails with "table doesn't exist"
3. Provide `sql_refresh_schema` tool to force re-read
4. Cache TTL: 5 minutes (configurable)

## [S5] Multi-Database Schema

When multiple databases are connected, schema shows all:

```
=== SQLite (local.db) ===
- users (id, name, email)
- sessions (id, user_id, data)

=== MySQL (production) ===
- orders (id, user_id, amount, status)
- products (id, name, price)

=== Redis (cache) ===
- Key patterns: session:*, user:*, product:*
```

Agent can query across databases using the `database` parameter:
```json
{"database": "mysql", "sql": "SELECT * FROM orders"}
```
