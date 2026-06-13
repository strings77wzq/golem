## ADDED Requirements

### Requirement: Gateway SHALL expose a /metrics endpoint
The system SHALL register a /metrics endpoint on the gateway server that returns metrics in Prometheus exposition format.

#### Scenario: /metrics endpoint is accessible
- **WHEN** a GET request is made to /metrics
- **THEN** the gateway returns HTTP 200 with metrics in Prometheus text format

### Requirement: HTTP request metrics SHALL be tracked
The system SHALL track HTTP request count, duration, and active connections for the gateway.

#### Scenario: HTTP request is counted
- **WHEN** an HTTP request is processed by the gateway
- **THEN** the http_requests_total counter is incremented with method and status labels

#### Scenario: HTTP request duration is recorded
- **WHEN** an HTTP request completes
- **THEN** the http_request_duration_seconds histogram is updated with the request duration

### Requirement: Agent metrics SHALL be tracked
The system SHALL track agent request count, error count, and latency.

#### Scenario: Agent request is counted
- **WHEN** the agent processes a user message
- **THEN** the golem_agent_requests_total counter is incremented

#### Scenario: Agent error is counted
- **WHEN** the agent encounters an error processing a message
- **THEN** the golem_agent_errors_total counter is incremented with error_type label

### Requirement: Token and cost metrics SHALL be tracked
The system SHALL track prompt tokens, completion tokens, and estimated cost for LLM calls.

#### Scenario: Token usage is recorded
- **WHEN** an LLM call completes
- **THEN** golem_tokens_prompt_total and golem_tokens_completion_total counters are incremented by the reported token counts

### Requirement: Provider metrics SHALL be tracked
The system SHALL track provider request count, error count, and latency.

#### Scenario: Provider request is counted
- **WHEN** a provider is called
- **THEN** the golem_provider_requests_total counter is incremented with provider label

#### Scenario: Provider error is counted
- **WHEN** a provider returns an error
- **THEN** the golem_provider_errors_total counter is incremented with provider and error_type labels

### Requirement: Session metrics SHALL be tracked
The system SHALL track active sessions and total sessions created.

#### Scenario: Active session is tracked
- **WHEN** a session becomes active
- **THEN** the golem_sessions_active gauge is incremented

#### Scenario: Session is created
- **WHEN** a new session is created
- **THEN** the golem_sessions_total counter is incremented
