# Tasks

- [ ] Confirm that `establish-local-process-supervision` is implemented, verified, and stable enough to provide the protocol and initial CLI prerequisite for this change.
- [ ] Extract or complete one shared Go protocol client used by every CLI command and by the TUI without exposing registry persistence or process adapters.
- [ ] Complete the non-interactive CLI command surface and add deterministic human-readable output, global `--json` output, structured errors, exit statuses, and redaction tests.
- [ ] Evaluate the current stable Bubble Tea v2 ecosystem against the dependency boundary, record the selected exact versions and licenses, and add approved dependencies through `go get`.
- [ ] Implement `dovik tui` with the project and process navigator, selected state, output pane, contextual help, and on-demand runtime details.
- [ ] Implement serialized start, stop, and restart actions with pending state, result refresh, retry behavior, and stale-response protection.
- [ ] Implement bounded periodic status and log refresh with sequence continuity, stdout and stderr attribution, truncation notice, scrolling, and selection preservation.
- [ ] Implement keyboard-only operation, non-color state cues, terminal resizing, minimum-size handling, non-interactive rejection, and terminal restoration.
- [ ] Add deterministic CLI and TUI model tests using fake protocol clients, controlled refresh ticks, and functional text assertions rather than color-dependent full-screen snapshots.
- [ ] Add native Windows Named Pipe smoke coverage for representative CLI and TUI lifecycle and output flows.
- [ ] Add Linux container smoke coverage that executes the CLI and TUI through the container-local Unix Domain Socket without publishing a port.
- [ ] Add the smallest required `just` development command and update user documentation for CLI automation, JSON mode, TUI launch, key bindings, and the registry-configuration boundary.
- [ ] Run `just check`, the focused Windows smoke tests, and the Linux container integration suite when Docker is available.
- [ ] Review `release.md` against the implemented and verified client behavior before archival.
