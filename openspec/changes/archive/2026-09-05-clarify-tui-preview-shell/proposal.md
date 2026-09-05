# Why

The browser harness repeats Dovik branding and presents "Running", a PID, and "Restart process" above the actual terminal. These refer to the TUI child process but can be mistaken for a managed project's lifecycle controls. Persistent diagnostics also compete with the terminal during ordinary visual inspection.

The user approved this direction on 2026-09-05. Implementation and verification are complete as of 2026-09-05; archival remains outstanding.

# What changes

- Replace competing branding with a compact "TUI preview" toolbar.
- Explicitly identify child status and restart as belonging to the TUI.
- Move the child PID into diagnostic detail.
- Keep live diagnostics accessible in a collapsible panel, collapsed by default, without losing the live diagnostic stream while hidden.
- Preserve real PTY behavior and adjust terminal dimensions when diagnostics are expanded or collapsed.

This development-only outcome can be implemented and accepted independently of `clarify-tui-state-and-hierarchy`.

# Out of scope

- Terminal application layout, registry states, and lifecycle behavior.
- A browser product interface or web-based process management.
- New dependencies, transport changes, or distribution changes.
- Version bumps, publication, and releases.
