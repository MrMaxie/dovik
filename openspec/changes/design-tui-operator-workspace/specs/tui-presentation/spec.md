## ADDED Requirements

### Requirement: Coherent TUI visual system

The Dovik TUI SHALL use one fixed semantic presentation system for color, spacing, borders, emphasis, selection, process state, actions, notices, and output. Presentation SHALL adapt to the terminal color profile and SHALL remain understandable when color is disabled.

#### Scenario: Render with true color

- **WHEN** the TUI runs in a terminal that reports true-color support
- **THEN** it renders the Dovik palette consistently across the header, navigator, process summary, output, notices, and shortcut bar
- **AND** selected foreground and background pairs meet a 4.5:1 contrast ratio for ordinary text

#### Scenario: Render without color

- **WHEN** color output is disabled or unavailable
- **THEN** every selection, lifecycle state, stream, pending operation, notice, and failure remains identifiable through text, markers, borders, or emphasis
- **AND** the TUI emits no required meaning exclusively through color

### Requirement: Responsive operator workspace

The TUI SHALL organize process navigation, selected-process state and actions, bounded output, notices, and contextual shortcuts into stable regions that adapt without losing operator state.

#### Scenario: Use a wide terminal

- **WHEN** the terminal is at least 96 columns wide and 24 rows high
- **THEN** the TUI shows a persistent header, a left process navigator, a selected-process summary above the output work area, and a one-line contextual shortcut bar
- **AND** resizing within the wide layout does not change the selected process or output scroll position

#### Scenario: Use a compact terminal

- **WHEN** the terminal is at least 64 columns wide and 18 rows high but does not meet the wide-layout threshold
- **THEN** the TUI stacks navigation, selected-process summary, and output while retaining the selected process, its state, the applicable action, output access, help, and quit
- **AND** crossing between compact and wide layouts preserves selection, details visibility, help visibility, and output scroll position

#### Scenario: Use a terminal below the functional minimum

- **WHEN** the terminal is smaller than 64 columns or 18 rows
- **THEN** the TUI shows the required minimum dimensions and retains working help and quit controls
- **AND** restores the operator workspace when sufficient space returns

### Requirement: Immediate process-state hierarchy

The selected process, its lifecycle state, available action, pending work, and relevant recovery message SHALL be visually prominent without obscuring project and process identity or current output.

#### Scenario: Inspect a selected process

- **WHEN** registry and runtime data are available
- **THEN** the navigator marks the selected project and process using both a marker and visual emphasis
- **AND** the process summary shows the state label and only the lifecycle actions applicable to that state

#### Scenario: Observe a pending or failed action

- **WHEN** a lifecycle request is pending or fails
- **THEN** the process summary shows what is pending or what did not happen
- **AND** exposes the available retry or recovery action without replacing the current selection or output
- **AND** keeps raw protocol or transport diagnostics behind the existing details action

### Requirement: Readable bounded output

The output region SHALL prioritize recent ordered events, preserve sequence and truncation behavior, and distinguish stdout from stderr without relying only on color.

#### Scenario: Read mixed process output

- **WHEN** the selected runtime has both stdout and stderr events
- **THEN** the TUI renders them in daemon sequence order with persistent textual stream attribution
- **AND** styling may reinforce but does not replace that attribution

#### Scenario: Older output is unavailable

- **WHEN** the daemon reports truncated output
- **THEN** the output region shows a persistent truncation notice that is visually distinct from process output
- **AND** continues to show the available ordered tail

### Requirement: Preserved keyboard and daemon semantics

The visual redesign SHALL preserve all existing keyboard operations, action availability, stable selection, polling, stale-response handling, daemon authority, error disclosure, and terminal cleanup behavior.

#### Scenario: Operate the redesigned TUI

- **WHEN** a user navigates, starts, stops, restarts, refreshes, opens details or help, scrolls output, or quits
- **THEN** the same keys invoke the same model transitions and shared-client requests as before the visual change
- **AND** no presentation component reads registry state directly or performs a lifecycle operation

### Requirement: Browser-based visual acceptance

Visual TUI changes SHALL be verified through the development browser harness in addition to deterministic model and presentation tests.

#### Scenario: Accept the redesigned workspace

- **WHEN** implementation is ready for review
- **THEN** a developer or coding agent inspects screenshots of populated, empty, unavailable, pending, failed, compact, minimum-size, and below-minimum states rendered by `just tui-web`
- **AND** exercises keyboard navigation and browser-driven PTY resize
- **AND** confirms through live diagnostics that the visible TUI state matches the application state
