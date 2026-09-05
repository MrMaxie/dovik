# Tasks

- [x] Confirm `clarify-tui-state-and-hierarchy` is archived before changing its unavailable-state behavior; preserve the independent `clarify-tui-preview-shell` closeout.
- [x] Extract a testable daemon-launch boundary beside the shared protocol client without granting the TUI managed-process authority.
- [x] Implement native Windows sibling executable resolution and detached peer launch with no inherited terminal streams, TUI cancellation, waiting, or cleanup ownership.
- [x] Update repeatable native development and browser-harness startup to provide colocated temporary `dovik` and `dovikd` binaries without adding a runtime dependency.
- [x] Add the contextual explicit daemon-start action, pending state, and applicable unavailable-state help while preserving retry, diagnostics, quit, and existing process key meanings.
- [x] Add bounded protocol readiness, duplicate-action prevention, compatible concurrent-launch handling, and concise failure recovery.
- [x] Keep custom endpoints and the Linux container connection-only; preserve `dovikd` as the container PID 1 entrypoint.
- [x] Add deterministic model and launcher tests for eligibility, pending state, launch failure, readiness timeout, concurrent availability, and retry recovery.
- [x] Add isolated native Windows and real ConPTY browser verification proving that TUI exit and restart preserve the daemon, a replacement TUI reconnects, and cleanup touches only fixture resources.
- [x] Update user and development documentation for explicit native daemon startup and the remaining connection-only paths.
- [x] Run `just check` and the focused native Windows lifecycle verification; run the Linux integration suite when Docker is available.
- [x] Review `release.md` against what was actually delivered while retaining the `1.0.0` soak version.

## Verification

Implemented and verified on 2026-09-05; ready for archival.

- `just check` passed, including deterministic TUI state tests, the native Windows detached-lifetime test, and the real ConPTY browser harness lifecycle test.
- `just test-integration` passed against Docker, preserving the Linux daemon entrypoint and container boundary.
- Built-in browser verification started the isolated daemon through the TUI, observed protocol readiness, restarted the TUI with a different client PID, and reconnected without a second daemon launch.
- Closing or restarting the isolated TUI left the fixture daemon alive; harness shutdown remained the sole fixture cleanup owner.
- The user's default endpoint and registry were not used by the isolated harness.
- `release.md` matches delivery; version remains in the 1.0.0 soak.
