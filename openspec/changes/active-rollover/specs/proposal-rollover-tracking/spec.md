# proposal-rollover-tracking — spec deltas

## ADDED Requirements

### Requirement: Open tasks from partially-landed proposals SHALL survive archival via a rollover change

The cleanup pass SHALL carry every still-open task from a partially-landed proposal forward into a single `openspec/changes/active-rollover/` change before archiving the source proposal. The rollover SHALL exist only when at least one partially-landed proposal contributes open work; an empty rollover MUST NOT be created. Each carried-forward task block in the rollover SHALL be prefixed with an HTML comment of the form `<!-- inherited from <source-proposal> -->` so its origin is recoverable after the source proposal moves into the archive.

#### Scenario: Two partially-landed proposals roll over into a single change

- **WHEN** Slice E1 audit classifies proposal A and proposal B as
  partially-landed, with open tasks A.X and B.Y
- **THEN** Slice E2 SHALL create `openspec/changes/active-rollover/`
  containing a `proposal.md` that names A and B as sources, plus a
  `tasks.md` whose entries for A.X and B.Y each carry an
  `<!-- inherited from <source-proposal> -->` comment immediately
  before or above the task block, AND
- **THEN** proposals A and B SHALL be archived as if landed (their
  open tasks live exclusively in the rollover after archival).

#### Scenario: No partially-landed proposals means no rollover is created

- **WHEN** Slice E1 audit classifies every proposal as either
  `landed` or `active`
- **THEN** Slice E2 MUST NOT create `openspec/changes/active-rollover/`
  (no empty rollover).

### Requirement: A rollover change SHALL itself archive once its carried-forward work is resolved

A rollover change SHALL be archived as soon as every carried-forward task has either landed in the codebase or been re-scoped into a focused successor proposal. The rollover MUST be archived with `--skip-specs` because it defines no runtime capability, only the proposal-tracking discipline established here.

#### Scenario: Rollover archives when its open work is resolved

- **WHEN** every task in `openspec/changes/active-rollover/tasks.md`
  has been completed or absorbed into a successor proposal
- **THEN** a maintainer SHALL run `openspec archive active-rollover --skip-specs`
- **AND** the resulting archive SHALL appear under
  `openspec/specs/archive/<YYYY-MM-DD>-active-rollover/`.
