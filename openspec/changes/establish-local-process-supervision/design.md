# Approach

## Terminology

- **Project**: a registered local development project identified by a stable ID and an absolute root directory.
- **Managed process**: a process definition owned by one project and any current or most recent runtime instance created from it.
- **Daemon**: the sole local authority that owns the registry, creates managed processes, observes their lifecycle, captures output, and serves the control protocol.
- **Client**: a local CLI or future GUI or MCP adapter that requests operations from the daemon and does not supervise operating-system processes directly.
- **Process state**: one of `stopped`, `starting`, `running`, `stopping`, `exited`, or `failed`.

## Models

`ProjectDefinition` contains a stable project ID and an absolute root directory.

`ProcessDefinition` contains an ID unique within its project, an executable command, ordered arguments, an optional working directory, and environment overrides. The working directory defaults to the project root. The daemon starts the executable directly without an intermediate shell.

`ProcessRuntime` contains a new instance ID for every start attempt, the operating-system PID when one was created, process state, start and finish times, exit code when available, and a termination reason. Persisted PIDs are diagnostic data only and are never authority to signal a process after daemon restart.

## Daemon and client architecture

The daemon is the only writer of project definitions, process definitions, and runtime state. It exposes a versioned JSON request-response protocol over a transport restricted to the current user. Native Windows uses a Named Pipe. The Linux container uses a Unix Domain Socket inside the container. The first control plane has no TCP listener or public port.

The first CLI is a thin client of that protocol. Its command surface covers adding, removing, and listing projects; adding, removing, and listing process definitions; and process `status`, `start`, `stop`, `restart`, and `logs --tail`. The architecture keeps the protocol independent from the CLI so later clients can share the same daemon contract.

## Registry and private state

The daemon stores its registry and essential runtime state in a private per-user state location. Updates use write-to-temporary-file, flush, and atomic replacement so an interrupted write cannot leave a partially written registry. Environment values are stored only where required to start a process and are never returned by list or status operations.

On daemon startup, every persisted non-terminal runtime is changed to `failed` with a daemon-restart reason. The daemon does not reattach and does not signal a recorded PID, because that PID may now identify an unrelated operating-system process.

## Lifecycle ownership

Each process definition has at most one active runtime in `starting`, `running`, or `stopping`. A repeated start returns the existing active instance rather than creating a duplicate. Stopping an inactive definition succeeds without side effects. Restart waits for confirmed termination before creating a runtime with a new instance ID. There are no automatic restarts or schedules.

Natural termination is observed and recorded with its exit information. A zero exit becomes `exited`; a non-zero exit or start failure becomes `failed`. A process tree that terminates after a requested stop becomes `stopped`. Resources and output writers are closed in all terminal paths.

Native Windows places every runtime in a per-process Job Object configured to terminate the owned process tree when the daemon closes the job. Linux starts every runtime in a separate process group and signals that group during stop. When `dovikd` runs as PID 1, container termination provides the outer cleanup boundary for every process in the container. The containerized daemon never manages host processes.

## Output

The daemon reads stdout and stderr concurrently and appends both to one ordered bounded event buffer. Every event includes a monotonically increasing sequence number, capture time, stream name, and bytes. The buffer belongs to the current or most recent runtime of a process definition. Clients can request a tail without learning environment values.

When the buffer reaches its byte or event limit, it discards the oldest complete events and records that truncation occurred. Output is not persisted and is unavailable after daemon restart.

## Cadder extension point

A future accepted change may let Dovik ask the independent Cadder service for an endpoint and pass the result to a process through environment overrides. This change defines no Cadder dependency, client interface, endpoint schema, or failure policy.

## Acceptance boundary

The first working scope is accepted only when automated tests and bounded platform smoke tests demonstrate all of the following:

- project and process definitions survive daemon restart while environment values remain absent from list and status responses;
- start prevents duplicate active instances, inactive stop is idempotent, and restart waits for complete termination before creating a new instance;
- start failure, natural exit, requested stop, and daemon interruption produce the specified terminal state, exit information, cleanup, and restart reconciliation;
- Windows Job Objects and Linux process groups clean the owned process tree without signaling persisted PIDs after daemon restart;
- stdout and stderr retain stream attribution and ordering, tail reads are bounded, and truncation is visible;
- every initial CLI operation works through the user-local protocol without TCP;
- the Linux image runs `dovikd` as PID 1, exposes no port, and cannot supervise host processes;
- the repository gates and strict OpenSpec validation pass.

# Trade-offs

- A daemon-owned atomic file registry is simpler than introducing a database in the first scope and is sufficient for one local writer.
- No reattach after daemon restart avoids signaling a reused PID and keeps ownership explicit, at the cost of marking previously active state as failed.
- An in-memory bounded output buffer provides one current log view without defining retention, rotation, or durable log storage.
- Platform-specific process-tree adapters are required for correct ownership semantics. A single portable child-process abstraction would not provide the required Windows Job Object and Linux process-group guarantees.
- Local-only transports keep the initial security and deployment boundary narrow. Remote control can be specified separately if it is ever accepted.
