# Why

Local development processes are currently started from unrelated terminals, IDEs, scripts, and coding agents. This makes project ownership, duplicate instances, port conflicts, orphan cleanup, current status, and output difficult to understand and control. Dovik needs one local authority that can identify and supervise the processes owned by each project without expanding into general environment orchestration.

# What changes

Establish the first complete local process-supervision contract for Dovik:

- a daemon-owned registry of projects and process definitions;
- controlled start, stop, restart, status, and cleanup for one active instance per definition;
- captured stdout and stderr through a bounded in-memory event buffer;
- a versioned JSON control protocol over user-local Windows Named Pipes or Linux Unix Domain Sockets;
- a CLI for registry, lifecycle, status, and log operations;
- native Windows execution and an isolated Linux-in-Docker runtime;
- an idiomatic Go development, build, test, and container workflow.

These capabilities form one coherent outcome because registry ownership, process identity, lifecycle control, output, and local IPC must agree on the same process instance and state model.

The target version is SemVer `1.0.0` in soak. This change does not constitute a public stable release.

# Out of scope

- GUI and MCP clients;
- network distribution, remote execution, TCP control, or public ports;
- DNS, HTTPS, domains, reverse proxying, or routing;
- schedulers, automatic restarts, general orchestration, or container orchestration;
- a plugin system;
- a required Cadder dependency or a detailed Cadder integration contract.
