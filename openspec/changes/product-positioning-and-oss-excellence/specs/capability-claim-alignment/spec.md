## ADDED Requirements

### Requirement: Public claims require evidence
Public claims about runtime, production, security, deployment, provider, or integration capabilities SHALL be backed by runnable behavior, tests, examples, documentation, or explicit limitations.

#### Scenario: Documentation promotes a capability
- **WHEN** README, website, release notes, or docs promote a capability as available
- **THEN** the claim is backed by at least one runnable command, test, example, reference guide, or documented limitation

### Requirement: Capability maturity is classified
The project SHALL classify capability claims as stable, beta, in-progress, planned, or production-path when full readiness is not established.

#### Scenario: Capability is partially implemented
- **WHEN** a capability exists in code but is not wired into the primary runtime path or lacks verification
- **THEN** public messaging classifies it as in-progress, planned, or production-path rather than stable

### Requirement: Production claims are gated
The project SHALL reserve production-ready language for capabilities that are wired, tested, documented, and deployable end to end.

#### Scenario: Production surface is described
- **WHEN** a public surface describes gateway, security, metrics, health, Docker, Kubernetes, or Helm behavior
- **THEN** it distinguishes stable production-ready behavior from production-path or hardening work

### Requirement: Claim review is part of change review
Product-facing changes SHALL include a capability-claim review step.

#### Scenario: PR updates public messaging
- **WHEN** a pull request updates README, docs, website content, release notes, examples, or public-facing claims
- **THEN** reviewers check whether promoted capabilities match implementation and verification status
