## ADDED Requirements

### Requirement: Agent integrates provider health check
The Agent SHALL use provider health status to inform routing decisions.

#### Scenario: Skip unhealthy provider
- **WHEN** a provider is marked unhealthy
- **THEN** the Agent SHALL skip that provider in the routing chain

#### Scenario: Prefer healthy provider with lower latency
- **WHEN** multiple providers are healthy
- **THEN** the Agent SHALL prefer the provider with lower latency

### Requirement: Agent supports session export
The Agent SHALL support exporting the current session context.

#### Scenario: Export current session
- **WHEN** user requests session export
- **THEN** the Agent SHALL return session data in JSON format

#### Scenario: Export includes tool call history
- **WHEN** session with tool calls is exported
- **THEN** the export SHALL include tool call requests and results

### Requirement: Agent integrates memory module
The Agent SHALL use the memory module for long-term context retention.

#### Scenario: Store important interaction in memory
- **WHEN** an interaction is marked as important
- **THEN** the Agent SHALL store it in the memory module

#### Scenario: Retrieve relevant memories for context
- **WHEN** processing a user message
- **THEN** the Agent SHALL retrieve relevant memories and include in context
