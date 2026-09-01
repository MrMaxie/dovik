# Why

The local supervisor needs two complementary operator surfaces after its control protocol is stable. Scripts and coding agents need a complete, deterministic CLI, while developers need a fast interactive view of process state and output without managing several terminal commands manually.

The CLI and TUI belong in one change because they must use the same local protocol client, expose consistent lifecycle results, and preserve the daemon as the only process authority. Delivering only the TUI would leave automation and registry setup incomplete; delivering only the CLI would leave the accepted interactive operator workflow unresolved.

This change builds on `establish-local-process-supervision`. Implementation SHALL begin only after that change provides a stable versioned protocol and working CLI command surface.

# What changes

- complete the `dovik` CLI MVP as the full non-interactive surface for project registry, process definitions, lifecycle, status, and bounded log retrieval;
- add deterministic human-readable output, machine-readable JSON output, and meaningful process exit statuses to the CLI;
- add `dovik tui` as a keyboard-driven interactive client for navigating registered projects and processes, inspecting current state, controlling lifecycle, and following bounded output;
- keep project and process definition mutations in the CLI for the MVP while the TUI focuses on repeated operational work;
- share one protocol client and error model between CLI and TUI without duplicating daemon rules in either presentation layer;
- support native Windows terminals and interactive execution inside the Linux container without adding TCP, HTTP, a browser runtime, or host-process access;
- verify command behavior, TUI state transitions, terminal rendering constraints, and real daemon integration on both initial runtimes.

The target remains SemVer `1.0.0` in soak. This change prepares an additional user-facing capability and does not authorize a version bump, tag, release, or publication.

# Out of scope

- a web interface or desktop windowed application;
- an MCP client;
- adding, editing, or removing project and process definitions from the TUI;
- editing or revealing environment override values in either client;
- remote control, TCP, HTTP, public ports, or browser access;
- persistent logs, log search, log export, or output from runtimes older than the current or most recent runtime;
- mouse-only controls, themes, plugins, dashboards, alerts, scheduling, or automatic restart policies;
- Cadder integration or endpoint allocation.
