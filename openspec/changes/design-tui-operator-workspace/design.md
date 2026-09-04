# Approach

## Information hierarchy

The TUI becomes an operator workspace rather than a formatted text report. On terminals at least 96 columns wide and 24 rows high, it uses a persistent header, a bounded navigator on the left, and a work area on the right. The work area gives the selected process state and available action a compact summary above the output pane. A one-line contextual shortcut bar remains visible at the bottom.

Between the functional minimum of 64 by 18 and the wide threshold, the same information stacks vertically: a short navigator, selected-process summary, and output pane. The compact layout may reduce secondary metadata but must retain selection, state, the applicable action, output, help, and quit. Below the functional minimum, the existing small-terminal recovery view remains the only content.

Empty, loading, unavailable, pending, failed, and populated states use the same frame so data arrival or an error does not cause the main regions to jump unpredictably.

## Visual system

Use named presentation tokens rather than inline styling decisions. The initial true-color palette is:

- canvas `#0B1117` and surface `#111B24`;
- primary text `#E7EDF4` and muted text `#91A0B2`;
- accent `#75B8FF` for selection, focus, and active navigation;
- success `#7CD992` for running;
- attention `#E6C66A` for starting, stopping, and pending work;
- danger `#F08080` for failed operations and failed runtimes;
- border `#334252` and strong border `#5E7892`.

Color reinforces semantics but never carries them alone. Every process state keeps its text label. Selection uses a marker and emphasis in addition to color. stdout and stderr retain their textual stream labels even when their presentation differs. Pending and failure states include explicit wording and the available next action.

The selected true-color foreground and background pairs must meet a 4.5:1 contrast ratio for ordinary text. Styling must pass through the terminal's detected color profile so 256-color and 16-color terminals receive stable approximations. Normal `NO_COLOR` behavior remains respected by the TUI; the dedicated browser harness intentionally enables color for visual verification.

## Rendering boundary

Keep protocol access and state transitions in the existing model. Add a small presentation layer that derives semantic view data and renders named regions. Do not move daemon calls, polling, action serialization, or selection ownership into widgets.

Use the current stable `charm.land/lipgloss/v2` release if implementation confirms its Go 1.27 compatibility, MIT license, Windows behavior, color-profile handling, and fit with Bubble Tea v2. Add it through `go get` only when the rendering implementation begins. Do not hand-build ANSI escape sequences or add a general component framework.

## Interaction preservation

Existing keys remain authoritative: up/down and j/k change selection; s, x, and r request lifecycle actions; l refreshes; d discloses details; page keys scroll output; ? opens help; q and Ctrl+C quit. The visual change may reorganize where these actions are shown, but it must not change their meaning, availability, serialization, or daemon requests.

Focus and selection do not reset when the terminal crosses a layout threshold. Output order, stream attribution, truncation notices, and the current scroll position remain intact across resize.

## Verification

Model tests continue to prove state transitions and daemon interactions. Presentation tests cover semantic style selection, region visibility, layout thresholds, no-color output, and preservation of required text cues without asserting an entire platform-specific ANSI frame.

Visual acceptance uses `just tui-web` with populated, empty, pending, failed, unavailable, details, help, and too-small states. Capture and inspect screenshots at 120 by 34, 80 by 24, 64 by 18, and below-minimum dimensions. Exercise keyboard navigation and resize in the browser, and use the live diagnostic panel to confirm that visual state matches model events.
