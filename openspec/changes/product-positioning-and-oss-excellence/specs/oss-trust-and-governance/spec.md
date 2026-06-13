## ADDED Requirements

### Requirement: Support and security paths are discoverable
The repository SHALL provide discoverable root-level or GitHub-recognized guidance for user support and security reporting.

#### Scenario: User needs help or reports a vulnerability
- **WHEN** a user looks for support or security reporting instructions from the repository root
- **THEN** they can find the appropriate channel without reading unrelated implementation docs

### Requirement: Contribution path is explicit
The repository SHALL document how an external contributor can set up the project, run verification, choose an issue, and submit a pull request.

#### Scenario: New contributor prepares a PR
- **WHEN** a new contributor follows contribution guidance
- **THEN** they can identify setup steps, local checks, issue/label expectations, and pull request expectations

### Requirement: Release maturity is documented
The project SHALL document release cadence, compatibility expectations, and artifact expectations for public releases.

#### Scenario: User evaluates upgrade risk
- **WHEN** a user reads release or changelog guidance
- **THEN** they can determine how compatibility, breaking changes, and release artifacts are handled

### Requirement: Trust checks are visible
The repository SHALL make its quality gates visible to users and contributors.

#### Scenario: Contributor checks quality gates
- **WHEN** a contributor prepares a change
- **THEN** they can identify the expected lint, test, race, build, security, and documentation checks for the change type
