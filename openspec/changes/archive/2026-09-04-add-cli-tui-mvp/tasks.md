# Tasks

- [x] Confirm that `establish-local-process-supervision` is implemented, verified, and stable enough to provide the protocol and initial CLI prerequisite for this change.
- [x] Extract or complete one shared Go protocol client used by every CLI command and by the TUI without exposing registry persistence or process adapters.
- [x] Complete the non-interactive CLI command surface and add deterministic human-readable output, global `--json` output, structured errors, exit statuses, and redaction tests.
- [x] Evaluate the current stable Bubble Tea v2 ecosystem against the dependency boundary, record the selected exact versions and licenses, and add approved dependencies through `go get`.
- [x] Implement `dovik tui` with the project and process navigator, selected state, output pane, contextual help, and on-demand runtime details.
- [x] Implement serialized start, stop, and restart actions with pending state, result refresh, retry behavior, and stale-response protection.
- [x] Implement bounded periodic status and log refresh with sequence continuity, stdout and stderr attribution, truncation notice, scrolling, and selection preservation.
- [x] Implement keyboard-only operation, non-color state cues, terminal resizing, minimum-size handling, non-interactive rejection, and terminal restoration.
- [x] Add deterministic CLI and TUI model tests using fake protocol clients, controlled refresh ticks, and functional text assertions rather than color-dependent full-screen snapshots.
- [x] Add native Windows Named Pipe smoke coverage for representative CLI and TUI lifecycle and output flows.
- [x] Add Linux container smoke coverage that executes the CLI and TUI through the container-local Unix Domain Socket without publishing a port.
- [x] Add the smallest required `just` development command and update user documentation for CLI automation, JSON mode, TUI launch, key bindings, and the registry-configuration boundary.
- [x] Run `just check`, the focused Windows smoke tests, and the Linux container integration suite when Docker is available.
- [x] Review `release.md` against the implemented and verified client behavior before archival.
