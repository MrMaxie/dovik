## MODIFIED Requirements

### Requirement: Persistent workspace tabs

The TUI SHALL present `Personas`, `Projects`, and `Processes` as persistent top-level tabs in dependency order. Each tab SHALL include its relevant collection count, with Processes reporting running and registered counts. The active tab SHALL remain identifiable without color, and left and right arrow keys SHALL switch tabs without discarding retained workspace state.

#### Scenario: Inspect workspace counts

- **WHEN** identity and registry state are available
- **THEN** the header shows `Personas (t)`, `Projects (z)`, and `Processes (x/y)` using current snapshot and runtime state
- **AND** navigator bodies do not repeat the same collection heading or position counter

#### Scenario: Switch between primary workspaces

- **WHEN** an operator presses the left or right arrow key
- **THEN** the active workspace moves to the adjacent tab with wraparound
- **AND** returning to a tab restores its previous entity selection and applicable output state

### Requirement: Consistent entity navigation and primary actions

Every top-level TUI workspace SHALL use the up and down arrow keys for selectable entity navigation, `l` for refresh, and Enter for the selected entity's primary action. Process project-group rows SHALL not be selectable. Actions and editors SHALL remain inside the current TUI and SHALL not mutate state before explicit confirmation.

#### Scenario: Navigate grouped processes

- **WHEN** registered processes span one or more projects
- **THEN** the process navigator renders one unselectable project row followed by tree-connected process rows
- **AND** up and down move only between processes

#### Scenario: Open the selected entity

- **WHEN** an operator presses Enter on a selected process, project, or persona
- **THEN** Dovik opens an in-place action or editing overlay without clearing the terminal
- **AND** Escape or Ctrl+C returns without applying a change

## ADDED Requirements

### Requirement: Contextual terminal controls

The TUI SHALL use one contextual shortcut convention with lower-case labels, direct key symbols, and muted dot separators. Long command values SHALL wrap within their region instead of being replaced by an ellipsis.

#### Scenario: Read workspace controls

- **WHEN** an operator opens any workspace or overlay
- **THEN** visible hints use the same key-label grammar, such as `↑ up  •  ↓ down`
- **AND** only actions available in the current context are advertised

### Requirement: In-place forms and confirmation

Project and persona editors SHALL present all applicable fields in one navigable form inside the main TUI. Entered values SHALL be visually distinct from labels and focus indicators, and submission SHALL open a separate confirmation modal before mutation.

#### Scenario: Edit a project

- **WHEN** an operator opens a project editor
- **THEN** Tab, Shift+Tab, up, and down can move among applicable fields without completing artificial sections
- **AND** the form does not prefix entered values with a synthetic `> ` prompt
- **AND** submitting the form opens a distinct confirmation modal

### Requirement: Full retained output viewer

The TUI SHALL keep the process body preview focused on the newest output and SHALL provide an in-place, scrollable full retained-output view with an optional shell-free follow mode. Safe SGR styling MAY be retained, but unsafe terminal control sequences SHALL not affect the surrounding interface.

#### Scenario: Open process output

- **WHEN** an operator opens full output for a selected process
- **THEN** retained stdout and stderr are available in daemon sequence order with textual stream attribution
- **AND** the operator can scroll without leaving the main TUI

#### Scenario: Follow a stopped process

- **WHEN** an operator enables follow mode for a process that is not running
- **THEN** existing retained output remains visible
- **AND** polling continues until the operator exits follow mode without starting a shell or process
