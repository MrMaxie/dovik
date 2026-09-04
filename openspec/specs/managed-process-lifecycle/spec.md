# managed-process-lifecycle Specification

## Purpose
TBD - created by archiving change establish-local-process-supervision. Update Purpose after archive.
## Requirements
### Requirement: Explicit runtime model

Every start attempt SHALL create a `ProcessRuntime` with a new instance ID, process ID when available, state, start and finish times when known, exit code when available, and a termination reason. Runtime state SHALL be one of `stopped`, `starting`, `running`, `stopping`, `exited`, or `failed`.

#### Scenario: Record a successful start

- **WHEN** the operating system starts a configured process successfully
- **THEN** the runtime transitions from `starting` to `running`
- **AND** records its instance ID, process ID, and start time

#### Scenario: Record a start failure

- **WHEN** the operating system cannot start the configured command
- **THEN** the runtime transitions to `failed`
- **AND** records a termination reason without an active process ID

### Requirement: One active instance per definition

The daemon SHALL allow at most one runtime in `starting`, `running`, or `stopping` for a process definition.

#### Scenario: Start an already active process

- **WHEN** a client starts a process definition with an active runtime
- **THEN** the daemon returns the existing runtime
- **AND** does not create another operating-system process

### Requirement: Controlled stop and restart

Stopping an inactive definition SHALL succeed without side effects. Restart SHALL wait until the existing process tree has terminated before creating a new runtime instance. The daemon SHALL NOT schedule or automatically restart processes.

#### Scenario: Stop an inactive process

- **WHEN** a client stops a definition with no active runtime
- **THEN** the operation succeeds without changing its last terminal runtime

#### Scenario: Complete a requested stop

- **WHEN** the owned process tree terminates after a stop request
- **THEN** the runtime becomes `stopped`
- **AND** records its finish time, exit code when available, and requested-stop reason

#### Scenario: Restart an active process

- **WHEN** a client restarts a definition with an active runtime
- **THEN** the daemon requests termination and waits for confirmed process-tree exit
- **AND** only then starts a runtime with a new instance ID

### Requirement: Observe terminal outcomes

The daemon SHALL observe unexpected and requested process termination, close owned resources, and record the finish time, exit code when available, and termination reason.

#### Scenario: Process exits successfully

- **WHEN** a running process exits with code zero without a stop request
- **THEN** its runtime becomes `exited` and records exit code zero

#### Scenario: Process exits unsuccessfully

- **WHEN** a running process exits with a non-zero code
- **THEN** its runtime becomes `failed` and records that exit code

### Requirement: Own the process tree by platform

Native Windows SHALL place each runtime in a per-process Job Object configured for cleanup when the daemon closes it. Linux SHALL place each runtime in a separate process group. A containerized daemon running as PID 1 SHALL manage only processes inside its own container.

#### Scenario: Daemon terminates on Windows

- **WHEN** the native Windows daemon terminates and closes its Job Objects
- **THEN** every process tree owned by those Job Objects is terminated

#### Scenario: Stop a Linux runtime

- **WHEN** the Linux daemon stops a managed process
- **THEN** it signals the owned process group rather than only its initial process ID

### Requirement: Reconcile state without reattachment

After daemon restart, the daemon SHALL change every persisted non-terminal runtime to `failed`, SHALL NOT reattach, and SHALL NOT signal a process using only its persisted process ID.

#### Scenario: Load an interrupted runtime

- **WHEN** daemon startup loads a runtime persisted as `starting`, `running`, or `stopping`
- **THEN** the daemon marks it `failed` with a daemon-restart reason
- **AND** does not signal the recorded process ID
