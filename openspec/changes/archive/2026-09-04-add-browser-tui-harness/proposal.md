# Why

TUI changes currently have deterministic model and smoke coverage but no repeatable way for a coding agent to inspect the real terminal rendering and exercise it through a browser. Visual and interaction changes therefore depend on an operator opening a separate native terminal and correlating the screen with diagnostics manually.

# What changes

- add a development-only browser harness that runs the actual `dovik tui` process through a real PTY or Windows ConPTY;
- render that byte stream with xterm.js and forward raw input, special keys, terminal resizing, and supported mouse reporting over a loopback WebSocket;
- show a separate live diagnostic log beside the terminal without writing diagnostic records into the PTY output;
- add `just tui-web` as the single developer entrypoint and pin the required Node toolchain through mise;
- document browser-based visual verification as the standard agent workflow for TUI presentation and interaction changes.

This change does not add a browser surface, HTTP transport, or Node dependency to the Dovik runtime or distribution.

# Out of scope

- a product web interface or remote terminal;
- exposing the harness beyond loopback or adding authentication, HTTPS, sharing, or multi-user sessions;
- replacing Bubble Tea or emulating ANSI behavior outside xterm.js;
- changing daemon protocols, process supervision, registry state, or TUI product behavior;
- persistent diagnostic logging or diagnostic export.

## Implementation status

**Status:** Implementation Complete
**Completed:** 2026-09-04
