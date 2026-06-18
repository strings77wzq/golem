## ADDED Requirements

### Requirement: Provider health check interface
The system SHALL provide a `HealthChecker` interface that all LLM providers can implement to report their health status.

#### Scenario: Health check returns healthy status
- **WHEN** a provider is operational and responding
- **THEN** the health check SHALL return status "healthy" with latency measurement

#### Scenario: Health check returns degraded status
- **WHEN** a provider is responding but with elevated latency (>2s)
- **THEN** the health check SHALL return status "degraded" with latency measurement

#### Scenario: Health check returns unhealthy status
- **WHEN** a provider fails to respond or returns an error
- **THEN** the health check SHALL return status "unhealthy" with error details

### Requirement: Health check implementation for all providers
Each LLM provider (OpenAI, Anthropic, DeepSeek, Kimi, GLM, MiniMax, Qwen) SHALL implement the `HealthChecker` interface.

#### Scenario: OpenAI provider health check
- **WHEN** health check is called for OpenAI provider
- **THEN** the system SHALL send a minimal chat request and measure response time

#### Scenario: Provider without API key
- **WHEN** health check is called for a provider without configured API key
- **THEN** the health check SHALL return "unhealthy" with error "API key not configured"

### Requirement: Health check scheduling
The system SHALL support configurable health check scheduling with a default interval of 5 minutes.

#### Scenario: Periodic health check
- **WHEN** health check is enabled
- **THEN** the system SHALL check all providers at the configured interval

#### Scenario: Manual health check trigger
- **WHEN** user requests health check via CLI or API
- **THEN** the system SHALL immediately check all providers and return results

### Requirement: Health status caching
The system SHALL cache health check results to avoid excessive API calls.

#### Scenario: Cache hit within TTL
- **WHEN** health status is requested within cache TTL (default 30s)
- **THEN** the system SHALL return cached status without making API call

#### Scenario: Cache miss or expired
- **WHEN** health status is requested after cache TTL
- **THEN** the system SHALL perform fresh health check and update cache

### Requirement: Health check API endpoint
The Gateway SHALL expose a `/health/providers` endpoint returning health status of all configured providers.

#### Scenario: GET /health/providers returns all statuses
- **WHEN** client requests GET /health/providers
- **THEN** the system SHALL return JSON with health status of all providers

#### Scenario: Health endpoint requires authentication
- **WHEN** client requests GET /health/providers without valid auth token
- **THEN** the system SHALL return 401 Unauthorized
