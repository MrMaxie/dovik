## MODIFIED Requirements

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
