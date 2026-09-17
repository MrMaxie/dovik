# operator-clients Specification

## Purpose
TBD - created by archiving change add-cli-tui-mvp. Update Purpose after archive.
## Requirements
### Requirement: Complementary CLI and TUI surfaces

The `dovik` executable SHALL provide a complete non-interactive CLI and a `dovik tui` interactive terminal client. The CLI SHALL cover every initial registry, lifecycle, status, and log operation. The TUI SHALL cover navigation, status, start, stop, restart, and bounded output for existing definitions.

#### Scenario: Use the CLI as the complete client

- **WHEN** a user or coding agent needs any supported registry or process operation
- **THEN** the operation is available as a non-interactive CLI command
- **AND** using the TUI is not required

#### Scenario: Use the TUI for repeated process operation

- **WHEN** a developer starts `dovik tui` in an interactive terminal
- **THEN** the client provides project and process navigation, current state, lifecycle actions, and output access
- **AND** directs registry mutations to the corresponding CLI commands

### Requirement: Shared protocol client and daemon authority

The CLI and TUI SHALL use the same versioned local protocol client. Neither client SHALL read or write daemon registry state directly, create managed operating-system processes, or implement an independent lifecycle state machine.

#### Scenario: Perform an operation from either client

- **WHEN** the CLI or TUI requests a supported operation
- **THEN** the shared client sends the request through the configured user-local transport
- **AND** the daemon validates and performs the operation

#### Scenario: Reject an incompatible daemon

- **WHEN** either client connects to a daemon with an incompatible protocol version
- **THEN** the client presents a version incompatibility error
- **AND** does not attempt the requested mutation through another path

### Requirement: Deterministic CLI output modes

Every CLI command SHALL provide concise human-readable output by default and SHALL support a global `--json` mode. JSON mode SHALL emit exactly one JSON document without ANSI styling or interactive prompts and SHALL preserve the daemon's omission of environment override values.

#### Scenario: Run a successful command in JSON mode

- **WHEN** a user runs a supported command with `--json` and the daemon succeeds
- **THEN** the CLI writes one structured JSON result to stdout
- **AND** writes no progress or presentation text around that document
- **AND** exits with status zero

#### Scenario: Run a failed command in JSON mode

- **WHEN** a command with `--json` fails validation, connection, protocol negotiation, or daemon execution
- **THEN** the CLI writes one structured JSON error to stderr
- **AND** exits with a non-zero status

#### Scenario: Redirect normal CLI output

- **WHEN** a user runs a human-readable command with redirected output
- **THEN** the CLI emits no ANSI styling or interactive prompt

### Requirement: TUI operational overview

The TUI SHALL present registered projects and their process definitions, the selected process state, available lifecycle actions, and ordered output from the current or most recent runtime in one navigable operator view.

#### Scenario: Open the TUI with registered processes

- **WHEN** the TUI loads registry and status data from a compatible daemon
- **THEN** it shows the project and process hierarchy
- **AND** preserves selection by stable project and process IDs across refreshes
- **AND** shows the selected process state and applicable actions

#### Scenario: Inspect extended runtime details

- **WHEN** a user explicitly opens details for the selected process
- **THEN** the TUI shows available instance ID, process ID, timestamps, exit code, and termination reason
- **AND** does not show environment override values

### Requirement: TUI lifecycle control

The TUI SHALL support start, stop, and restart for the selected process definition. It SHALL serialize mutation requests for that selection, show when a request is in progress, and refresh the affected state after the daemon responds.

#### Scenario: Start a selected stopped process

- **WHEN** a user invokes start for a selected inactive process
- **THEN** the TUI sends one start request through the shared client
- **AND** prevents another mutation for that selection while the request is pending
- **AND** renders the returned runtime state

#### Scenario: Restart a selected active process

- **WHEN** a user invokes restart for a selected active process
- **THEN** the TUI sends the daemon restart operation rather than composing a client-side stop and start
- **AND** renders the new runtime only after the daemon confirms the result

#### Scenario: Operation fails

- **WHEN** a lifecycle request fails
- **THEN** the TUI keeps the current selection
- **AND** shows what did not happen and an available retry or recovery action
- **AND** leaves protocol diagnostics behind an explicit details action

### Requirement: TUI bounded output refresh

The TUI SHALL periodically request sequence-aware bounded output for the selected process without requiring a streaming protocol. It SHALL preserve event order and stream attribution, avoid overlapping refreshes for one selection, and ignore results made obsolete by a selection change.

#### Scenario: Follow current output

- **WHEN** the selected runtime produces new stdout or stderr events while the TUI is connected
- **THEN** the TUI appends them in daemon sequence order within two seconds under normal local operation
- **AND** identifies each event's stream without relying only on color

#### Scenario: Earlier output was truncated

- **WHEN** the daemon reports that older output events were discarded
- **THEN** the TUI shows a persistent truncation notice for that runtime
- **AND** continues to render the available ordered tail

#### Scenario: Selection changes during refresh

- **WHEN** a user selects another process before an earlier refresh completes
- **THEN** the TUI does not render the obsolete result under the new selection

