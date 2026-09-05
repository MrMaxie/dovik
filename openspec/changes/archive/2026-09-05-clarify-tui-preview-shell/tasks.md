# Tasks

- [x] 1. Reduce the toolbar to one preview identity and explicitly TUI-scoped status/restart controls; move the child PID into diagnostic detail.
- [x] 2. Add a keyboard-accessible diagnostics disclosure, collapsed by default, preserving bounded history, live delivery, and clear-log behavior.
- [x] 3. Refit the terminal and propagate PTY size changes when toggling diagnostics; preserve the TUI process and interaction state.
- [x] 4. Verify restart affects only the TUI child and shell liveness labels do not represent managed-process or daemon state.
- [x] 5. Through `just tui-web`, inspect screenshots at 1270x986 and a narrower viewport with diagnostics collapsed and expanded; exercise keyboard disclosure, terminal focus, resize, restart, and live diagnostic delivery.
- [x] 6. Extend harness behavioral tests for diagnostic disclosure/history and resize as appropriate; run `just check`.
- [x] 7. Review `release.md` against actual delivery before archival; retain the 1.0.0 soak version.

## Verification

Implemented and verified on 2026-09-05; ready for archival.

- `just check` passed, including browser event-wiring tests for disclosure, bounded hidden history, resize-only toggling, clear logs, and TUI-specific status/restart.
- Built-in browser screenshots covered 1270x986 and narrower viewports, collapsed and expanded diagnostics, keyboard activation and focus, live retained diagnostics, and PTY resize.
- Toggling diagnostics retained the child PID and TUI state. Restarting the preview replaced its TUI child while the fixture-managed process remained running.
- Startup connection failure remained visible inside the terminal while the shell correctly reported that the TUI was running.
- `release.md` matches delivery; version remains in the 1.0.0 soak.
