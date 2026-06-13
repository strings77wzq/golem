## ADDED Requirements

### Requirement: Session export to JSON
The system SHALL allow exporting a session to a JSON file with complete conversation history and metadata.

#### Scenario: Export session via CLI
- **WHEN** user runs `golem session export <session-id> -o session.json`
- **THEN** the system SHALL create a JSON file with session data

#### Scenario: Export session via API
- **WHEN** client requests GET /sessions/{id}/export
- **THEN** the system SHALL return JSON with session data and appropriate headers

#### Scenario: Export non-existent session
- **WHEN** user attempts to export a session that does not exist
- **THEN** the system SHALL return an error "session not found"

### Requirement: Session import from JSON
The system SHALL allow importing a session from a JSON file, creating or restoring the session.

#### Scenario: Import session via CLI
- **WHEN** user runs `golem session import -i session.json`
- **THEN** the system SHALL create a new session with the imported data

#### Scenario: Import session via API
- **WHEN** client POSTs JSON to /sessions/import
- **THEN** the system SHALL create a new session and return the session ID

#### Scenario: Import with invalid JSON
- **WHEN** user attempts to import a malformed JSON file
- **THEN** the system SHALL return an error "invalid session format"

#### Scenario: Import with unsupported version
- **WHEN** user attempts to import a session with unsupported format version
- **THEN** the system SHALL return an error "unsupported session format version"

### Requirement: Export format specification
The export format SHALL include a version field, export timestamp, and complete session data.

#### Scenario: Export format includes required fields
- **WHEN** a session is exported
- **THEN** the JSON SHALL contain "version", "exported_at", and "session" fields

#### Scenario: Export format includes message history
- **WHEN** a session with messages is exported
- **THEN** the JSON SHALL include all messages with role, content, and timestamps

### Requirement: Import conflict handling
The system SHALL handle import conflicts when a session with the same ID already exists.

#### Scenario: Import with overwrite flag
- **WHEN** user imports a session with existing ID and --overwrite flag
- **THEN** the system SHALL replace the existing session

#### Scenario: Import without overwrite flag
- **WHEN** user imports a session with existing ID without --overwrite flag
- **THEN** the system SHALL generate a new session ID and import as new session

### Requirement: Export list command
The system SHALL provide a command to list all sessions available for export.

#### Scenario: List sessions via CLI
- **WHEN** user runs `golem session list`
- **THEN** the system SHALL display all sessions with ID, creation time, and message count
