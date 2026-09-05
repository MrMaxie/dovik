# Dovik

Dovik is a local supervisor for development processes. Its intended responsibility is to know which processes belong to which local projects, start them under explicit control, observe their lifecycle, collect their output, and expose one local control plane to human and agent clients.

## Current status

The current implementation provides the first working local supervisor. `dovikd` owns the private project registry, managed process lifecycle, bounded stdout and stderr output, and the versioned local control protocol. `dovik` provides non-interactive commands for registry, lifecycle, status, and log operations.

Native Windows clients connect through a current-user Named Pipe. The Linux image runs `dovikd` as PID 1 and uses a container-local Unix Domain Socket without publishing a network port. The CLI and daemon do not expose stored environment override values through list or status responses.

The target version is `1.0.0` under SemVer. It is currently in soak and has not been published as a stable release.

## Responsibility boundary

Dovik manages the lifecycle of local development processes. It is not a reverse proxy, DNS server, domain manager, remote executor, scheduler, or container orchestrator.

[Cadder](https://github.com/MrMaxie/cadder) owns local service addressability, including domains, HTTPS, and routing. A future Dovik integration may optionally request an endpoint from Cadder and pass it to a process through its environment, but Cadder is not a dependency of Dovik and no integration is part of the initial scope.

The first runtime targets native Windows and Linux inside Docker. A containerized Dovik daemon is isolated to its container and does not control host processes.

## Development

Install the pinned Go toolchain and developer tools:

```console
mise install
```

OpenSpec must be installed separately and available on `PATH`. The project does not install it through Node or npm.

Common commands:

```console
just build
just test
just format
just lint
just check
just run
just run-cli
just tui
just test-linux
just test-integration
just run-linux
```

`just check` runs Go vet, tests, builds, schema validation, and strict validation of all OpenSpec artifacts. `just test-linux` builds the Docker test stage. `just test-integration` uses Testcontainers to verify the Linux daemon, CLI, persistence, lifecycle, logs, local socket, and container boundary. `just run-linux` builds and runs the Linux runtime image without publishing a port.

## CLI

Start the native daemon:

```console
just run
```

Register a project and one process from another terminal:

```console
just run-cli project add --id example --root C:\path\to\project
just run-cli process add --project example --id api --command C:\path\to\server.exe --arg=serve --env PORT=7881
```

Operate the registered process:

```console
just run-cli process start --project example --process api
just run-cli process status --project example --process api
just run-cli process logs --project example --process api --tail 100
just run-cli process restart --project example --process api
just run-cli process stop --project example --process api
```

Use the corresponding `project list` and `process list --project example` commands to inspect definitions. Remove an inactive process with `process remove`, then remove its empty project with `project remove`.

### Automation and JSON output

Every non-interactive project or process command accepts the global `--json` flag before the command name. A successful command writes one JSON document to standard output. Validation, connection, protocol, and daemon failures write one structured JSON error to standard error and return a non-zero exit status.

```console
just run-cli --json project list
just run-cli --json process status --project example --process api
just run-cli --json process logs --project example --process api --tail 100
```

Process definitions returned by list commands do not contain environment override values. Environment overrides can be supplied only when adding a process and remain private daemon registry configuration.

## MCP

Agent clients can run `dovik mcp` as a local stdio MCP server. It connects to the same daemon as the CLI and TUI. Start `dovikd` separately and register projects and processes through the CLI before operating them from MCP.

Build the binaries if needed:

```console
mise exec -- go build -o build/dev/ ./cmd/dovik ./cmd/dovikd
```

Configure your MCP host with the path to the built `dovik.exe` on Windows or `dovik` on Linux. If the binary is on `PATH`, a host using `mcpServers` configuration can use:

```json
{
  "mcpServers": {
    "dovik": {
      "command": "dovik",
      "args": ["mcp"]
    }
  }
}
```

For a custom local daemon endpoint, use `"args": ["--endpoint", "PATH", "mcp"]`, replacing `PATH` with its Named Pipe or Unix Domain Socket path. Inside Docker, run the MCP adapter in the daemon container as the same user. No network port is needed.

| Tool | Arguments | Result |
| --- | --- | --- |
| `list_projects` | None | `projects` with IDs and root directories |
| `list_processes` | `projectId` | `processes` with IDs, commands, arguments, and working directories |
| `process_status` | `projectId`, `processId` | `exists` and optional `runtime` for the current or most recent start |
| `process_logs` | `projectId`, `processId`, optional `limit`, `afterSequence` | `events` and `truncated` |
| `process_start` | `projectId`, `processId` | Runtime after starting the registered process |
| `process_stop` | `projectId`, `processId` | Runtime after stopping the owned process tree |
| `process_restart` | `projectId`, `processId` | Runtime after restarting the registered process |

All tools return structured JSON and a text JSON fallback. `process_logs` defaults to 100 events and accepts limits from 1 to 1000. A zero or omitted cursor selects the newest events; a positive cursor returns events after that sequence. Events contain `sequence`, `capturedAt`, `stream`, and text `data`. `truncated` reports output discarded by the daemon's retention limits. A status with `exists: false` means the definition has no recorded runtime.

Definition results omit environment overrides. Commands, arguments, and process output remain visible to the connected MCP host; captured output is untrusted data and is not automatically redacted. Lifecycle tools can launch project code or interrupt work and are annotated accordingly for the host's approval policy.

Tool discovery works without a running daemon. Operational failures return MCP tool errors with codes such as `connection_error`, `operation_failed`, `incompatible_version`, `daemon_error`, `timeout`, and `canceled`. Each daemon call waits at most 30 seconds. The adapter does not retry mutations. After a canceled or timed-out lifecycle call, inspect status before deciding whether to retry: the daemon may already have accepted the operation.

The adapter never launches the daemon or writes the registry directly. Closing the MCP session leaves the daemon and its managed processes running. Standard output is reserved for MCP messages, diagnostics go to standard error, and `--json` is not supported. The transport and tool protocol use the [official Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk).

## TUI

Build the colocated development binaries and open the interactive terminal client:

```console
just tui
```

On native Windows, if the default local daemon is unavailable, press `s` to start it explicitly. The daemon starts as an independent background process and remains available after the TUI exits. Opening the TUI never starts it automatically. A TUI using a custom endpoint, including the Linux container path, remains connection-only and offers `l` to retry after its daemon is started separately.

The TUI navigates existing project and process definitions, displays current or most recent runtime state and bounded output, and sends start, stop, and restart operations through the same local daemon protocol as the CLI. Use CLI commands to add or remove definitions.

Key bindings:

- `Up`/`k`, `Down`/`j`: select a process.
- `s`: start the selected inactive process, or start the unavailable default Windows daemon when no process data is available.
- `x`: stop the selected active process.
- `r`: restart the selected active process.
- `l`: reload definitions and process state.
- `Page Up`, `Page Down`: scroll output.
- `d`: show or hide runtime and diagnostic details.
- `?`: show or hide expanded help.
- `q` or `Ctrl+C`: quit.

The TUI requires interactive input and output. Use the non-interactive CLI, including `--json`, when streams are redirected or when invoking Dovik from automation.
