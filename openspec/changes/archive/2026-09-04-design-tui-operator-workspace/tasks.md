# Tasks

- [x] Confirm the current stable Lip Gloss v2 version, license, Go 1.27 compatibility, Windows color-profile behavior, and Bubble Tea v2 integration before adding it through `go get`.
- [x] Define Dovik presentation tokens for color, spacing, borders, state emphasis, stream attribution, and no-color fallback.
- [x] Split the existing view into testable header, navigator, process summary, output, notice, details, help, and shortcut regions without moving protocol or lifecycle logic out of the model.
- [x] Implement the wide and compact workspace layouts while preserving selection, scroll position, details, help, and pending actions across resize.
- [x] Apply semantic, non-color-only presentation for selection, every process state, lifecycle availability, pending work, notices, failures, stdout, stderr, and truncation.
- [x] Add focused tests for style semantics, required textual cues, color-disabled rendering, layout thresholds, and resize preservation.
- [x] Run `just check` and exercise navigation, actions, details, help, scrolling, Ctrl+C, and resize through `just tui-web`.
- [x] Capture and inspect browser screenshots for populated, empty, unavailable, pending, failed, compact, minimum-size, and below-minimum states.
- [x] Update concise agent or contributor guidance only if the implemented visual workflow differs from the existing `just tui-web` instruction.
