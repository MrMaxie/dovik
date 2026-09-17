## ADDED Requirements

### Requirement: Persistent workspace tabs

The TUI SHALL present `Processes`, `Identity`, and `Personas` as persistent top-level tabs in every supported workspace. The active tab SHALL remain identifiable without color, and left and right arrow keys SHALL switch tabs without discarding the selection or state retained by another workspace.

#### Scenario: Switch between primary workspaces

- **WHEN** an operator presses the left or right arrow key at a supported terminal size
- **THEN** the active workspace moves to the adjacent tab with wraparound
- **AND** returning to a tab restores its previous navigator selection and applicable process output state

#### Scenario: Inspect identity and personas

- **WHEN** an operator opens Identity or Personas
- **THEN** both workspaces use the current identity snapshot through the existing operator client
- **AND** Personas shows public persona metadata and project assignments without credentials or secrets

## MODIFIED Requirements

### Requirement: Responsive operator workspace

The TUI SHALL present a state-specific workspace with persistent top-level tabs and a stable navigator-and-body composition. Processes SHALL retain its navigator while registry data is loading, successfully empty, unavailable, or populated. Populated workspaces SHALL organize navigation, selected-process state and actions, bounded output, notices, and contextual shortcuts into stable regions that adapt without losing operator state.

#### Scenario: Render without a selected process

- **WHEN** the registry is loading, successfully empty, or unavailable at startup
- **THEN** the process navigator remains visible and identifies the current collection state
- **AND** the adjacent or stacked body presents the corresponding explanation and recovery actions without fake process details

#### Scenario: Use a compact terminal

- **WHEN** the terminal supports the functional minimum but does not meet the wide-layout threshold
- **THEN** the active navigator stacks above its body and the top-level tabs remain available
- **AND** switching workspaces or crossing layout thresholds preserves each workspace selection
