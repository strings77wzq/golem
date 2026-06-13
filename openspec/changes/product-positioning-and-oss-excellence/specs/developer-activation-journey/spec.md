## ADDED Requirements

### Requirement: Primary journey is sequential
The project SHALL define the primary developer journey as run locally, extend with tools/MCP/RAG, expose as gateway, and deploy anywhere.

#### Scenario: User reaches next-step guidance
- **WHEN** a user completes the initial local run
- **THEN** the docs route them to extension, gateway, or deployment paths in journey order instead of presenting all modules as equal starting points

### Requirement: Local first run is the activation anchor
The project SHALL make local agent execution the default activation anchor before advanced RAG, MCP, gateway, or deployment flows.

#### Scenario: User starts from quickstart
- **WHEN** a user follows the primary quickstart
- **THEN** they can complete a local agent interaction before selecting an advanced path

### Requirement: Advanced paths preserve continuity
The project SHALL present extension and deployment paths as continuations of the same agent runtime rather than unrelated features.

#### Scenario: User moves from CLI to gateway
- **WHEN** a user reads gateway or deployment guidance
- **THEN** the guidance explains how the gateway path relates to the same agent runtime introduced in the local path

### Requirement: Examples map to journey stages
The project SHALL organize examples and tutorials by user journey stage.

#### Scenario: User chooses an example
- **WHEN** a user browses examples or tutorials
- **THEN** each item identifies whether it supports run, extend, expose, deploy, or contribute stages
