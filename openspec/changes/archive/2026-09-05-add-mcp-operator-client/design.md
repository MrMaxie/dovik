# MCP operator client design

## Decisions

- Run the official Go MCP SDK in `dovik mcp` using stdio only. The SDK owns initialization, tool discovery, JSON-RPC framing, schema validation, and cancellation.
- Reuse the global `--endpoint` option and `control.NewClient`. Do not read registry files or launch the daemon or managed processes from the adapter.
- Register seven tools: `list_projects`, `list_processes`, `process_status`, `process_logs`, `process_start`, `process_stop`, and `process_restart`.
- Use explicit JSON result views. Exclude environment overrides from every definition response. Return captured output as text data with sequence, stream, timestamp, and truncation metadata. Process output is untrusted data and is not automatically redacted.
- Give each daemon call a 30-second deadline bounded by request cancellation. Never retry mutations automatically. Cancellation stops waiting; it cannot undo an operation already accepted by the daemon.
- Require nonempty project and process IDs. Logs default to 100 events and accept limits from 1 to 1000 plus an optional nonnegative sequence cursor.
- Mark inspection tools read-only and lifecycle tools potentially destructive, non-idempotent, and closed-world. Annotations guide hosts but are not authorization enforcement.
- Return daemon and connection failures as MCP tool errors with a stable code and a bounded, actionable message. Do not forward arbitrary transport or daemon diagnostic strings to the agent.
- Keep startup diagnostics on stderr and stdout exclusively for MCP messages. Discovery remains available without a running daemon; EOF or session cancellation ends the MCP adapter without stopping managed processes.

## Verification

Use SDK client sessions for schema, error, and cancellation coverage. Exercise the actual CLI stdio entry point against isolated local control servers and real managed fixture processes. Run the same tests in the Docker test stage, plus the existing Linux integration suite. Preserve unrelated assets and private workstation state.
