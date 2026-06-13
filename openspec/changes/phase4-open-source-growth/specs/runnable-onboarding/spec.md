## ADDED Requirements

### Requirement: User can reach first successful run quickly
The system SHALL provide a documented and verifiable first-run path that takes a new developer from installation to a successful Golem interaction in a few minutes.

#### Scenario: Quick start produces first success
- **WHEN** a new developer follows the primary quick start flow
- **THEN** they SHALL be able to install Golem, configure one provider, run one command, and observe a successful response without reading architecture documentation first

### Requirement: Installation paths cover primary target environments
The project SHALL provide installation instructions for Linux amd64 and Android/Termux ARM64, and SHALL clearly distinguish official versus community-supported environments.

#### Scenario: Target platform matrix is documented
- **WHEN** a user opens installation documentation
- **THEN** they SHALL see supported platforms, install methods, prerequisites, and known limitations for each primary environment

### Requirement: First-run validation is explicit
The project SHALL include a validation step that confirms whether the local setup is correct.

#### Scenario: User verifies setup
- **WHEN** a user completes configuration
- **THEN** the documentation or product flow SHALL provide a concrete validation command or action and expected success output
