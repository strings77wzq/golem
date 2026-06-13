## ADDED Requirements

### Requirement: Agent presentation aligns with product positioning
The Agent capability SHALL be presented externally as part of Golem's Go-native, local-first agent runtime rather than as an isolated chatbot or feature list.

#### Scenario: Agent capability is described publicly
- **WHEN** README, docs, examples, website content, or release notes describe the Agent capability
- **THEN** they connect it to the local-first runtime journey and avoid presenting it only as a standalone chat surface

### Requirement: Agent capability claims distinguish maturity
Agent-related public claims SHALL distinguish stable local runtime behavior from production-path or in-progress behavior.

#### Scenario: Agent capability includes deployment claims
- **WHEN** public documentation describes agent behavior through CLI, TUI, gateway, Telegram, tools, MCP, RAG, memory, skills, or provider integrations
- **THEN** it accurately labels the maturity of each surface or integration according to whether it is wired, runnable, tested, and documented
