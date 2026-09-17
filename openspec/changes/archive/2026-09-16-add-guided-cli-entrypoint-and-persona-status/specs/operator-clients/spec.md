## ADDED Requirements

### Requirement: Guided default operator entrypoint

An interactive bare `dovik` invocation SHALL present a compact operator menu whose default action is help. Inside a Git repository it SHALL also offer project identity configuration with a label that distinguishes configured and unconfigured projects when daemon state is available, and it SHALL offer the TUI. Outside a Git repository it SHALL omit project configuration. A redirected bare invocation SHALL print help and exit successfully without contacting the daemon or emitting terminal control sequences.

#### Scenario: Open Dovik in a configured repository

- **WHEN** an operator runs bare `dovik` in an interactive terminal inside a configured Git repository
- **THEN** help is the default selection
- **AND** the menu offers `Edit this repository` and `Open TUI`

#### Scenario: Open Dovik in an unconfigured repository

- **WHEN** an operator runs bare `dovik` in an interactive terminal inside an unconfigured Git repository
- **THEN** the menu offers `Configure this repository`
- **AND** selecting it opens the existing questionnaire for that repository root

#### Scenario: Open Dovik outside a repository

- **WHEN** an operator runs bare `dovik` outside a Git repository
- **THEN** the menu offers help and the TUI
- **AND** omits project configuration

#### Scenario: Redirect bare Dovik output

- **WHEN** bare `dovik` does not have interactive input and output
- **THEN** it prints normal help to stdout and exits with status zero
- **AND** does not contact or start the daemon

#### Scenario: Cancel the operator menu

- **WHEN** an operator presses Escape or Ctrl+C before choosing an action
- **THEN** Dovik exits with status zero
- **AND** performs no configuration or daemon action

### Requirement: Prompt-safe persona status

`dovik whoami` SHALL provide a bounded local answer describing the selected persona for the current repository or isolated session without contacting GitHub. Human output SHALL be exactly one persona name, `?`, or `!` plus a newline. JSON output SHALL expose `state`, `configured`, and `persona` without transport diagnostics.

#### Scenario: Inspect a configured repository

- **WHEN** the current repository has a valid configured persona
- **THEN** human output is the persona display name and status is zero
- **AND** JSON state is `configured`, `configured` is true, and `persona` is the display name

#### Scenario: Inspect an unconfigured repository

- **WHEN** the current Git repository has no Dovik project identity
- **THEN** human output is `?` and status is zero
- **AND** JSON state is `unconfigured`, `configured` is false, and `persona` is null

#### Scenario: Inspect a non-repository directory

- **WHEN** the current directory is not inside a Git repository
- **THEN** human output is `?` and status is zero
- **AND** JSON state is `not_repository`

#### Scenario: Status is unavailable

- **WHEN** the daemon, session context, or configured persona cannot provide a valid answer within 400 milliseconds
- **THEN** human output is `!` and status is zero
- **AND** JSON state is `unavailable`, `configured` and `persona` are null

#### Scenario: Inspect an isolated session

- **WHEN** `DOVIK_AGENT_ENDPOINT` identifies an active isolated session
- **THEN** `dovik whoami` reads the allowed session context instead of the operator identity API
- **AND** reports the session's active persona without expanding restricted administration access
