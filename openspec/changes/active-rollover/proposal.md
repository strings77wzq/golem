# Active Rollover — Open Work from Partially-Landed Proposals

## Why

Slice E1 of `e2e-harness-and-spec-cleanup` audited the eight in-flight
OpenSpec proposals. Two were classified `partially-landed`:

- `fix-ci-vet-provider-health-types` — code work landed; CI run URL
  capture remains open.
- `project-review-and-roadmap` — sections 1–4 (Telegram fix, coverage
  uplift, provider health, session export/import) all landed; sections
  5–8 (memory completion, gateway audit log, agent integration, docs
  finalisation) are still open and warrant their own proposal lifecycle.

This rollover consolidates the still-open task blocks from both source
proposals so the source proposals can be archived cleanly without
losing the trailing work. Each carried-forward task block names its
origin via an HTML comment so future readers can trace decisions back.

## What Changes

- Carry `fix-ci-vet-provider-health-types` tasks 3.2 and 3.3 (CI
  evidence capture) forward.
- Carry `project-review-and-roadmap` sections 5 (memory module
  completion), 6 (gateway audit log), 7 (agent integration), and 8
  (documentation + version finalisation) forward verbatim.
- Do **not** restructure the carried tasks. Owners of the original
  proposals can claim them as-is; restructuring is out of scope for a
  rollover.

This change creates no new specs and modifies no capabilities. It is a
pure tracking artefact that exists so that:

1. The source proposals can be archived in the same Slice E2 commit
   without orphaning open work.
2. Future maintainers see the open tasks in a single place rather than
   inside an archived directory tree.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- None.

## Impact

- Affected code: none. This is a tracking proposal.
- Affected docs: `openspec/changes/active-rollover/` (this directory
  only). The two source proposals are archived; no other repo state
  changes from this rollover.
- Dependencies: none.

## Cleanup criteria

This rollover proposal can itself be archived once:

- `fix-ci-vet-provider-health-types` rollover items 3.2 and 3.3 have a
  CI run URL recorded in commit metadata or release notes.
- `project-review-and-roadmap` rollover sections 5–8 have either
  landed or been re-scoped into focused successor proposals
  (e.g. `memory-and-audit-completion`).

When that happens, archive this rollover with `--skip-specs` since it
defines no new spec deltas.
