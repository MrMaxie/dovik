# Add an MCP operator client

**Status:** Archived
**Completed:** 2026-09-06

## Why

Agent clients need a discoverable tool interface to the existing local supervisor. The daemon already owns the registry, runtime state, output, and lifecycle operations used by the CLI and TUI.

## What Changes

- Add `dovik mcp`, a local MCP server over standard input and output.
- Expose project and process lists, status, bounded logs, and start, stop, and restart for registered processes through `operatorclient.Client`.
- Document client configuration and verify the MCP session, local IPC, and daemon ownership on Windows and Linux in Docker.

## Scope

No registry mutation tools, arbitrary command execution, daemon auto-start, remote transport, network listener, credential storage, Cadder integration, or TUI changes. Existing CLI and daemon contracts remain compatible. Version `1.0.0` stays in soak.

## Impact

- New `mcp-operator-client` capability and official Go MCP SDK dependency.
- CLI dispatch and operator documentation.
- Protocol and lifecycle tests using isolated daemon fixtures.

## Archive Information

The change was archived with all eight implementation tasks complete and the quality gates passing. Three requirements were merged into `openspec/specs/mcp-operator-client/spec.md`.
