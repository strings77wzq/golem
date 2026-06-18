## ADDED Requirements

### Requirement: Audit log recording
The Gateway SHALL record audit logs for all HTTP requests.

#### Scenario: Record successful request
- **WHEN** a request is processed successfully
- **THEN** the system SHALL record method, path, status code, latency, client IP, and timestamp

#### Scenario: Record failed request
- **WHEN** a request fails with error
- **THEN** the system SHALL record error details in the audit log

#### Scenario: Record authenticated request
- **WHEN** a request includes valid authentication
- **THEN** the system SHALL record user ID in the audit log

### Requirement: Audit log storage
The system SHALL store audit logs in SQLite with structured schema.

#### Scenario: Initialize audit log table
- **WHEN** gateway starts
- **THEN** the system SHALL create audit_logs table if not exists

#### Scenario: Query audit logs
- **WHEN** admin queries audit logs with time range
- **THEN** the system SHALL return logs within the specified range

### Requirement: Audit log retention
The system SHALL support configurable audit log retention with default of 7 days.

#### Scenario: Automatic log cleanup
- **WHEN** audit log age exceeds retention period
- **THEN** the system SHALL delete the log entry

#### Scenario: Configure retention period
- **WHEN** retention is configured to 30 days
- **THEN** the system SHALL keep logs for 30 days before cleanup

### Requirement: Audit log fields
Each audit log entry SHALL contain the following fields: ID, timestamp, method, path, status code, latency, client IP, user ID, request size, response size.

#### Scenario: Complete audit log entry
- **WHEN** a request is logged
- **THEN** the entry SHALL include all required fields

#### Scenario: Anonymous request logging
- **WHEN** a request without authentication is logged
- **THEN** the user ID field SHALL be empty or null

### Requirement: Audit log API endpoint
The Gateway SHALL expose an admin endpoint for querying audit logs.

#### Scenario: GET /admin/audit-logs returns logs
- **WHEN** admin requests GET /admin/audit-logs
- **THEN** the system SHALL return paginated audit logs

#### Scenario: Filter by status code
- **WHEN** admin requests audit logs with filter status=500
- **THEN** the system SHALL return only logs with status code 500

#### Scenario: Filter by time range
- **WHEN** admin requests audit logs with from/to parameters
- **THEN** the system SHALL return logs within the time range

### Requirement: Audit log security
Access to audit logs SHALL require admin authentication.

#### Scenario: Unauthorized audit log access
- **WHEN** unauthenticated user requests audit logs
- **THEN** the system SHALL return 401 Unauthorized

#### Scenario: Non-admin audit log access
- **WHEN** authenticated non-admin user requests audit logs
- **THEN** the system SHALL return 403 Forbidden

### Requirement: Sensitive data handling
The audit log SHALL NOT record sensitive data such as API keys, passwords, or message content.

#### Scenario: Request body not logged
- **WHEN** a POST request with sensitive body is logged
- **THEN** the system SHALL NOT include request body in audit log

#### Scenario: Headers filtered
- **WHEN** a request with Authorization header is logged
- **THEN** the system SHALL NOT include Authorization header value
