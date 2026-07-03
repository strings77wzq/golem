# sql-identifier-guard

## ADDED Requirements

### Requirement: SQL identifier tools MUST validate table names against the live schema

Database tools that interpolate a table identifier supplied by the LLM (e.g. `sql_analyze`) MUST verify the identifier names a real table in the live schema before interpolating it into SQL. An identifier that is not present in the schema MUST be rejected with a typed, LLM-visible error and MUST NOT be sent to the database driver. Validation MUST use the driver's existing schema contract (`GetSchemaForTable`) as a presence probe; no new `SQLDriver` interface method is introduced.

#### Scenario: LLM supplies a valid existing table name

- **WHEN** the `sql_analyze` tool receives `table = "products"` and `products` is a table in the connected schema
- **THEN** the tool issues `SELECT COUNT(*) … FROM products` and returns row count and schema to the LLM

#### Scenario: LLM supplies a table name not in the schema

- **WHEN** the `sql_analyze` tool receives `table = "nonexistent"` and `nonexistent` is not a table in the connected schema
- **THEN** the tool returns a `ToolResult` with `IsError=true` and a human/LLM-readable error naming the rejected identifier, and issues NO query to the database driver

#### Scenario: LLM supplies an identifier containing an SQL injection payload

- **WHEN** the `sql_analyze` tool receives `table = "users; DROP TABLE users--"` (or any value failing the strict identifier charset `[A-Za-z_][A-Za-z0-9_]*`)
- **THEN** the tool rejects the identifier before any database round-trip and returns a `ToolResult` with `IsError=true`, and the payload MUST NOT be interpolated into or executed as SQL

#### Scenario: Schema metadata lookup itself fails

- **WHEN** the driver's `GetSchemaForTable` returns an error (connection lost, permissions)
- **THEN** the tool returns a `ToolResult` with `IsError=true` carrying the driver error and does not issue the counting query