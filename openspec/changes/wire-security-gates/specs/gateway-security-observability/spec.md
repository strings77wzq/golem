## ADDED Requirements

### Requirement: Gateway security headers
The system SHALL add production security headers to gateway HTTP responses by default.

#### Scenario: Successful response includes security headers
- **WHEN** the gateway returns a successful HTTP response
- **THEN** the response includes `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`, and a configured Content Security Policy

#### Scenario: Error response includes security headers
- **WHEN** the gateway returns an error response from validation, authentication, rate limiting, recovery, or not-found handling
- **THEN** the response still includes the configured security headers

#### Scenario: Operator can disable specific header only by explicit configuration
- **WHEN** an operator explicitly disables a configurable security header
- **THEN** that header is omitted and all other default security headers remain enabled

### Requirement: Gateway request body size limit
The system SHALL reject gateway requests whose body exceeds the configured maximum size before handlers read the full body into memory.

#### Scenario: Oversized chat request is rejected
- **WHEN** a `POST /api/chat` request body exceeds the configured maximum body size
- **THEN** the gateway returns HTTP 413 Payload Too Large and does not invoke the agent handler

#### Scenario: Oversized streaming request is rejected
- **WHEN** a `POST /api/chat/stream` request body exceeds the configured maximum body size
- **THEN** the gateway returns HTTP 413 Payload Too Large and does not invoke the streaming or non-streaming agent handler

#### Scenario: Valid request under limit proceeds
- **WHEN** a gateway request body is below the configured maximum body size
- **THEN** the request proceeds to normal authentication, rate limiting, and route handling

### Requirement: Gateway authentication audit events
The system SHALL emit structured audit events for authentication failures when gateway authentication is enabled.

#### Scenario: Missing token is audited
- **WHEN** a protected gateway request omits the required authentication token
- **THEN** the gateway rejects the request and emits an `auth_failure` audit event with method, path, client IP, and severity

#### Scenario: Invalid token is audited
- **WHEN** a protected gateway request provides an invalid authentication token
- **THEN** the gateway rejects the request and emits an `auth_failure` audit event with method, path, client IP, and severity

#### Scenario: Audit event redacts token
- **WHEN** an authentication audit event is emitted
- **THEN** the event does not include the raw authorization header or token value

### Requirement: Gateway rate-limit and IP-denial audit events
The system SHALL emit structured audit events when gateway requests are rejected by rate limiting or IP allowlist checks.

#### Scenario: Rate limit hit is audited
- **WHEN** a gateway request is rejected with HTTP 429 due to rate limiting
- **THEN** the gateway emits a `rate_limit_hit` audit event with method, path, client IP, and severity

#### Scenario: IP block is audited
- **WHEN** a gateway request is rejected because the client IP is not allowed
- **THEN** the gateway emits an `ip_blocked` audit event with method, path, client IP, and severity

#### Scenario: Audit event excludes request body
- **WHEN** a gateway denial audit event is emitted
- **THEN** the event does not include the raw request body

### Requirement: Gateway Prometheus metrics
The system SHALL expose Prometheus metrics at `/metrics` using metric names registered by the codebase.

#### Scenario: Metrics endpoint returns Prometheus text format
- **WHEN** a GET request is made to `/metrics`
- **THEN** the gateway returns HTTP 200 with `text/plain; version=0.0.4` compatible content

#### Scenario: HTTP request counter is updated
- **WHEN** the gateway processes an HTTP request
- **THEN** the HTTP request counter registered in the metrics registry is incremented with method and status labels when supported

#### Scenario: Documentation names match registered metrics
- **WHEN** monitoring documentation references a metric name
- **THEN** that metric name is registered by code or explicitly marked as planned and not available

### Requirement: Middleware order preserves safety and observability
The system SHALL order gateway middleware so safety checks, security headers, recovery, audit, and metrics apply consistently.

#### Scenario: Panic recovery is observable
- **WHEN** a handler panics
- **THEN** recovery returns a safe HTTP error response, security headers are present, and metrics record the request outcome

#### Scenario: Body limit runs before handler body read
- **WHEN** a request body exceeds the configured maximum size
- **THEN** the route handler does not read the entire body into memory

#### Scenario: Security headers apply after denial
- **WHEN** authentication or rate limiting rejects a request
- **THEN** the rejection response includes security headers
