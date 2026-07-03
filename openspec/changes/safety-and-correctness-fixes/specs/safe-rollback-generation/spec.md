# safe-rollback-generation

## ADDED Requirements

### Requirement: Rollback SQL generators MUST produce syntactically valid SQL or nothing

`GenerateDeleteRollback` and `GenerateUpdateRollback` MUST return `(string, error)`. They MUST refuse to emit malformed SQL: identifiers MUST match the strict identifier charset `[A-Za-z_][A-Za-z0-9_]*` and be quoted; string values MUST have single quotes escaped (doubled); an UPDATE rollback with no `oldValues` MUST return an error rather than producing `UPDATE … SET  WHERE …`. The generators MUST validate inputs themselves and MUST NOT rely on the caller having pre-validated them.

#### Scenario: DELETE rollback for a valid table and WHERE clause

- **WHEN** `GenerateDeleteRollback("products", "id = ?", []interface{}{5})` is called with a charset-valid table name
- **THEN** it returns a non-error rollback string of the form `INSERT INTO "products" … SELECT … FROM "products" WHERE …` with the identifier quoted and `args` preserved as positional placeholders for later binding

#### Scenario: DELETE rollback with an injection payload in the table name

- **WHEN** `GenerateDeleteRollback("users; DROP TABLE users--", "1=1", nil)` is called
- **THEN** it returns `("", error)` and does NOT produce a string containing the payload

#### Scenario: UPDATE rollback is called with real old values

- **WHEN** `GenerateUpdateRollback("products", "id = ?", map[string]interface{}{"price": 9.99, "name": "Bob's pen"})` is called
- **THEN** it returns a non-error rollback string in which the embedded single quote in `Bob's pen` is escaped (`''`) so the produced SQL remains syntactically valid

#### Scenario: UPDATE rollback is called with no old values

- **WHEN** `GenerateUpdateRollback("products", "id = ?", nil)` (or an empty map) is called
- **THEN** it returns `("", error)` with a message indicating old values are required, and MUST NOT emit `UPDATE "products" SET  WHERE …`

### Requirement: Rollback generator signatures MUST bind parameters rather than inline values

The rollback generators MUST NOT inline raw values via `fmt.Sprintf` `%v`. Scalar values MUST be represented either as positional placeholders (for DELETE rollback, matching the supplied `args`) or as escaped literals quoted by the generator itself (for UPDATE rollback, from `oldValues`). The legacy `setClause` parameter is REMOVED from `GenerateUpdateRollback` because it was redundant with `oldValues` and the source of the empty-SET bug.

#### Scenario: DELETE rollback preserves positional placeholders for args

- **WHEN** `GenerateDeleteRollback("t", "id IN (?,?)", []interface{}{1, 2})` is called
- **THEN** the returned SQL keeps `?` placeholders for the WHERE clause (no inlined raw values), and the `args` are returned alongside for the caller to bind when executing the rollback

#### Scenario: sql_query call sites use the new signatures

- **WHEN** `core/tools/database/sql_query.go` runs a DELETE or UPDATE
- **THEN** it calls `GenerateDeleteRollback` / `GenerateUpdateRollback` with the new signatures and handles the returned error; on error it records a safe placeholder (e.g. `"-- rollback unavailable: <reason>"`) in `AuditEntry.RollbackSQL` rather than storing malformed SQL