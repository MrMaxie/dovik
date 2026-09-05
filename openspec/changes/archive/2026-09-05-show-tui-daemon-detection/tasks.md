# Tasks

- [x] Add one textual daemon-detection presentation mapping for checking, detected, starting, and not-detected registry states.
- [x] Render the indicator at the right edge of the persistent TUI header while preserving the product identity and supported terminal widths.
- [x] Reuse the existing semantic styles without making color the only carrier of daemon state.
- [x] Add deterministic presentation tests for every label, alignment, truncation, and color-disabled output.
- [x] Verify unavailable, starting, and detected states through `just tui-web`, including separation from the browser toolbar's TUI-child status.
- [x] Run `just check` and review the rendered result against the annotated target.
- [x] Review `release.md` against actual delivery while retaining the `1.0.0` soak version.

## Verification

Implemented and verified on 2026-09-05; ready for archival.

- Deterministic TUI presentation tests cover checking, online empty, online populated, starting, and offline labels at the minimum supported width with color disabled.
- Built-in browser verification showed `• daemon offline` inside the TUI header, then `• daemon online` after the isolated daemon became ready, with the header background preserved across the full row.
- The browser harness toolbar remained unchanged and continued to identify only the TUI child process.
- `just check` passed, including Go tests, build, real ConPTY browser harness tests, and strict OpenSpec validation.
- `release.md` matches delivery; version remains in the 1.0.0 soak.
