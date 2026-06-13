# Spec: Database Tools — 分层 Tool 设计

## [S1] 工具架构决策

**决策：路径 B — 分层 Tool 设计**

不使用统一 Driver 接口。而是按数据库类型分工具：

```
SQL 数据库 (SQLite/MySQL/PostgreSQL):
  → sql_query, sql_schema, sql_analyze, sql_insert, sql_update, sql_delete
  → 底层用 database/sql 统一接口

NoSQL (Redis):
  → redis_get, redis_set, redis_del, redis_keys, redis_hgetall, redis_lrange, redis_info
  → 底层用 go-redis 专用接口

VectorDB (Qdrant/Milvus):
  → vector_search, vector_insert, vector_delete, vector_collections
  → 底层用各 DB 的 HTTP API
```

**理由：**
- LLM 擅长从 20+ 工具中选 5 个，不擅长把 Redis 伪装成 SQL
- 每种数据库有最自然的操作方式
- 扩展新数据库只需要加工具，不改接口

## [S2] SQL 工具集

### sql_query
执行 SQL 查询，返回结果。

**Input:** `{"database": "mysql", "sql": "SELECT * FROM users WHERE id = ?", "args": [1]}`
**Output:** 格式化表格或 JSON 结果
**Safety:** 只读（SELECT），始终允许

### sql_schema
返回数据库 schema 信息。

**Input:** `{"database": "sqlite", "table": "users"}` (table 可选，省略返回全量)
**Output:** 表结构描述
**Safety:** 只读，始终允许

### sql_analyze
分析表的数据分布。

**Input:** `{"database": "mysql", "table": "orders"}`
**Output:** 行数、空值统计、 distinct 值、数值范围
**Safety:** 只读，始终允许

### sql_insert
插入数据。

**Input:** `{"database": "sqlite", "table": "users", "values": {"name": "alice", "email": "a@b.com"}}`
**Safety:** 需 `--allow-writes`

### sql_update
更新数据。

**Input:** `{"database": "mysql", "table": "users", "values": {"name": "bob"}, "where": "id = ?", "args": [1]}`
**Safety:** 需 `--allow-writes`，WHERE 必填

### sql_delete
删除数据。

**Input:** `{"database": "sqlite", "table": "users", "where": "id = ?", "args": [1]}`
**Safety:** 需 `--allow-writes` + `--confirm-delete`

### sql_refresh_schema
强制刷新 schema 缓存。

**Input:** `{"database": "mysql"}`
**Safety:** 只读，始终允许

## [S3] Redis 工具集

### redis_get
获取 key 的值。

**Input:** `{"database": "redis", "key": "user:1001"}`
**Safety:** 只读，始终允许

### redis_set
设置 key-value。

**Input:** `{"database": "redis", "key": "cache:home", "value": "...", "ttl": 3600}`
**Safety:** 需 `--allow-writes`

### redis_del
删除 key。

**Input:** `{"database": "redis", "key": "cache:old"}`
**Safety:** 需 `--allow-writes` + `--confirm-delete`

### redis_keys
搜索 key 模式。

**Input:** `{"database": "redis", "pattern": "user:*"}`
**Safety:** 只读，始终允许

### redis_hgetall
获取 hash 的所有字段。

**Input:** `{"database": "redis", "key": "user:1001"}`
**Safety:** 只读，始终允许

### redis_lrange
获取 list 范围。

**Input:** `{"database": "redis", "key": "queue:tasks", "start": 0, "stop": -1}`
**Safety:** 只读，始终允许

### redis_info
获取 Redis 服务器信息。

**Input:** `{"database": "redis"}`
**Safety:** 只读，始终允许

## [S4] VectorDB 工具集

### vector_search
语义搜索。

**Input:** `{"database": "qdrant", "collection": "documents", "query": "machine learning", "top_k": 5}`
**Output:** 相似文档列表 + 相似度分数
**Safety:** 只读，始终允许

### vector_insert
插入向量。

**Input:** `{"database": "qdrant", "collection": "documents", "id": "doc-1", "text": "...", "metadata": {...}}`
**Safety:** 需 `--allow-writes`

### vector_delete
删除向量。

**Input:** `{"database": "qdrant", "collection": "documents", "id": "doc-1"}`
**Safety:** 需 `--allow-writes` + `--confirm-delete`

### vector_collections
列出所有 collection。

**Input:** `{"database": "qdrant"}`
**Safety:** 只读，始终允许

## [S5] 安全分级总表

| 工具 | 始终允许 | --allow-writes | --confirm-delete |
|------|---------|----------------|-----------------|
| sql_query | ✅ | | |
| sql_schema | ✅ | | |
| sql_analyze | ✅ | | |
| sql_refresh_schema | ✅ | | |
| sql_insert | | ✅ | |
| sql_update | | ✅ | |
| sql_delete | | | ✅ |
| redis_get | ✅ | | |
| redis_keys | ✅ | | |
| redis_hgetall | ✅ | | |
| redis_lrange | ✅ | | |
| redis_info | ✅ | | |
| redis_set | | ✅ | |
| redis_del | | | ✅ |
| vector_search | ✅ | | |
| vector_collections | ✅ | | |
| vector_insert | | ✅ | |
| vector_delete | | | ✅ |
