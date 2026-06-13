# Spec: Database Tools

## [S1] sql_query Tool

Execute SQL queries and return results.

**Input:**
```json
{
  "sql": "SELECT * FROM articles WHERE status = ? ORDER BY created_at DESC",
  "args": ["published"]
}
```

**Output:**
```
| id | title | author_id | status | created_at |
|----|-------|-----------|--------|------------|
| 1  | Golem v0.6.0 Release | 1 | published | 2026-06-13 |
| 2  | Getting Started with Go Agents | 2 | published | 2026-06-12 |
```

**Safety:**
- Read-only by default (SELECT only)
- DELETE requires `--allow-writes` flag + confirmation
- All queries use parameterized arguments (no string interpolation)
- Query timeout: 5 seconds

## [S2] sql_schema Tool

Return database schema information.

**Input:**
```json
{"table": "articles"}  // optional, omit for full schema
```

**Output (single table):**
```
Table: articles
Columns:
- id (INTEGER, PK)
- author_id (INTEGER, NOT NULL, FK → users.id)
- title (TEXT, NOT NULL)
- slug (TEXT, NOT NULL, UNIQUE)
- content (TEXT, NOT NULL)
- summary (TEXT)
- status (TEXT, DEFAULT 'draft')
- view_count (INTEGER, DEFAULT 0)
- created_at (DATETIME)
- updated_at (DATETIME)
- published_at (DATETIME)
```

**Output (full schema):**
Returns the complete schema string as defined in database-schema/S4.

## [S3] sql_analyze Tool

Analyze data distribution for a table.

**Input:**
```json
{"table": "articles"}
```

**Output:**
```
Table: articles
- Total rows: 3
- Columns:
  - status: 2 published, 1 draft, 0 archived
  - author_id: 3 distinct values (1, 2, 3)
  - view_count: min=0, max=0, avg=0
  - created_at: range 2026-06-12 to 2026-06-13
  - published_at: 2 non-null, 1 null
```

**Analysis includes:**
- Row count
- Null count per column
- Distinct value count per column
- For numeric columns: min, max, avg
- For text columns: min/max length
- For datetime columns: min/max date

## [S4] sql_insert Tool

Insert a new record.

**Input:**
```json
{
  "table": "articles",
  "values": {
    "author_id": 1,
    "title": "New Article",
    "slug": "new-article",
    "content": "Article content...",
    "status": "draft"
  }
}
```

**Output:**
```
Inserted into articles: id=4
```

**Safety:**
- Requires `--allow-writes` flag
- Validates column names against schema (prevents injection)
- Validates value types against column types
- Returns inserted row ID

## [S5] sql_update Tool

Update existing records.

**Input:**
```json
{
  "table": "articles",
  "values": {"status": "published", "published_at": "2026-06-13"},
  "where": "id = ?",
  "args": [1]
}
```

**Output:**
```
Updated 1 row in articles
```

**Safety:**
- Requires `--allow-writes` flag
- WHERE clause is mandatory (no mass updates)
- Validates column names and types

## [S6] sql_delete Tool

Delete records.

**Input:**
```json
{
  "table": "comments",
  "where": "id = ?",
  "args": [5]
}
```

**Output:**
```
Deleted 1 row from comments
```

**Safety:**
- Requires `--allow-writes` flag AND `--confirm-delete` flag
- WHERE clause is mandatory (no mass deletes)
- Logs the deletion for audit
