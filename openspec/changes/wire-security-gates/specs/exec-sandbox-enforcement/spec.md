## ADDED Requirements

### Requirement: Shell execution is disabled by default
The system SHALL execute `exec` tool commands without shell interpretation by default.

#### Scenario: Simple command executes without shell
- **WHEN** the `exec` tool receives an allowlisted command such as `pwd` under default configuration
- **THEN** the command is executed directly without invoking `sh -c`

#### Scenario: Shell syntax is not interpreted by default
- **WHEN** the `exec` tool receives a command containing shell-only syntax such as pipes or redirects under default configuration
- **THEN** the command is not interpreted through a shell and cannot use shell expansion to bypass validation

#### Scenario: Shell mode requires explicit configuration
- **WHEN** an operator configures the `exec` tool with explicit shell allowance
- **THEN** shell interpretation is enabled only for that configured tool instance

### Requirement: Command allow and deny checks run before execution
The system SHALL validate every `exec` tool command against configured allow and deny rules before starting a process.

#### Scenario: Denied command is blocked before process start
- **WHEN** the `exec` tool receives a command matching a denied command or denied pattern
- **THEN** the tool returns a security error and does not start an operating system process

#### Scenario: Non-allowlisted command is blocked in sandbox mode
- **WHEN** the `exec` tool runs in sandbox mode and receives a command whose executable name is not in the allowlist
- **THEN** the tool returns a security error and does not start an operating system process

#### Scenario: Allowlisted command passes built-in validation
- **WHEN** the `exec` tool runs in sandbox mode and receives an allowlisted command that does not match any denied rule
- **THEN** the command passes built-in validation and proceeds to custom validator checks if configured

### Requirement: Custom sandbox validator integration
The system SHALL support a custom command validator and SHALL run it before process execution.

#### Scenario: Custom validator blocks command
- **WHEN** the `exec` tool is configured with a validator and the validator rejects a command
- **THEN** the tool returns a security error and does not start an operating system process

#### Scenario: Custom validator accepts command
- **WHEN** the `exec` tool is configured with a validator and the validator accepts a command after built-in checks pass
- **THEN** the tool may execute the command within the configured workspace and timeout

#### Scenario: Validator error is visible to the tool caller
- **WHEN** the custom validator rejects a command with a reason
- **THEN** the tool result includes a non-secret security error message suitable for the LLM and user-facing channel

### Requirement: Workspace boundary is enforced
The system SHALL run commands from the configured workspace and SHALL validate workspace-sensitive paths when a path-aware sandbox is configured.

#### Scenario: Command runs in configured workspace
- **WHEN** the `exec` tool executes a permitted command
- **THEN** the process working directory is the configured workspace

#### Scenario: Denied path is blocked
- **WHEN** a path-aware sandbox validator detects that a command targets a denied path
- **THEN** the tool returns a security error and does not start an operating system process

#### Scenario: Path outside allowed roots is blocked
- **WHEN** a path-aware sandbox validator detects that a command targets a path outside configured allowed roots
- **THEN** the tool returns a security error and does not start an operating system process

### Requirement: Exec audit and metrics events
The system SHALL expose security-relevant exec decisions through audit and metric hooks when configured.

#### Scenario: Blocked command emits audit event
- **WHEN** an `exec` command is blocked by built-in checks or a custom validator and an audit sink is configured
- **THEN** the audit sink receives command name, reason, workspace, and status `denied` without command output

#### Scenario: Successful command emits audit event
- **WHEN** a permitted `exec` command completes and an audit sink is configured
- **THEN** the audit sink receives command name, duration, exit status, workspace, and status `success` without leaking full command output by default

#### Scenario: Exec denial increments security metric
- **WHEN** an `exec` command is blocked by a security check and security metrics are configured
- **THEN** the relevant denied security counter is incremented
