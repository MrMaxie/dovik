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
