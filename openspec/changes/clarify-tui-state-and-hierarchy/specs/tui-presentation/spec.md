## MODIFIED Requirements

### Requirement: Responsive operator workspace

The TUI SHALL present a state-specific workspace that distinguishes registry loading, successful emptiness, unavailable data, and populated data. Populated workspaces SHALL organize navigation, selected-process state and actions, bounded output, notices, and contextual shortcuts into stable regions that adapt without losing operator state.

#### Scenario: Use a wide terminal with registered processes

- **WHEN** registry data contains processes and the terminal is at least 96 columns wide and 24 rows high
- **THEN** the TUI shows a persistent header, left process navigator, selected-process summary above output, and one-line contextual shortcut bar
- **AND** the summary has no separate enclosing frame and uses identity, state, and applicable action as its primary hierarchy
- **AND** resizing within the wide layout does not change selection or output scroll position

#### Scenario: Use a compact terminal with registered processes

- **WHEN** registry data contains processes and the terminal is at least 64 columns wide and 18 rows high but does not meet the wide-layout threshold
- **THEN** the TUI stacks navigation, selected-process summary, and output while retaining state, applicable actions, output access, help, and quit
- **AND** crossing between compact and wide layouts preserves selection, details visibility, help visibility, and output scroll position

#### Scenario: Use a terminal below the functional minimum

- **WHEN** the terminal is smaller than 64 columns or 18 rows
- **THEN** the TUI shows required minimum dimensions and retains working help and quit controls
- **AND** restores the appropriate state-specific workspace when sufficient space returns

#### Scenario: Render without a selected process

- **WHEN** the registry is loading, successfully empty, or unavailable at startup
- **THEN** the TUI presents the corresponding primary state instead of empty selected-process and output panels
- **AND** retains the actions and explanation applicable to that state at supported terminal sizes

## ADDED Requirements

### Requirement: Explicit registry outcome and recovery

The TUI SHALL distinguish a successful empty registry from failure to obtain registry data. A registry failure SHALL remain visible without requiring a selected process, with recovery available through existing keyboard controls.

#### Scenario: Load registry data

- **WHEN** the initial registry request is pending
- **THEN** the TUI identifies loading as the current activity
- **AND** does not claim that there are no registered processes

#### Scenario: Open an empty registry

- **WHEN** a registry request succeeds with zero process definitions
- **THEN** the TUI shows one primary "No processes registered" message with concise CLI registration guidance
- **AND** exposes refresh, setup help, and quit
- **AND** does not show selection or process-detail hints without an applicable process

#### Scenario: Fail to load the registry at startup

- **WHEN** the registry request fails before any processes have been loaded
- **THEN** the TUI shows that process data is unavailable and exposes retry and quit
- **AND** does not present the failure as missing process definitions or instruct registration as its recovery
- **AND** raw diagnostic details remain accessible only through explicit disclosure that works without a selected process

#### Scenario: Fail to refresh a populated registry

- **WHEN** a registry refresh fails after processes were successfully loaded
- **THEN** the TUI retains prior selection and output and clearly marks the unavailable or stale data
- **AND** shows recovery instructions without implying a successful refresh

#### Scenario: Retry after a registry failure

- **WHEN** the user presses the existing refresh key and the request succeeds
- **THEN** the TUI clears the failure and renders either the successful empty state or populated workspace according to the response

### Requirement: Actionable terminal setup guidance

The empty-state help SHALL describe the existing CLI registration workflow with valid command syntax, explicit placeholders, and the subsequent terminal refresh step. It SHALL NOT add registry mutation to the TUI.

#### Scenario: Request setup help from an empty registry

- **WHEN** the user opens help from the successful empty state
- **THEN** help provides commands for registering a project and process that match the actual CLI contract
- **AND** clearly distinguishes user-supplied values from literal command syntax
- **AND** explains how to return and refresh the process list using the existing key

### Requirement: Restrained terminal hierarchy

The populated workspace SHALL prioritize selected process identity, lifecycle state, applicable action, and output over framing and decoration. Styling SHALL preserve textual state and stream meaning, readable contrast, and operation without color.

#### Scenario: Inspect populated process data

- **WHEN** a user views a populated workspace
- **THEN** section headings use ordinary case and the primary accent emphasizes selection and contextual keyboard cues
- **AND** alignment and minimal separators establish regions without a separately framed summary or competing filled boxes
- **AND** lifecycle and stream attribution remain readable through text rather than color alone

#### Scenario: Observe a lifecycle operation

- **WHEN** an operation is pending or fails
- **THEN** the summary identifies the action in progress or the failed action and available recovery
- **AND** retains selected identity and output

#### Scenario: Read contextual keyboard hints

- **WHEN** registry, selection, help, or operation state changes
- **THEN** visible hints describe actions applicable to the current context
- **AND** existing key meanings, including refresh, remain unchanged