### Requirement: Terminal-safe keyboard operation

Every MVP TUI function SHALL be operable by keyboard, SHALL expose its active key bindings in context or help, and SHALL not rely only on color to communicate state. The TUI SHALL handle terminal resizing and SHALL restore terminal state after normal exit or a handled failure.

#### Scenario: Resize below the functional layout

- **WHEN** the terminal becomes too small to render the operator view
- **THEN** the TUI shows the required minimum dimensions
- **AND** retains working help and quit controls
- **AND** restores the operator view when sufficient space returns

#### Scenario: Start without an interactive terminal

- **WHEN** `dovik tui` starts without an interactive terminal
- **THEN** it exits non-zero with a concise instruction to use CLI commands or an interactive terminal
- **AND** does not emit terminal control sequences

#### Scenario: Daemon is unavailable

- **WHEN** the TUI cannot connect to the local daemon
- **THEN** it shows that process state is unavailable
- **AND** provides retry and quit actions
- **AND** does not display raw transport data as the primary error

### Requirement: Initial runtime boundary

The CLI and TUI SHALL operate through a current-user Windows Named Pipe on native Windows and through the container-local Unix Domain Socket when executed inside the Linux container. Neither client SHALL require TCP, HTTP, a published port, or access to host processes from the container.

#### Scenario: Use both clients on Windows

- **WHEN** the current Windows user runs CLI commands or the TUI against the local daemon
- **THEN** both connect through the same user-restricted Named Pipe contract

#### Scenario: Use both clients in the Linux container

- **WHEN** a user executes the CLI or TUI interactively inside the daemon container
- **THEN** the client connects through the container-local Unix Domain Socket
- **AND** no network port is published

### Requirement: Independent native daemon launch from the TUI

When the native Windows TUI cannot connect to the default local daemon, it SHALL let the operator explicitly start the colocated daemon as an independent peer process. The daemon SHALL NOT inherit the TUI terminal or standard streams, SHALL NOT be owned by TUI cancellation or cleanup, and SHALL remain running after the TUI exits, restarts, or fails. After launch, the TUI SHALL prove readiness and perform every registry and managed-process operation through the existing versioned local protocol.

#### Scenario: Start the unavailable default daemon

- **WHEN** the native Windows TUI cannot connect to the default current-user Named Pipe and the operator invokes `Start daemon`
- **THEN** the TUI starts one detached colocated `dovikd.exe` peer without inherited terminal input or output
- **AND** identifies daemon startup as pending without accepting another launch action
- **AND** loads registry state only after a compatible response arrives through the default Named Pipe

#### Scenario: Leave the TUI after starting the daemon

- **WHEN** a daemon started from the TUI becomes ready and the TUI then exits, restarts, or fails
- **THEN** the daemon remains running and connectable to a later CLI or TUI client
- **AND** the TUI does not wait for, stop, kill, or otherwise supervise the daemon during cleanup

#### Scenario: Race with another daemon launch

- **WHEN** another compatible daemon claims the default endpoint while the TUI launch is pending
- **THEN** the TUI connects to the compatible daemon that became authoritative
- **AND** does not report the losing launch attempt as the primary failure
- **AND** does not create a client-side daemon lifecycle state machine

#### Scenario: Daemon launch does not become ready

- **WHEN** the colocated daemon cannot be started or no compatible daemon responds before the bounded readiness deadline
- **THEN** the TUI returns to an unavailable state and states that the daemon did not start or become available
- **AND** exposes applicable start, retry, diagnostic, and quit actions without raw process or transport detail in the primary message

#### Scenario: Use a connection-only runtime

- **WHEN** an ordinary production TUI uses a custom endpoint or runs inside the Linux daemon container
- **THEN** it does not offer or attempt daemon launch
- **AND** preserves connection retry, diagnostic, and quit behavior appropriate to that runtime

#### Scenario: Open the TUI while the daemon is unavailable

- **WHEN** the native Windows TUI first detects that the default daemon is unavailable
- **THEN** it waits for explicit operator action and does not start a process automatically

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

### Requirement: Follow retained process output

The CLI SHALL support `dovik process logs --follow` for any registered process. Follow mode SHALL print the retained tail, then request only later sequence events until interrupted, regardless of whether the process is running or stopped. Human output SHALL preserve safe process color styling while retaining textual stream attribution; JSON mode SHALL reject follow mode because it cannot produce one finite JSON document.

#### Scenario: Follow a running process

- **WHEN** an operator runs `dovik process logs --project PROJECT --process PROCESS --follow`
- **THEN** the CLI prints the requested retained tail and subsequently prints new ordered events
- **AND** exits cleanly when its context is cancelled

#### Scenario: Follow a stopped process

- **WHEN** the selected process has retained output but is stopped
- **THEN** the CLI prints that output and continues waiting for later events
- **AND** does not require or start a runtime

#### Scenario: Request JSON follow mode

- **WHEN** `--json` and `--follow` are combined
- **THEN** the CLI rejects the incompatible output modes with an invalid-arguments error
