# Tasks

- [ ] 1. Reduce the toolbar to one preview identity and explicitly TUI-scoped status/restart controls; move the child PID into diagnostic detail.
- [ ] 2. Add a keyboard-accessible diagnostics disclosure, collapsed by default, preserving bounded history, live delivery, and clear-log behavior.
- [ ] 3. Refit the terminal and propagate PTY size changes when toggling diagnostics; preserve the TUI process and interaction state.
- [ ] 4. Verify restart affects only the TUI child and shell liveness labels do not represent managed-process or daemon state.
- [ ] 5. Through `just tui-web`, inspect screenshots at 1270x986 and a narrower viewport with diagnostics collapsed and expanded; exercise keyboard disclosure, terminal focus, resize, restart, and live diagnostic delivery.
- [ ] 6. Extend harness behavioral tests for diagnostic disclosure/history and resize as appropriate; run `just check`.
- [ ] 7. Review `release.md` against actual delivery before archival; retain the 1.0.0 soak version.
