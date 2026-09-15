## ADDED Requirements

### Requirement: Private personas and project intent

The daemon SHALL own versioned, atomically persisted persona metadata and project intent outside project repositories. Personas SHALL identify a Git author and a GitHub host/account without containing credentials. Project intent SHALL bind a root, repository, persona, allowed alternatives, explicit policy, and enforcement mode.

#### Scenario: Configure a project
- **WHEN** an operator completes the terminal questionnaire and confirms its summary
- **THEN** the daemon validates and saves the complete configuration
- **AND** canceling or omitting the policy grants no permissions

#### Scenario: Replace git-user
- **WHEN** an operator previews and confirms import of a legacy persona file and selects a persona
- **THEN** Dovik provides project-local Git author configuration and concise prompt status without changing global Git configuration

### Requirement: Account-specific governed gh execution

Dovik SHALL provide a gh proxy and `dovik gh -- ...`, validate supported commands and flags against the current project policy, and execute the original gh outside untrusted checkouts. Credentials SHALL remain inside the trusted execution boundary, and each invocation SHALL select its account without switching global authentication state.

#### Scenario: Execute an allowed operation
- **WHEN** a configured caller requests a supported operation permitted by the project policy
- **THEN** the executor verifies the repository and account, forwards the original output and exit code, and performs no automatic mutation retry

#### Scenario: Create a pull request interactively
- **WHEN** an operator runs `gh pr create` without non-interactive title and body flags from a proxy-level project terminal
- **THEN** Dovik starts the original GitHub CLI in that terminal with the configured persona and repository
- **AND** the original prompts, input, output, and exit code remain unchanged

#### Scenario: Attempt an escape
- **WHEN** an agent requests token export, authentication changes, unrecognized flags, another repository, an editor, alias, extension, arbitrary GraphQL, or an unrecognized REST operation
- **THEN** the executor rejects the request before acquiring credentials or starting gh

#### Scenario: Use Git over HTTPS in a proxy-level project
- **WHEN** an operator configures a proxy-level project and Git requests credentials through the repository-local `gh auth git-credential` helper
- **THEN** Dovik validates the Git credential protocol, host, and optional repository path before returning the configured persona's credential to Git
- **AND** direct token export, other hosts, other repositories, and isolated sessions remain denied
- **AND** Git credential `store` and `erase` notifications complete without mutating authentication state

### Requirement: Explicit enforcement modes and revocable sessions

Proxy-level SHALL be the default and SHALL describe its same-user bypass limitation. Agent-isolation SHALL use an operator-provisioned native account or a Docker/Podman container, with a restricted channel separate from daemon administration. Sessions SHALL be project-bound and revocable, SHALL use current policy, and SHALL not survive daemon restart.

#### Scenario: Use native isolation
- **WHEN** an operator enables a session for a separate non-administrative account on Windows, Linux, or macOS
- **THEN** Dovik authenticates the OS peer through a restricted Named Pipe or Unix Domain Socket and rejects administrative operations

#### Scenario: Use container isolation
- **WHEN** an operator starts an approved Docker or Podman agent session
- **THEN** the agent communicates through a private stdio bridge without exposed control ports or access to operator credentials and runtime administration

#### Scenario: Isolation is unavailable
- **WHEN** the requested isolation boundary cannot be established
- **THEN** session startup fails without falling back to proxy-level

### Requirement: Consistent operator surfaces and platform evidence

CLI and TUI SHALL expose persona, project context, policy, and session state without secrets. The questionnaire SHALL support presets and explicit operation exceptions. Native Windows, Linux, macOS, Docker, and Podman verification SHALL be reported separately; compilation alone SHALL NOT establish isolation support.

#### Scenario: Use gh without configured project identity
- **WHEN** a caller invokes gh from a repository without approved project intent
- **THEN** the proxy delegates the invocation unchanged to the original GitHub CLI with its normal authentication and output
- **AND** it does not prompt for a persona, policy, or project configuration
