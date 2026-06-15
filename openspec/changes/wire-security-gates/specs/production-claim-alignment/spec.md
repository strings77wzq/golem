## ADDED Requirements

### Requirement: Public safety claims match runtime behavior
The system SHALL keep public safety claims aligned with wired, tested runtime behavior.

#### Scenario: README claims read-only default
- **WHEN** README states that database operations are read-only by default
- **THEN** tests prove write SQL is denied under the default database tool configuration

#### Scenario: README claims WHERE enforcement
- **WHEN** README states that UPDATE or DELETE requires a WHERE clause
- **THEN** tests prove UPDATE and DELETE without WHERE are denied before SQL driver execution

#### Scenario: README claims rollback SQL
- **WHEN** README states that rollback SQL is generated
- **THEN** documentation describes rollback SQL as best-effort and tests cover both generated and unavailable rollback cases

#### Scenario: README claims audit logging
- **WHEN** README states that operations are audit logged
- **THEN** tests prove allowed and denied security-relevant operations emit audit events when audit is configured

### Requirement: Monitoring documentation matches registered metrics
The system SHALL keep monitoring examples aligned with metrics registered by runtime code.

#### Scenario: PromQL metric exists
- **WHEN** `docs/MONITORING.md` includes a PromQL query using a metric name
- **THEN** that metric name is registered in code and appears in `/metrics` output under the documented runtime condition

#### Scenario: Planned metric is marked planned
- **WHEN** documentation mentions a metric that is not registered in code
- **THEN** the metric is marked as planned or removed from the production-ready monitoring section

#### Scenario: Dashboard files are referenced only when present
- **WHEN** documentation references a Grafana dashboard file
- **THEN** that file exists in the repository or the documentation clearly marks it as planned

### Requirement: Gateway API documentation matches authentication behavior
The system SHALL keep gateway API documentation aligned with actual authentication and security behavior.

#### Scenario: Auth token behavior is documented
- **WHEN** gateway authentication can be enabled through configuration or environment variables
- **THEN** gateway documentation describes how to enable it and how requests should authenticate

#### Scenario: No-auth mode is documented as local/dev default when applicable
- **WHEN** gateway authentication is disabled by default
- **THEN** documentation frames no-auth mode as local or development behavior, not as production guidance

#### Scenario: Security headers and body limits are documented
- **WHEN** gateway security headers and body size limits are enabled by default
- **THEN** gateway documentation lists the defaults and how operators can configure them

### Requirement: Deferred production work is captured explicitly
The system SHALL record production-related work that is discovered but intentionally out of scope.

#### Scenario: Provider failover deferred
- **WHEN** provider health visibility ships without automatic failover
- **THEN** the deferred failover work is captured in `TODOS.md` or a separate OpenSpec change with context and acceptance criteria

#### Scenario: Metric prefix migration deferred
- **WHEN** metric names are aligned to existing code instead of migrated to a new prefix
- **THEN** any desired metric naming migration is captured as deferred work rather than implied as completed

#### Scenario: Rate limiter lifecycle follow-up deferred
- **WHEN** rate limiter goroutine lifecycle is identified but not changed in this slice
- **THEN** the follow-up is captured with risk, affected package, and suggested test strategy
