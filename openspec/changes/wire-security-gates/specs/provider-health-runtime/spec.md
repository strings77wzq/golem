## ADDED Requirements

### Requirement: Provider health endpoint reports configured health status
The system SHALL expose provider health status through the gateway when a provider health checker is configured.

#### Scenario: Configured checker returns provider statuses
- **WHEN** the gateway has a configured provider health checker and a GET request is made to `/health/providers`
- **THEN** the response contains the health status map returned by the checker

#### Scenario: Status includes operator-relevant fields
- **WHEN** provider health status is returned
- **THEN** each provider status includes availability state, last check time, latency when known, and a sanitized error message when unhealthy

#### Scenario: Secrets are not exposed in health response
- **WHEN** a provider health check fails because of credentials or upstream errors
- **THEN** the health response does not include API keys, authorization headers, or raw upstream request payloads

### Requirement: Not-configured provider health behavior is explicit
The system SHALL return an explicit not-configured response when no provider health checker is attached to the gateway.

#### Scenario: No health checker attached
- **WHEN** the gateway has no provider health checker and a GET request is made to `/health/providers`
- **THEN** the response returns HTTP 200 with status `not_configured` and a clear explanatory message

#### Scenario: Not-configured health is documented
- **WHEN** documentation describes `/health/providers`
- **THEN** it explains both configured and not-configured behavior

### Requirement: Provider health lifecycle is context-controlled
The system SHALL start and stop provider health checks using the gateway or command context lifecycle.

#### Scenario: Health manager starts with gateway runtime
- **WHEN** the gateway command starts and provider health checking is configured
- **THEN** the health manager starts health checks using a context controlled by the command lifecycle

#### Scenario: Health manager stops on shutdown
- **WHEN** the gateway shuts down
- **THEN** provider health check goroutines stop without leaking goroutines

#### Scenario: Missing provider credentials do not crash gateway startup
- **WHEN** provider health checking is enabled but a provider lacks credentials
- **THEN** gateway startup succeeds and the provider health status reflects not-configured or unhealthy state without leaking secrets

### Requirement: Provider failover is explicitly out of runtime scope unless already supported
The system SHALL NOT claim automatic provider failover unless runtime request routing actually uses health status to choose providers.

#### Scenario: Health visibility without failover
- **WHEN** provider health is exposed but request routing still uses the configured provider directly
- **THEN** documentation describes health visibility only and does not claim automatic failover

#### Scenario: Failover claim requires routing test
- **WHEN** documentation claims automatic provider failover
- **THEN** there is an integration test proving requests are routed away from an unhealthy primary provider to a healthy fallback provider
