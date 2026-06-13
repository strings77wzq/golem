## ADDED Requirements

### Requirement: Gateway responses SHALL include security headers
The system SHALL add security headers to all HTTP responses from the gateway server.

#### Scenario: Response includes security headers
- **WHEN** any HTTP response is sent from the gateway
- **THEN** the response includes X-Content-Type-Options: nosniff, X-Frame-Options: DENY, and Referrer-Policy: strict-origin-when-cross-origin

#### Scenario: Response includes CSP header
- **WHEN** any HTTP response is sent from the gateway
- **THEN** the response includes Content-Security-Policy: default-src 'self'

### Requirement: Gateway SHALL limit request body size
The system SHALL reject requests that exceed a configurable maximum body size.

#### Scenario: Request body within limit
- **WHEN** a request is sent with a body size under the configured limit
- **THEN** the request is processed normally

#### Scenario: Request body exceeds limit
- **WHEN** a request is sent with a body size exceeding the configured limit
- **THEN** the gateway returns HTTP 413 Payload Too Large

### Requirement: Security headers and body limits SHALL be configurable
The system SHALL allow operators to configure which security headers are enabled and what the request body size limit is.

#### Scenario: Security headers are enabled by default
- **WHEN** the gateway starts without explicit security header configuration
- **THEN** all security headers are enabled with sensible defaults

#### Scenario: Individual headers can be disabled
- **WHEN** an operator disables a specific security header in configuration
- **THEN** that header is omitted from responses
