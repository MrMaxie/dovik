# Approach

Reuse the existing harness HTML, CSS, JavaScript, xterm renderer, and PTY bridge. Use one compact toolbar named "TUI preview". Label restart "Restart TUI" and qualify status with its subject, such as "TUI running" or "TUI exited". Do not imply daemon connectivity or managed-process health from child liveness.

Use an accessible disclosure control for diagnostics, with an explicit expanded state and keyboard operation. The panel starts collapsed and can be expanded for repeated diagnostic work. Keep the existing bounded diagnostic history and live delivery active while collapsed; reopening displays retained entries and follows new records. Keep the child PID and clear-log action within diagnostic detail.

Expansion/collapse must resize the backing PTY to match available terminal space without restarting the TUI, stealing terminal state, or losing selected process/output position. Keep a visible keyboard focus indicator for shell controls and make it possible to return focus to the terminal.

Preserve loopback binding, development-only dependencies, restart behavior, diagnostic/output separation, and cleanup.

# Trade-offs

A collapsed diagnostic panel prioritizes visual inspection while preserving the developer's live debugging workflow on demand. Removing diagnostics would weaken the established verification path, so they remain available beside or below the terminal when expanded.

A small change to existing markup and layout is sufficient; an additional UI framework would add no required capability.
