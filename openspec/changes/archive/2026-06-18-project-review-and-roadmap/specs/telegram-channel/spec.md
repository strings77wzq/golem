## ADDED Requirements

### Requirement: Telegram adapter builds successfully
The Telegram adapter package SHALL compile without errors.

#### Scenario: Package compiles
- **WHEN** running `go build ./internal/channels/telegram/...`
- **THEN** the build SHALL succeed without errors

#### Scenario: No type redeclarations
- **WHEN** the package is compiled
- **THEN** there SHALL be no duplicate type declarations for Update, User, or Chat

### Requirement: Telegram client has required methods
The Telegram Client SHALL have all methods referenced by webhook functionality.

#### Scenario: Client has doRequest method
- **WHEN** webhook.go calls c.doRequest()
- **THEN** the method SHALL exist on the Client type

#### Scenario: doRequest handles errors
- **WHEN** doRequest encounters an HTTP error
- **THEN** it SHALL return a wrapped error with context

### Requirement: Telegram types are unified
All Telegram API types SHALL be defined in a single location to avoid conflicts.

#### Scenario: Types defined in types.go only
- **WHEN** the package is compiled
- **THEN** Update, User, Chat, and Message types SHALL be defined only in types.go

#### Scenario: Webhook uses shared types
- **WHEN** webhook.go needs Telegram types
- **THEN** it SHALL import from types.go in the same package

### Requirement: Telegram webhook handler validates requests
The webhook handler SHALL validate incoming requests before processing.

#### Scenario: Validate secret token
- **WHEN** a request is received with secret token configured
- **THEN** the handler SHALL validate the X-Telegram-Bot-Api-Secret-Token header

#### Scenario: Reject invalid method
- **WHEN** a non-POST request is received
- **THEN** the handler SHALL return 405 Method Not Allowed

#### Scenario: ACK immediately
- **WHEN** a valid webhook request is received
- **THEN** the handler SHALL return 200 OK before processing

### Requirement: Telegram adapter test coverage
The Telegram adapter SHALL have minimum 70% test coverage.

#### Scenario: Unit tests for client methods
- **WHEN** running tests
- **THEN** GetUpdates, SendMessage, SetWebhook, DeleteWebhook SHALL be tested

#### Scenario: Unit tests for webhook handler
- **WHEN** running tests
- **THEN** WebhookHandler SHALL be tested for valid and invalid requests
