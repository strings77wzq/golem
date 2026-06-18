> Status: active — last reviewed 2026-06-18

## Why

Golem already has substantial engineering depth—ReAct agent loop, tools, providers, MCP, RAG, memory, skills, CLI/TUI/gateway surfaces, and deployment assets—but its public positioning still reads more like a broad feature inventory than a focused product promise. This change establishes a unified product positioning and open-source excellence standard so new users understand who Golem is for, why it differs from Python-first agent frameworks, and how to move from first local run to real usage.

## What Changes

- Establish Golem's north star: make Golem the default starting point for Go developers building local-first AI agents.
- Define the product positioning: a Go-native, local-first AI agent runtime with a path from terminal prototype to deployable gateway.
- Clarify primary and secondary audiences:
  - Primary: Go backend, CLI, and infrastructure developers who want agent capabilities without adopting a Python-first stack.
  - Secondary: developers learning how agent runtimes are built.
- Reframe the external user journey around a single path: run locally → extend with tools/MCP/RAG → expose as gateway → deploy anywhere.
- Define a messaging architecture for README, docs, website, release notes, and community surfaces.
- Establish claim discipline: strongly market only capabilities that are wired, runnable, tested, and documented; describe incomplete production features as a path, planned work, or in-progress hardening.
- Define an OSS excellence standard covering activation, trust, usability, proof assets, and contribution readiness.

## Capabilities

### New Capabilities

- `product-positioning`: Defines Golem's product identity, target users, non-goals, message hierarchy, tone, and strategic differentiation.
- `developer-activation-journey`: Defines the primary user journey from first local run through extension, gateway exposure, and deployment.
- `documentation-information-architecture`: Defines how README, docs, examples, website, release notes, and bilingual content should be organized around the product journey.
- `oss-trust-and-governance`: Defines repository-level trust, support, governance, release, security, and contribution expectations for an excellent open-source project.
- `capability-claim-alignment`: Defines how public claims about capabilities must align with implemented, runnable, tested, and documented behavior.

### Modified Capabilities

- `agent`: Aligns the existing agent capability's external presentation with the product positioning and requires agent-related claims to distinguish stable, runnable behavior from production-path or in-progress capabilities.

## Impact

- Affects product-facing documentation: `README.md`, `docs/`, `CHANGELOG.md`, release notes, website/source content if introduced, and future public messaging.
- Affects repository community surfaces: `CONTRIBUTING.md`, root-level support/security/governance files, issue/PR templates, and contribution guidance.
- Affects examples and tutorials by requiring them to follow the user journey rather than a module inventory.
- Affects product planning by creating a capability-claim alignment gate before promoting runtime features as stable or production-ready.
- Does not introduce new runtime dependencies and does not change the pure Go, CGO-free, single static binary constraint.
