## ADDED Requirements

### Requirement: Truthful state of `openspec/changes/`

The `openspec/changes/` directory SHALL contain only proposals whose work is actively in progress or genuinely planned. Proposals whose tasks have all landed in the codebase SHALL be moved to `openspec/specs/archive/` within one cleanup pass per release cycle.

#### Scenario: All-landed proposal is archived

- **WHEN** every task in a proposal's `tasks.md` has a corresponding merged commit in `git log`
- **AND** there is no remaining active work referenced by the proposal
- **THEN** the proposal directory SHALL be moved to `openspec/specs/archive/<YYYY-MM-DD>-<name>/`
- **AND** the move SHALL use the `openspec archive` workflow rather than ad-hoc `git mv`

#### Scenario: Active proposal stays in place

- **WHEN** a proposal has at least one task that is not yet implemented in `main`
- **AND** the maintainer confirms the work is still planned
- **THEN** the proposal SHALL remain in `openspec/changes/`
- **AND** its `proposal.md` SHALL begin with a quoted status line of the form `> Status: active — last reviewed YYYY-MM-DD`

### Requirement: Partial completion handled by rollover, never by silent drop

When a proposal's tasks are partially landed, the unfinished tasks SHALL be migrated to a follow-up proposal so no committed work is silently abandoned and no completed work is needlessly retained as "in progress".

#### Scenario: Partial proposal becomes archive + rollover

- **WHEN** an audit determines a proposal has both landed tasks and genuinely unfinished tasks
- **THEN** the unfinished tasks SHALL be copied verbatim into a new `openspec/changes/<name>-rollover/tasks.md` (or grouped into a single `active-rollover` proposal if the remainders are coherent across multiple sources)
- **AND** the original proposal SHALL then be archived as a landed proposal
- **AND** the rollover proposal's `proposal.md` SHALL link back to each source proposal it inherited tasks from

### Requirement: Archive verdicts must be evidenced

Every archive or rollover decision in a cleanup pass SHALL be recorded with verifiable evidence (commit SHAs, file paths, or test names) in an `audit.md` document that lives with the cleanup change.

#### Scenario: Audit document captures the verdict

- **WHEN** a cleanup change archives one or more proposals
- **THEN** the cleanup change directory SHALL contain `audit.md`
- **AND** for each proposal audited, `audit.md` SHALL list:
  - the proposal name
  - the verdict (`landed` / `partially-landed` / `active`)
  - the evidence (commit SHAs implementing the tasks, or the specific tasks still open)
  - the resulting action (`archived` / `archived + rollover` / `kept active`)

#### Scenario: Archive without evidence is rejected

- **WHEN** a reviewer evaluates a proposed cleanup change
- **AND** an archive verdict in `audit.md` lacks at least one piece of evidence
- **THEN** the reviewer SHALL request the evidence be added before approving the cleanup

### Requirement: Active proposals carry a status header

Any proposal kept in `openspec/changes/` for longer than one release cycle SHALL carry a status header so future contributors can tell at a glance whether the proposal reflects current intent.

#### Scenario: Long-lived active proposal gets status line

- **WHEN** a proposal remains in `openspec/changes/` after a cleanup pass
- **THEN** its `proposal.md` SHALL contain a top-of-file line of the form `> Status: active — last reviewed YYYY-MM-DD`
- **AND** the date SHALL be no older than the most recent cleanup pass

#### Scenario: Status header drift is a smell

- **WHEN** the most recent `> Status: active — last reviewed` date is more than 90 days old
- **THEN** the proposal SHALL be reviewed for archive or rollover in the next cleanup pass
- **AND** the absence of recent review SHALL be noted in the next cleanup's `audit.md`
