## ADDED Requirements

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
