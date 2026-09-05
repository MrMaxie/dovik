# Tasks

- [ ] 1. Model and render loading, successful empty, unavailable, and populated registry outcomes explicitly; prevent connection failures from appearing as successful emptiness.
- [ ] 2. Add the single empty state and contextual setup help with commands verified against the CLI; expose refresh, help, and quit without irrelevant process hints.
- [ ] 3. Preserve prior selection/output on refresh failure, disclose diagnostic details without requiring a registered process, and verify retry recovery.
- [ ] 4. Reduce populated-workspace framing and filled surfaces; establish identity/state/action hierarchy, ordinary heading case, and focused accent use while preserving keyboard and daemon semantics.
- [ ] 5. Add behavioral regression coverage for startup failure versus empty success, retry success, populated refresh failure, setup help, and contextual shortcuts.
- [ ] 6. Through `just tui-web`, inspect and capture empty, loading, unavailable, populated, pending, and failed states at wide, compact, minimum, and below-minimum terminal sizes as applicable; verify navigation, output scrolling, resize preservation, contrast, and no-color meaning. Use isolated fixture definitions and clean up only those fixtures.
- [ ] 7. Run `just check`.
- [ ] 8. Review `release.md` against the actual delivered result before archival; retain the 1.0.0 soak version.
