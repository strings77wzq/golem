## Context

Golem is already a working Go AI agent runtime and learning-oriented codebase with ReAct agent behavior, tools, providers, MCP, RAG, memory, skills, CLI/TUI/gateway surfaces, and deployment assets. The project also has extensive documentation and a completed `phase4-open-source-growth` planning change, but the public story remains broader than the product promise: users can see many capabilities before they understand the primary reason to choose Golem.

This change acts as a strategic alignment layer above existing growth, roadmap, and production-readiness work. It does not replace implementation changes such as `phase3-production-readiness`; instead, it defines how Golem should present itself and how future work should decide whether a capability is ready to be promoted publicly.

## Goals / Non-Goals

**Goals:**

- Make the agreed north star explicit: Golem should become the default starting point for Go developers building local-first AI agents.
- Align README, docs, examples, website content, releases, and community surfaces around one positioning: Go-native, local-first AI agent runtime with a path from terminal prototype to deployable gateway.
- Define the primary audience as Go backend, CLI, and infrastructure developers who want agent capabilities without adopting a Python-first stack.
- Preserve the secondary educational value for developers learning how agent runtimes are built.
- Replace feature-list-first messaging with a product journey: run locally → extend with tools/MCP/RAG → expose as gateway → deploy anywhere.
- Establish a capability-claim discipline so public claims only become strong claims when they are wired, runnable, tested, and documented.

**Non-Goals:**

- Do not introduce new core agent runtime capabilities in this change.
- Do not reimplement Phase 4 growth assets or duplicate all existing docs planning.
- Do not replace Phase 3 production-readiness implementation work.
- Do not position Golem as a no-code agent platform, hosted SaaS, or Python agent framework clone.
- Do not claim full production-ready status before production-path capabilities are fully wired and verified.

## Decisions

### D1. Use “Go-native Agent Runtime” as the primary identity

Golem SHALL present its main identity as a Go-native agent runtime rather than only a CLI, learning project, or production gateway.

- **Why:** This identity can contain the CLI/TUI entry point, extension systems, gateway deployment path, and educational value without letting any one surface dominate the project.
- **Alternative considered:** Lead with “AI Agent Framework.” Rejected because it is generic and does not explain why users should choose Golem over Python-first frameworks.
- **Alternative considered:** Lead with “AI Agent CLI.” Rejected because it underplays the framework, gateway, and deployment path.

### D2. Use local-first as the entry experience

The first product journey SHALL start with local use: install, initialize, run a local agent, then choose advanced paths.

- **Why:** Local-first maps to the project’s single-binary, pure Go, SQLite/session, CLI/TUI, and Termux strengths.
- **Alternative considered:** Lead with gateway deployment. Rejected because it raises initial complexity and makes production hardening gaps more visible before the user has experienced value.

### D3. Say “production path” until production readiness is fully closed

Public language SHALL prefer “path from terminal prototype to deployable gateway” or “production path” until production capabilities are wired, tested, and documented end to end.

- **Why:** This keeps messaging ambitious but honest. Current production-adjacent packages and docs exist, but some capabilities are still integration or hardening work.
- **Alternative considered:** Claim “production-ready” now. Rejected because claim accuracy is more valuable for open-source trust than stronger marketing language.

### D4. Organize external content around journey stages, not module inventory

README, docs, tutorials, examples, and website content SHALL orient around user progress: run locally, extend, expose, deploy, contribute.

- **Why:** Module-first organization is useful for maintainers, but new users need an activation path before they need package boundaries.
- **Alternative considered:** Preserve a feature-list-first homepage. Rejected because it increases cognitive load and dilutes the product promise.

### D5. Treat claim alignment as a release and docs gate

Any public claim about runtime, production, security, deployment, or provider behavior SHALL have evidence: a runnable command, test, example, reference doc, or documented limitation.

- **Why:** Excellent OSS projects build trust by being accurate about maturity.
- **Alternative considered:** Promote roadmap capabilities early to generate interest. Rejected because broken expectations damage credibility.

### D6. Use English-first with full Chinese support for global OSS growth

Primary global surfaces SHOULD be English-first, while Chinese documentation and study guides remain first-class supporting material.

- **Why:** GitHub discovery and global adoption require English-first first screens. Chinese material remains a differentiator and community strength.
- **Alternative considered:** Keep bilingual-first in every primary surface. Rejected because mixed first screens can make the core message harder for both audiences to parse.

## Risks / Trade-offs

- **Risk: Strategic docs do not produce visible user value quickly.** → Mitigation: tasks must connect positioning work to README, docs, examples, and review gates.
- **Risk: Positioning overlaps Phase 4 growth planning.** → Mitigation: this change defines the north star, message system, and claim discipline; Phase 4 remains the asset planning baseline.
- **Risk: Production language becomes too cautious.** → Mitigation: use “production path” and “deployable gateway” to preserve ambition while avoiding overclaiming.
- **Risk: English-first could reduce Chinese community resonance.** → Mitigation: keep Chinese study content and zh-CN docs as a supported content path rather than removing them.
- **Risk: Claim gates slow down releases.** → Mitigation: classify claims as stable, beta, planned, or in-progress instead of blocking all mentions.

## Migration Plan

1. Capture this positioning in proposal, design, specs, and tasks.
2. Use CEO/product review to confirm the product identity, audience, and claim discipline before implementation.
3. During implementation, update public-facing docs and examples first, then repository trust/governance surfaces, then claim-alignment checks.
4. Keep production-hardening tasks in Phase 3 or related technical changes, but make messaging accurate until those tasks land.
5. Revisit the positioning after production-readiness work is complete and decide whether “production path” can be upgraded to “production-ready.”

## Success Criteria

- A new visitor can understand from the README top path that Golem is a Go-native, local-first AI agent runtime with a path from terminal prototype to deployable gateway.
- The README top path is English-first across hero, value props, install, quickstart/first-success path, docs entry, and zh-CN/study links.
- Every strong public claim about runtime, production, security, monitoring, routing, deployment, or provider behavior has a maturity label and evidence link or explicit limitation.
- The quickstart either provides a tested zero-key mock/dry-run path or clearly states that a real provider API key is required and records mock mode as a follow-up.
- Support, security, governance/contribution, and release-maturity paths are discoverable from repository-level or GitHub-recognized locations.

## Risk Register

| Risk | Impact | Mitigation | Triage |
|------|--------|------------|--------|
| Strategy artifacts ship without changing first-minute UX | High | Put README top-path correction and quickstart work before internal-only artifacts | Do not mark complete until README/docs entry changed |
| Mock-first path is not feasible within architecture boundaries | Medium | Run feasibility gate before implementation; require tests if user-facing | Cut mock implementation before cutting claim discipline |
| English-first migration weakens Chinese community value | Medium | Keep clear zh-CN/study links and preserve Chinese study content as first-class learning material | Move translations/supporting content, do not delete useful Chinese docs |
| Capability classifications become disputed | Medium | Use rubric plus evidence links; maintainer decision is fallback | Prefer lower maturity label when evidence is ambiguous |

## Open Questions

- Should capability claim alignment stay manual through review checklists first, or later become a lightweight docs lint/test process?
- Should the first example set be created under this change or left to a follow-up implementation change focused on examples?
