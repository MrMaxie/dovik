# mcp-operator-client Specification

## Purpose
Expose registered local processes to MCP clients through a stdio adapter while preserving daemon authority, bounded calls, and private environment configuration.
## Requirements
### Requirement: Local MCP stdio session

Dovik SHALL expose `dovik [--endpoint PATH] mcp` as an MCP server over stdio on native Windows and Linux inside Docker. Standard output SHALL contain only MCP protocol messages. The adapter SHALL NOT open a network listener, launch a daemon, or accept `--json` or extra command arguments.

#### Scenario: Discover tools without a daemon

- **WHEN** a client initializes an MCP session while the daemon is unavailable
- **THEN** initialization and tool discovery succeed
- **AND** daemon-backed calls return MCP tool errors without launching a daemon

#### Scenario: Close an MCP session

- **WHEN** the client closes stdin or cancels the adapter session
- **THEN** the MCP adapter exits
- **AND** the daemon and its managed processes remain independent

### Requirement: Daemon-backed operator tools

The MCP server SHALL expose exactly `list_projects`, `list_processes`, `process_status`, `process_logs`, `process_start`, `process_stop`, and `process_restart`. All operations SHALL use the existing operator client and versioned local IPC. The adapter SHALL NOT access the registry directly or provide definition mutation or arbitrary execution tools.

#### Scenario: Inspect registered definitions and runtime

- **WHEN** a client lists definitions or requests status with an existing project and process ID
- **THEN** the tool returns structured daemon-derived results
- **AND** status distinguishes a process with no runtime from a recorded runtime
- **AND** definition results omit stored environment overrides

#### Scenario: Change a managed process lifecycle

- **WHEN** a client calls start, stop, or restart with nonempty project and process IDs
- **THEN** the adapter delegates one matching operation to the daemon and returns its runtime result
- **AND** does not retry the mutation or supervise the process itself

#### Scenario: Describe tool effects

- **WHEN** a client discovers the tools
- **THEN** inspection tools are annotated read-only
- **AND** lifecycle tools are annotated potentially destructive and non-idempotent

### Requirement: Bounded output and failures

Tools SHALL validate required nonempty IDs and input schemas before IPC. Logs SHALL accept an optional nonnegative `afterSequence` cursor and a `limit` from 1 to 1000, defaulting to 100. Daemon calls SHALL have a maximum 30-second wait and honor request cancellation. Operational failures SHALL return MCP tool errors with stable codes without raw private diagnostics.

#### Scenario: Read captured output

- **WHEN** a client requests logs for a registered process
- **THEN** it receives at most the requested number of retained events, each with sequence, timestamp, stream, and text data, plus the daemon truncation flag
- **AND** the server describes output as untrusted process data

#### Scenario: Reject invalid input

- **WHEN** a client omits required IDs, supplies empty IDs, unknown fields, wrong types, or an out-of-range log limit
- **THEN** the tool rejects the request before any daemon operation

#### Scenario: Daemon failure or canceled wait

- **WHEN** IPC fails, the daemon rejects an operation, or the request is canceled or times out
- **THEN** the adapter reports failure without fabricating runtime state or retrying a mutation
- **AND** cancellation does not claim to undo an operation the daemon has accepted
