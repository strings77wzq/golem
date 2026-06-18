---
name: openspec-proposal-lifecycle
description: Workflow command scaffold for openspec-proposal-lifecycle in golem.
allowed_tools: ["Bash", "Read", "Write", "Grep", "Glob"]
---

# /openspec-proposal-lifecycle

Use this workflow when working on **openspec-proposal-lifecycle** in `golem`.

## Goal

Manages the lifecycle of OpenSpec proposals, including creation, auditing, archiving, and promoting specs to stable. Ensures proposal hygiene and traceability.

## Common Files

- `openspec/changes/*/proposal.md`
- `openspec/changes/*/design.md`
- `openspec/changes/*/specs/*.md`
- `openspec/changes/*/tasks.md`
- `openspec/changes/archive/*`
- `openspec/specs/*.md`

## Suggested Sequence

1. Understand the current state and failure mode before editing.
2. Make the smallest coherent change that satisfies the workflow goal.
3. Run the most relevant verification for touched files.
4. Summarize what changed and what still needs review.

## Typical Commit Signals

- Create or update proposal and design documents under openspec/changes/<proposal-name>/
- Add or update capability specs under openspec/changes/<proposal-name>/specs/
- Update tasks.md to reflect progress or new tasks
- Audit status of in-flight proposals, marking them as landed, active, or archived
- Archive completed/inactive proposals to openspec/changes/archive/ and promote stable specs to openspec/specs/

## Notes

- Treat this as a scaffold, not a hard-coded script.
- Update the command if the workflow evolves materially.