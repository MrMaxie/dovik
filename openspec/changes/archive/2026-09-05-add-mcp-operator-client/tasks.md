## 1. MCP adapter

- [x] 1.1 Add the pinned official Go MCP SDK and seven typed daemon-backed tools.
- [x] 1.2 Validate references and log bounds, classify failures, redact definition views, and bound calls by cancellation and timeout.
- [x] 1.3 Add `dovik mcp` stdio dispatch with protocol-only stdout and no daemon launch.

## 2. Verification

- [x] 2.1 Test MCP initialization, discovery, schemas, structured results, invalid requests, errors, and cancellation.
- [x] 2.2 Test the real stdio command through local IPC, lifecycle operations, logs, and adapter exit while a process remains managed by the daemon.
- [x] 2.3 Run native checks, Docker tests, Linux integration, and strict OpenSpec validation.

## 3. Handoff

- [x] 3.1 Document setup, tool arguments, output, and lifecycle boundaries.
- [x] 3.2 Review scope and diff and record implementation status and verification.

## Verification results

- `just check`: passed Go vet, tests, builds, four browser harness tests, schema validation, and all eight strict OpenSpec validations.
- `just test-linux`: passed Go tests, vet, and builds in Docker, including MCP stdio lifecycle tests over a Unix Domain Socket.
- `just test-integration`: passed the existing Linux daemon, CLI, TUI, lifecycle, persistence, and container boundary suite.
- Native MCP stdio tests passed initialization, seven-tool discovery, real Named Pipe operations, output cursors and bounds, and independent daemon ownership after adapter exit.
- Adapter tests passed input rejection before IPC, definition environment omission, structured results, safe operational errors, and cancellation propagation.
- `git diff --check` and targeted Go formatting checks passed.
- MCP was verified with SDK client sessions and the built command. No desktop MCP host configuration was changed or manually verified.

Implementation and archival are complete. The accepted requirements are in `openspec/specs/mcp-operator-client/spec.md`; the target version remains `1.0.0` in soak.
