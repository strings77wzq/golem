# Spec: Database Schema — 摘要注入 + 按需详情

## [S1] Schema 管理决策

**决策：路径 2 — 摘要注入 + 按需详情**

```
默认（注入 system prompt）:
  表名列表 + 每表列名（~20 tokens/table）
  Agent 知道"有什么表、有什么列"

按需（agent 调用 sql_schema）:
  完整 DDL、类型、约束、索引、外键
  Agent 需要时才获取详细信息
```

**理由：**
- 50 张表的完整 schema 会吃掉 5000 tokens，没空间给 history
- 摘要保证 agent 知道数据库有什么
- 详情按需获取，节省 token

## [S2] Schema 摘要格式

注入 system prompt 的摘要：

```
Database: MySQL (production)

Tables:
- users (id, name, email, role, created_at)
- orders (id, user_id, total, status, created_at)
- products (id, name, price, stock, category)
- order_items (id, order_id, product_id, quantity)

Relationships:
- orders.user_id → users.id
- order_items.order_id → orders.id
- order_items.product_id → products.id
```

**Token 成本：** ~20 tokens per table。50 张表 = 1000 tokens，可接受。

## [S3] Schema 详情获取

Agent 调用 `sql_schema` tool 获取详细信息：

**请求：** `{"database": "mysql", "table": "orders"}`

**响应：**
```
Table: orders
Columns:
- id (INTEGER, PK, AUTO_INCREMENT)
- user_id (INTEGER, NOT NULL, FK → users.id)
- total (DECIMAL(10,2), NOT NULL)
- status (VARCHAR(50), DEFAULT 'pending')
- created_at (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP)

Indexes:
- PRIMARY KEY (id)
- idx_user_id (user_id)
- idx_status (status)

Row count: 15,234
```

## [S4] Schema 缓存与刷新

**缓存策略：**
1. 首次连接时读取 schema，缓存到内存
2. SQL 查询失败（"table doesn't exist"）→ 自动刷新
3. 提供 `sql_refresh_schema` tool 强制刷新
4. 缓存 TTL: 5 分钟（可配置）

## [S5] 多数据库 Schema

多个数据库连接时，摘要显示所有：

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

## [S6] Schema 自动发现

Agent 不需要预定义任何表。连接数据库后自动发现：

```
1. 连接: sql.Open(driver, dsn)
2. 查询 metadata:
   - SQLite: sqlite_master
   - MySQL: INFORMATION_SCHEMA.TABLES + COLUMNS
   - PostgreSQL: information_schema.tables + columns
3. 构建摘要
4. 注入 system prompt
```

## [S7] 跨数据库查询

**不支持跨库 SQL JOIN。** 跨数据库查询通过分步执行 + LLM 比较实现：

```
用户: "比较 MySQL 和 SQLite 的用户数据"
  → Step 1: sql_query (MySQL) SELECT * FROM users
  → Step 2: sql_query (SQLite) SELECT * FROM users
  → Step 3: LLM 比较两个结果集
```
