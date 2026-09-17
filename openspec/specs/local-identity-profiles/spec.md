# local-identity-profiles Specification

## Purpose
TBD - created by archiving change define-local-identity-profiles. Update Purpose after archive.
## Requirements
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

Dovik SHALL provide a transparent `gh` proxy and `dovik gh -- ...` for ordinary proxy-level project terminals. Every ordinary invocation SHALL execute the original GitHub CLI with the configured project persona without switching global authentication state. Dovik SHALL preserve exact arguments, standard streams, terminal interaction, and exit codes, and SHALL pass through commands unknown to Dovik. GitHub CLI authentication mutation and direct token export SHALL remain unavailable through a configured project proxy. Explicit native and container agent-isolation sessions SHALL continue to validate supported commands and flags against the current project policy. Credentials SHALL remain inside the trusted execution boundary.

#### Scenario: Execute an ordinary GitHub CLI command

- **WHEN** a configured proxy-level caller invokes a GitHub CLI command that does not mutate authentication or disclose credentials
- **THEN** the proxy starts the original GitHub CLI in the caller's terminal with the configured persona
- **AND** forwards the exact arguments, input, output, and exit code
- **AND** does not require the command or its flags to appear in a Dovik catalogue

#### Scenario: Execute a future or user-installed command

- **WHEN** an ordinary proxy-level caller invokes a command, alias, or extension unknown to the installed Dovik version
- **THEN** the proxy delegates the invocation unchanged to the original GitHub CLI with the configured persona
- **AND** GitHub CLI determines whether the invocation is valid

#### Scenario: Inspect authentication

- **WHEN** an ordinary proxy-level caller runs `gh auth status` without a credential-disclosure flag
- **THEN** the original GitHub CLI reports the configured persona's authentication status normally
- **AND** Dovik does not reject the command as an unknown policy operation

#### Scenario: Attempt authentication mutation or direct token export

- **WHEN** an ordinary configured caller requests token export, login, logout, refresh, account switching, setup-git, or authentication status with token disclosure
- **THEN** the proxy rejects the invocation before acquiring or exposing a credential
- **AND** global GitHub CLI authentication state remains unchanged

#### Scenario: Execute a governed isolated operation

- **WHEN** a native or container agent session requests a supported operation permitted by the project policy
- **THEN** the executor verifies the repository and account, forwards the original output and exit code, and performs no automatic mutation retry

#### Scenario: Attempt an isolated-session escape

- **WHEN** an isolated agent requests token export, authentication changes, unrecognized flags, another repository, an editor, alias, extension, arbitrary GraphQL, or an unrecognized REST operation
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

CLI and TUI SHALL expose persona, project context, policy, and session state without secrets. The interactive questionnaire SHALL use the established Dovik terminal palette, SHALL describe proxy-level as transparent persona routing, and SHALL request policy presets, explicit operation exceptions, and alternative personas only when agent-isolation is selected. Native Windows, Linux, macOS, Docker, and Podman verification SHALL be reported separately; compilation alone SHALL NOT establish isolation support.

#### Scenario: Configure transparent proxy-level routing

- **WHEN** an operator configures a project in proxy-level mode
- **THEN** the questionnaire selects the persona and optional repository-local Git author
- **AND** does not present isolated-agent policy or permission questions
- **AND** preserves an existing stored policy for later isolation use

#### Scenario: Configure agent isolation

- **WHEN** an operator selects agent-isolation
- **THEN** the questionnaire requests the starting preset, explicit permission choices, and any permitted alternative personas
- **AND** includes those isolation controls in the confirmation summary

#### Scenario: Use consistent terminal presentation

- **WHEN** the root operator menu or identity questionnaire is rendered
- **THEN** both use the shared Dovik terminal palette and interaction styling
- **AND** do not introduce a parallel visual theme
