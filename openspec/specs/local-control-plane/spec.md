# local-control-plane Specification

## Purpose
TBD - created by archiving change establish-local-process-supervision. Update Purpose after archive.
## Requirements
### Requirement: Daemon authority and versioned protocol

Clients SHALL request registry, lifecycle, status, and log operations through a versioned JSON protocol. The daemon SHALL remain the sole authority that directly manages operating-system processes.

#### Scenario: Client requests a lifecycle operation

- **WHEN** a client sends a supported request with a compatible protocol version
- **THEN** the daemon validates and performs the operation
- **AND** returns a structured JSON result or error

#### Scenario: Client uses an incompatible version

- **WHEN** a client sends an unsupported protocol version
- **THEN** the daemon rejects it with a structured version error and performs no requested mutation

### Requirement: User-local transport

Native Windows SHALL expose the protocol through a Named Pipe accessible only to the current user. Linux in Docker SHALL expose it through a Unix Domain Socket accessible only inside the container and to the configured container user. The first control plane SHALL NOT listen on TCP or publish a network port.

#### Scenario: Connect on native Windows

- **WHEN** the current user starts a client on the daemon host
- **THEN** the client connects through the user-restricted Named Pipe

#### Scenario: Connect inside the Linux container

- **WHEN** the CLI runs inside the daemon container
- **THEN** it connects through the container-local Unix Domain Socket without TCP

### Requirement: Initial CLI operations

The CLI SHALL support adding, removing, and listing projects; adding, removing, and listing process definitions; and process `status`, `start`, `stop`, `restart`, and `logs --tail` operations.

#### Scenario: Control a process through the CLI

- **WHEN** a user invokes a supported lifecycle command for an existing project and process definition
- **THEN** the CLI sends the corresponding protocol request and renders the daemon response

#### Scenario: Tail process output

- **WHEN** a user invokes `logs --tail` for an existing process definition
- **THEN** the CLI requests and renders the requested tail of the current or most recent runtime buffer
