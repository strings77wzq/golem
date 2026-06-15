## ADDED Requirements

### Requirement: Read-only SQL by default
The system SHALL execute database tool SQL in read-only mode by default unless an explicit write or admin permission level is configured by the operator.

#### Scenario: SELECT is allowed by default
- **WHEN** the `sql_query` tool receives a single read-only `SELECT` statement under the default permission level
- **THEN** the statement is passed to the SQL driver's query path and the result is returned

#### Scenario: INSERT is denied by default
- **WHEN** the `sql_query` tool receives an `INSERT` statement under the default permission level
- **THEN** the statement is denied before reaching the SQL driver execute path

#### Scenario: UPDATE is denied by default
- **WHEN** the `sql_query` tool receives an `UPDATE` statement under the default permission level
- **THEN** the statement is denied before reaching the SQL driver execute path

#### Scenario: DELETE is denied by default
- **WHEN** the `sql_query` tool receives a `DELETE` statement under the default permission level
- **THEN** the statement is denied before reaching the SQL driver execute path

#### Scenario: Admin SQL is denied by default
- **WHEN** the `sql_query` tool receives `DROP`, `ALTER`, `TRUNCATE`, or an unknown executable SQL operation under the default permission level
- **THEN** the statement is denied before reaching any SQL driver execution path

### Requirement: Explicit write permission
The system SHALL allow write SQL only when an operator explicitly configures a permission level that permits the requested operation.

#### Scenario: INSERT is allowed with write permission
- **WHEN** the `sql_query` tool receives an `INSERT` statement and the tool is configured with write permission
- **THEN** the statement is passed to the SQL driver's execute path

#### Scenario: UPDATE is allowed with write permission and safety checks
- **WHEN** the `sql_query` tool receives an `UPDATE` statement and the tool is configured with write permission
- **THEN** the system runs all destructive SQL quality gates before passing the statement to the SQL driver's execute path

#### Scenario: DELETE requires delete permission
- **WHEN** the `sql_query` tool receives a `DELETE` statement and the tool is configured only with write permission
- **THEN** the statement is denied before reaching the SQL driver execute path

#### Scenario: Admin SQL requires admin permission
- **WHEN** the `sql_query` tool receives `DROP`, `ALTER`, `TRUNCATE`, or an unknown executable operation without admin permission
- **THEN** the statement is denied before reaching the SQL driver execute path

### Requirement: Destructive SQL WHERE gate
The system SHALL reject `UPDATE` and `DELETE` statements that do not contain a meaningful `WHERE` clause when destructive SQL quality gates are enabled.

#### Scenario: UPDATE without WHERE is blocked
- **WHEN** the `sql_query` tool receives `UPDATE users SET role = 'admin'` with sufficient write permission
- **THEN** the statement is denied before reaching the SQL driver execute path

#### Scenario: DELETE without WHERE is blocked
- **WHEN** the `sql_query` tool receives `DELETE FROM users` with sufficient delete permission
- **THEN** the statement is denied before reaching the SQL driver execute path

#### Scenario: UPDATE with WHERE is checked then executed
- **WHEN** the `sql_query` tool receives `UPDATE users SET role = 'admin' WHERE id = ?` with sufficient write permission
- **THEN** the system passes the quality gate and sends the statement to the SQL driver execute path

#### Scenario: DELETE with WHERE is checked then executed
- **WHEN** the `sql_query` tool receives `DELETE FROM users WHERE id = ?` with sufficient delete permission
- **THEN** the system passes the quality gate and sends the statement to the SQL driver execute path

#### Scenario: Empty WHERE is blocked
- **WHEN** the `sql_query` tool receives a destructive statement whose parsed WHERE clause is empty or missing after normalization
- **THEN** the statement is denied before reaching the SQL driver execute path

### Requirement: Conservative SQL classification
The system SHALL classify SQL conservatively so ambiguous or multi-statement input is denied unless explicitly permitted by admin-level configuration.

#### Scenario: Empty SQL is rejected
- **WHEN** the `sql_query` tool receives an empty or whitespace-only SQL string
- **THEN** the tool returns an explicit error and does not call a SQL driver

#### Scenario: Unknown SQL is denied
- **WHEN** the `sql_query` tool receives SQL whose first executable operation cannot be classified
- **THEN** the statement is treated as admin-level and denied unless admin permission is configured

#### Scenario: Multiple statements are denied by default
- **WHEN** the `sql_query` tool receives multiple SQL statements in one request
- **THEN** the tool denies the request before reaching any SQL driver execution path

#### Scenario: Leading comments do not bypass classification
- **WHEN** the `sql_query` tool receives SQL with leading comments before a destructive operation
- **THEN** the destructive operation is still classified and gated before driver execution

### Requirement: Rollback SQL is best-effort and honest
The system SHALL generate rollback SQL only when it can produce a meaningful rollback statement and SHALL not fabricate unavailable old values.

#### Scenario: DELETE rollback statement is generated when table and WHERE are known
- **WHEN** a permitted `DELETE FROM <table> WHERE <condition>` statement executes successfully
- **THEN** the tool result includes a best-effort rollback statement based on the table and WHERE clause

#### Scenario: UPDATE rollback is not fabricated without old values
- **WHEN** a permitted `UPDATE` statement executes successfully and old row values were not captured before execution
- **THEN** the tool result does not fabricate old values and instead reports that exact rollback SQL is unavailable

#### Scenario: Unsupported rollback shape is explicit
- **WHEN** a permitted write statement executes successfully but rollback generation cannot determine the table or predicate safely
- **THEN** the tool result explicitly states that rollback SQL is unavailable for that operation shape

### Requirement: SQL audit events
The system SHALL emit audit events for security-relevant SQL decisions when an audit sink is configured.

#### Scenario: Allowed write emits audit event
- **WHEN** a permitted write statement executes successfully and an audit sink is configured
- **THEN** the audit sink receives an event with operation, database, table when known, affected rows, status `success`, and rollback information when available

#### Scenario: Denied SQL emits audit event
- **WHEN** a SQL statement is denied by permission or quality gates and an audit sink is configured
- **THEN** the audit sink receives an event with operation, database, reason, and status `denied`

#### Scenario: SQL audit redacts sensitive values
- **WHEN** a SQL audit event is emitted
- **THEN** the event does not include provider API keys, authorization headers, or unredacted parameter values unless explicit verbose audit mode is configured
