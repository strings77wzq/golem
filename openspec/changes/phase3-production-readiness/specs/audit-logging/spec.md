## ADDED Requirements

### Requirement: Failed authentication attempts SHALL be logged
The system SHALL log failed authentication attempts as structured audit events.

#### Scenario: Invalid auth token is used
- **WHEN** a request is made with an invalid or missing auth token
- **THEN** an audit log entry is created with event_type: "auth_failure", client_ip, and request_path

### Requirement: Rate limit hits SHALL be logged
The system SHALL log when a client is rate limited as a structured audit event.

#### Scenario: Client is rate limited
- **WHEN** a request is rejected due to rate limiting (HTTP 429)
- **THEN** an audit log entry is created with event_type: "rate_limit_hit", client_ip, and request_path

### Requirement: Blocked IPs SHALL be logged
The system SHALL log when a request is blocked by IP whitelist as a structured audit event.

#### Scenario: IP is blocked by whitelist
- **WHEN** a request is rejected because the client IP is not in the AllowFrom whitelist
- **THEN** an audit log entry is created with event_type: "ip_blocked", client_ip, and request_path

### Requirement: Audit log entries SHALL be structured JSON
The system SHALL format audit log entries as structured JSON with consistent fields.

#### Scenario: Audit entry contains standard fields
- **WHEN** an audit event is logged
- **THEN** the log entry includes timestamp, event_type, client_ip, request_path, and severity fields
