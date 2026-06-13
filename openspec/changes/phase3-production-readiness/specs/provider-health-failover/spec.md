## ADDED Requirements

### Requirement: Provider health SHALL be monitored periodically
The system SHALL run health checks on all registered providers at a configurable interval and store health status.

#### Scenario: Health check runs on startup
- **WHEN** the agent starts with provider health monitoring enabled
- **THEN** health checks are performed for all registered providers and status is recorded

#### Scenario: Health check runs periodically
- **WHEN** a configured interval elapses after the last health check
- **THEN** health checks are re-run and status is updated

### Requirement: Provider health status SHALL be queryable
The system SHALL expose provider health status so operators can monitor provider availability.

#### Scenario: Health status includes all providers
- **WHEN** health status is queried
- **THEN** the response includes status (healthy/degraded/unhealthy), latency, last check time, and error message for each provider

### Requirement: Automatic provider failover SHALL work when a provider is unhealthy
The system SHALL automatically route requests to a fallback provider when the primary provider is unhealthy.

#### Scenario: Primary provider fails health check
- **WHEN** a provider's health status is unhealthy
- **THEN** requests for that provider are routed to the next provider in the fallback chain

#### Scenario: All providers in chain are unhealthy
- **WHEN** all providers in a fallback chain are unhealthy
- **THEN** an error is returned indicating no healthy providers available

#### Scenario: Provider recovers
- **WHEN** a previously unhealthy provider passes a health check
- **THEN** it is automatically re-added to the active provider pool
