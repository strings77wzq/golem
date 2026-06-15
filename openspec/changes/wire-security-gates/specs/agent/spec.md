## ADDED Requirements

### Requirement: Agent production metrics reflect runtime behavior
The Agent SHALL update production metrics when message processing, LLM calls, tool calls, sessions, context usage, and security-gate decisions occur.

#### Scenario: Agent message increments message metric
- **WHEN** the Agent processes a user message
- **THEN** the configured agent message or request counter is incremented

#### Scenario: LLM call metrics are updated
- **WHEN** the Agent calls an LLM provider and receives a response or error
- **THEN** LLM call count, latency, error, and token metrics are updated when those metrics are registered

#### Scenario: Tool call metrics are updated
- **WHEN** the Agent executes a tool during the ReAct loop
- **THEN** tool call count, latency, and error metrics are updated when those metrics are registered

#### Scenario: Context metrics are updated
- **WHEN** the Agent builds or compresses context for an LLM call
- **THEN** context token and compression metrics are updated when those metrics are registered

#### Scenario: Session metrics are updated
- **WHEN** a session is created, loaded, activated, or completed through an agent-visible runtime path
- **THEN** active session and total session metrics are updated when those metrics are registered

### Requirement: Agent security-gate metrics reflect allowed and denied decisions
The Agent SHALL update security-gate metrics when tool execution is allowed or denied by SQL safety gates, exec sandbox rules, or gateway security checks that are visible to the agent runtime.

#### Scenario: SQL gate allowed metric
- **WHEN** a SQL statement passes configured safety gates and executes through an agent tool call
- **THEN** the security gate allowed counter is incremented when that metric is registered

#### Scenario: SQL gate denied metric
- **WHEN** a SQL statement is denied by permission or quality gates during an agent tool call
- **THEN** the security gate denied counter is incremented when that metric is registered

#### Scenario: Exec gate denied metric
- **WHEN** an exec command is denied by sandbox rules during an agent tool call
- **THEN** the security gate denied counter is incremented when that metric is registered

#### Scenario: Metrics do not change on validation-only setup
- **WHEN** a tool is constructed or registered but no runtime decision has occurred
- **THEN** security-gate decision counters are not incremented

### Requirement: Agent errors remain safe for users
The Agent SHALL return security and production errors without exposing secrets or raw internal details.

#### Scenario: Security denial is user-safe
- **WHEN** a tool call is denied by a security gate
- **THEN** the Agent response or tool result explains that the operation was blocked without exposing authorization tokens, provider API keys, or raw internal stack traces

#### Scenario: Provider health error is user-safe
- **WHEN** provider health or provider call failures are surfaced to the Agent
- **THEN** the Agent response includes a safe reason and omits secrets from upstream provider errors
