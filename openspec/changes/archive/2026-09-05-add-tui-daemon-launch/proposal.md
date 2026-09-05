# Why

Opening the TUI currently ends at an unavailable-data screen when the local daemon is not already running. The operator must leave the TUI, start `dovikd` in another terminal, and then retry. Native Windows should provide a complete local entry path without turning the TUI into the daemon's owner or weakening daemon authority over managed processes.

The daemon started from the TUI must have an independent peer lifetime. Closing, restarting, or failing the TUI must not stop the daemon, and the daemon must not inherit the TUI terminal or depend on its context, input, output, or cleanup.

This change extends the unavailable state defined by `clarify-tui-state-and-hierarchy`. That change must be archived before implementation begins so the new delta is applied to one accepted TUI state contract.

The user approved this direction on 2026-09-05. Implementation and verification are complete as of 2026-09-05; archival remains outstanding.

# What changes

- add an explicit native Windows action for starting the default local daemon when the TUI cannot connect;
- launch the colocated `dovikd` executable as a detached peer process whose lifetime is independent of the TUI;
- keep daemon startup separate from the shared protocol client and preserve the daemon as the sole authority over registry and managed-process lifecycle operations;
- show bounded startup progress, reconnect through the existing versioned Named Pipe protocol, and handle concurrent launch attempts without treating the losing launch as a user-visible failure when a compatible daemon becomes available;
- retain actionable retry, quit, and diagnostic behavior when launch fails or the daemon does not become ready;
- keep custom endpoints and the Linux container runtime connection-only, because the TUI cannot safely infer how those daemons should be started;
- make `just tui` and `just tui-web` able to exercise the colocated daemon launch through isolated development artifacts and verification state;
- verify that restarting or closing the TUI does not terminate the daemon and that development cleanup removes only isolated fixture resources.

The target remains SemVer `1.0.0` in soak. This change does not authorize a version bump, tag, release, commit, push, or publication.

# Out of scope

- automatic daemon startup when the TUI opens;
- daemon stop, restart, shutdown, health-management, or service-installation controls in the TUI;
- treating the daemon as a managed project process or adding a client-side managed-process lifecycle;
- native Linux daemonization, host process control from a container, or changing the Linux image entrypoint;
- starting a daemon for a custom `--endpoint`;
- TCP, HTTP, remote control, scheduling, automatic managed-process restart, or Cadder integration;
- persistent daemon logs, a browser product surface, packaging changes, version changes, or release publication.
