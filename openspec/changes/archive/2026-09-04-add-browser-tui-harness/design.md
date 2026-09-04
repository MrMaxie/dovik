# Approach

## Technology decision

The harness uses xterm.js 6.0.0 in the browser, node-pty 1.1.0 for PTY and ConPTY hosting, `ws` 8.21.3 for the loopback WebSocket, and the xterm fit addon 0.11.0. All four dependencies are MIT licensed and live in a private development-only npm package under `devtools/tui-harness`.

`node-pty` is selected because it provides the same mature ConPTY-backed process boundary used by Visual Studio Code and exposes direct input, output, resize, PID, exit, and kill operations on Windows and Unix. The Go alternative would combine xterm.js with an HTTP and WebSocket server plus `charmbracelet/x/xpty`; that keeps the server in Go but relies on an experimental package without compatibility guarantees and requires more custom lifecycle and transport code. Existing servers such as WeTTY and ttyd target general shell sharing, add configuration and session behavior not needed here, and do not provide Dovik's separate diagnostic channel. ttyd also documents native Windows PTY compatibility limitations.

Node 24.18.0 is pinned through mise. npm is used only for the nested harness package. The Go module, production binaries, daemon protocol, and runtime image do not import or package any web or Node dependency.

## Terminal path

`just tui-web` installs the exact locked harness dependencies and starts one loopback HTTP server. The server builds the current `cmd/dovik` with the `dovik_dev_harness` build tag into an operating-system temporary directory, then spawns that binary with `tui` through node-pty in the repository working directory. The child inherits the current environment with `TERM=xterm-256color` and `COLORTERM=truecolor` added. The harness removes an inherited `NO_COLOR` value from the child because this dedicated visual-debugging surface must expose the TUI's color output even when the parent agent shell suppresses color in command output.

The browser uses xterm.js without WebGL so rendering stays stable for screenshots. xterm input is forwarded without key interpretation. Terminal output is written directly into xterm.js, preserving ANSI, true color, raw input, alternate screen, application cursor keys, and mouse sequences enabled by the TUI. The fit addon computes terminal rows and columns from the visible panel; its debounced resize event resizes the backing PTY, not only the DOM.

## Diagnostic channel

The development build tag activates a small TUI diagnostic writer. It connects to a harness-owned Windows Named Pipe or Unix Domain Socket named in `DOVIK_TUI_DEV_LOG_PIPE` and writes newline-delimited JSON records. Normal builds compile a no-op implementation and do not connect, listen, or log.

The harness forwards these records to a distinct browser log panel together with process lifecycle and PTY resize events. Diagnostic records avoid environment values and arbitrary user input. They are bounded in memory and disappear with the harness.

## Lifecycle and security

The HTTP listener binds only to `127.0.0.1`, checks the WebSocket Origin, serves a fixed route allowlist, disables caching, and applies a restrictive Content Security Policy. One PTY process belongs to one connected browser tab.

Closing the tab, restarting the process, pressing Ctrl+C in the harness terminal, or interrupting `just tui-web` begins with the terminal's native Ctrl+C input. If the TUI does not exit promptly, the harness terminates the exact child process tree and closes the PTY. Shutdown closes WebSockets, the diagnostic listener, the HTTP server, and the temporary build directory.

## Verification

Automated tests cover loopback serving, process startup, raw input, PTY resizing, diagnostic separation, restart, exit reporting, and browser disconnect cleanup. Live browser verification exercises the actual TUI, special keys, Ctrl+C, a viewport resize, live diagnostics, and screenshots the final layout.
