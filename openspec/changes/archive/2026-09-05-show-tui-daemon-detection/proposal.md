# Why

The TUI shows registry and process content, but its header does not state whether a compatible daemon was detected. In the empty-registry state, the large body correctly says that no processes are registered, while the unused right side of the header gives no persistent confirmation that this is a successful daemon response rather than an offline or stale screen.

The user requested a daemon indicator in the right side of the TUI header on 2026-09-05.

The user confirmed on 2026-09-05 that the indicator belongs inside the terminal TUI, not in the browser harness interface. Implementation and verification are complete; archival remains outstanding.

# What changes

- add a compact, right-aligned daemon-detection label to the persistent TUI header;
- derive the label from the latest registry connection outcome already held by the TUI model;
- distinguish checking, detected, starting, and not-detected states with explicit text, with color used only as reinforcement;
- keep the browser harness status separate because its existing `Running` label describes only the TUI child process;
- preserve the current registry, process, keyboard, polling, and daemon-launch behavior.

# Out of scope

- daemon health monitoring, a new heartbeat, or a new protocol endpoint;
- daemon PID, endpoint, transport, version, or other diagnostic detail in the default header;
- daemon stop, restart, shutdown, or service controls;
- changes to the browser harness toolbar;
- version bumps, tags, releases, or publication.
