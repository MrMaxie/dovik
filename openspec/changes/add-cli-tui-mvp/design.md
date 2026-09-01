# Approach

## Sequencing

`establish-local-process-supervision` remains the source of the planned registry, lifecycle, output, and local protocol contracts. This change consumes those contracts and does not redefine daemon behavior. Implementation starts only after the protocol request and response shapes, transport security, and initial CLI operations are working and stable enough for a second client.

## One executable, two operator modes

The existing `dovik` executable remains the distribution unit. Ordinary subcommands provide the non-interactive CLI. `dovik tui` starts the interactive terminal client. Starting the TUI requires an interactive terminal; redirected or non-interactive use receives a clear error and keeps the CLI available.

Both modes depend on one Go protocol client that owns transport connection, version negotiation, request serialization, response decoding, and mapping structured daemon errors into client errors. CLI commands and TUI updates call that client. They do not access registry files, create operating-system processes, or reproduce lifecycle rules.

## CLI MVP

The CLI remains the complete client surface. It covers project and process definition registration and removal, listing, status, start, stop, restart, and bounded log tailing as established by the predecessor change.

Human-readable output is concise and stable enough for operators but is not a machine contract. A global `--json` mode emits one JSON document for every command, uses the protocol's public response fields, omits ANSI styling and prompts, and preserves the daemon's environment-value redaction. Successful JSON goes to stdout with exit status zero. A failed command writes one structured JSON error to stderr and exits non-zero.

The CLI does not prompt interactively in the MVP. Required values are supplied through arguments or flags so commands remain usable by scripts and coding agents.

## TUI MVP

The TUI is optimized for the repeated operator loop rather than configuration. Its primary view contains:

- a project and process navigator;
- the selected process state and available lifecycle actions;
- an ordered stdout and stderr output pane for the current or most recent runtime;
- a compact help line showing the active keyboard commands.

Extended runtime metadata, including instance ID, process ID, timestamps, exit code, and termination reason, is shown on demand rather than occupying the default view. Environment override values are never requested or rendered.

The first TUI supports keyboard navigation, start, stop, restart, refresh, details, log scrolling, help, and quit. Registry mutations remain in the CLI. Actions operate on stable project and process IDs, display an in-progress state while a request is outstanding, serialize mutations for the selected process, and refresh state after the daemon replies.

The TUI periodically polls status and sequence-aware log tails through the request-response protocol. It does not require protocol streaming. Polling is bounded, only one refresh per selected process may be in flight, and changing selection cancels or ignores obsolete results. The implementation default should make external state visible within two seconds under normal local operation without busy polling.

Output preserves daemon sequence order and distinguishes stdout from stderr with text labels or symbols that do not depend on color. When the daemon reports truncation, the output pane shows a persistent notice until a different runtime is selected. Resizing recomputes the layout without losing selection or scroll state. A terminal too small for the functional layout shows the required minimum size and retains quit and help controls.

Connection and operation failures are presented in the affected view with a useful retry action. Protocol or transport diagnostics remain behind an explicit details action. The TUI does not print raw internal errors over the primary interface or exit without restoring terminal state.

## TUI dependency boundary

Use a focused Go-native TUI library rather than web technologies or a custom terminal renderer. Bubble Tea v2 is the preferred candidate, but the implementation task must verify the current stable version, license, Go compatibility, Windows behavior, and supporting component versions before adding it through `go get`. The dependency decision must not add Node, npm, Rust, a browser engine, or a network listener.

Keep protocol access, application state, rendering, and key bindings in separate packages or modules so state transitions can be tested without a real terminal and daemon integration can be tested without snapshotting platform-specific escape sequences.

## Verification

CLI tests cover arguments, output streams, JSON schemas, exit statuses, redaction, and transport and daemon errors. TUI model tests use a fake protocol client and deterministic ticks to cover loading, refresh, selection changes, lifecycle actions, stale responses, truncation, failures, resizing, and terminal cleanup.

Integration tests run both modes against the real local daemon contract. Native Windows smoke tests exercise the Named Pipe. Linux smoke tests execute `dovik` inside the daemon container through the Unix Domain Socket without publishing a port. TUI smoke uses a pseudo-terminal and verifies functional text and state rather than exact color or complete-screen snapshots.

# Trade-offs

- One executable is simpler to distribute and keeps client versioning aligned, but the TUI dependency increases the `dovik` binary even for CLI-only users.
- Keeping registry mutation in the CLI makes the TUI smaller and avoids exposing commands and environment input in forms, but initial setup remains command-driven.
- Polling works with the existing request-response protocol and avoids a streaming extension, at the cost of slightly delayed updates and repeated local requests.
- A Go-native TUI preserves the current toolchain and local IPC boundary, but provides less visual flexibility than a browser or windowed GUI.
- Machine-readable JSON creates a durable automation surface, so its field shape and error behavior require compatibility tests in addition to human-output tests.
