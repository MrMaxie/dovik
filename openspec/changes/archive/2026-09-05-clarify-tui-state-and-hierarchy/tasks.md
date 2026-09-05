# Tasks

- [x] 1. Model and render loading, successful empty, unavailable, and populated registry outcomes explicitly; prevent connection failures from appearing as successful emptiness.
- [x] 2. Add the single empty state and contextual setup help with commands verified against the CLI; expose refresh, help, and quit without irrelevant process hints.
- [x] 3. Preserve prior selection/output on refresh failure, disclose diagnostic details without requiring a registered process, and verify retry recovery.
- [x] 4. Reduce populated-workspace framing and filled surfaces; establish identity/state/action hierarchy, ordinary heading case, and focused accent use while preserving keyboard and daemon semantics.
- [x] 5. Add behavioral regression coverage for startup failure versus empty success, retry success, populated refresh failure, setup help, and contextual shortcuts.
- [x] 6. Through `just tui-web`, inspect and capture empty, loading, unavailable, populated, pending, and failed states at wide, compact, minimum, and below-minimum terminal sizes as applicable; verify navigation, output scrolling, resize preservation, contrast, and no-color meaning. Use isolated fixture definitions and clean up only those fixtures.
- [x] 7. Run `just check`.
- [x] 8. Review `release.md` against the actual delivered result before archival; retain the 1.0.0 soak version.

## Verification

Implemented and verified on 2026-09-05; ready for archival.

- `just check` passed: Go vet, tests, build, real ConPTY harness tests, and strict OpenSpec validation.
- Browser screenshots covered loading, empty/setup help, startup unavailability, retained output after registry failure, populated running, pending start, and failed start.
- Browser-driven PTY sizes included 151 columns, 80x24, 64x18, and 55x13. Keyboard selection, output paging, help, diagnostic disclosure, and retry to an empty registry were exercised. The minimum-size setup help remained fully visible.
- Isolated fixture storage and its managed process were cleaned up. No user registry definitions were changed.
- No-color meaning and exact supported layout dimensions passed deterministic presentation tests. Calculated primary, muted, selection, and lifecycle text contrast exceeded 4.5:1.
- `release.md` matches delivery; version remains in the 1.0.0 soak.
