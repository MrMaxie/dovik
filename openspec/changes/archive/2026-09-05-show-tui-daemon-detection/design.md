# Approach

## Use the existing connection outcome

Render the indicator from the TUI model's current registry outcome rather than adding a second source of truth. `loading` maps to `• daemon checking`, a successful empty or populated registry maps to `• daemon online`, daemon launch pending maps to `• daemon starting`, and an unavailable registry maps to `• daemon offline`.

These labels describe the latest explicit protocol-backed registry result. They do not claim continuous health monitoring. Existing refresh and daemon-start transitions update the label through the same model state changes that already drive the main workspace.

## Keep the header hierarchy stable

Retain `dovik` and `Local processes` on the left. Place the daemon label at the right edge of the same one-line header at every supported terminal width. Preserve the header background across the full row. Allocate space to the status first, then truncate the descriptive title if necessary while keeping the product name and full status readable.

Use color only for the bullet: green for online, yellow for starting, red for offline, and muted for checking. Keep the adjacent text in the ordinary header foreground. The complete text remains present when color is disabled, so state is never communicated by color alone.

## Verify state meaning and layout

Add deterministic presentation coverage for every daemon label, right alignment, supported-width rendering, and color-disabled meaning. Use `just tui-web` to inspect the unavailable, starting, and detected empty-registry states in the actual browser terminal and confirm that the browser toolbar's TUI-child status remains visually and semantically separate.

# Trade-offs

- Basing the label on the latest registry outcome keeps it honest and requires no new protocol operation, but it is not a live daemon heartbeat.
- Reserving header width for the full status may truncate `Local processes` at narrow supported widths, while preserving the more actionable daemon state.
