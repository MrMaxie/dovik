# Tasks

- [x] Compare current Go PTY and ConPTY libraries, node-pty, xterm.js, and existing browser terminal projects against Windows correctness, maintenance burden, and deterministic agent operation.
- [x] Record the development-only dependency and runtime boundary, exact versions, licenses, transport, diagnostic channel, and cleanup model.
- [x] Add the isolated npm harness package, loopback HTTP and WebSocket server, xterm.js frontend, and PTY lifecycle management.
- [x] Add the build-tagged TUI diagnostic side channel without changing normal runtime behavior.
- [x] Add the pinned Node toolchain and `just tui-web` entrypoint.
- [x] Update agent guidance for browser-based TUI verification and live diagnostics.
- [x] Add automated harness coverage for input, resize, Ctrl+C, exit, diagnostics, restart, and disconnect cleanup.
- [x] Run repository gates and verify the actual TUI visually through the browser harness.
