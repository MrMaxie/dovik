# tui-presentation Specification

## Purpose
TBD - created by archiving change design-tui-operator-workspace. Update Purpose after archive.
## Requirements
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

The TUI SHALL present a state-specific workspace with persistent top-level tabs and a stable navigator-and-body composition. Processes SHALL retain its navigator while registry data is loading, successfully empty, unavailable, or populated. Populated workspaces SHALL organize navigation, selected-process state and actions, bounded output, notices, and contextual shortcuts into stable regions that adapt without losing operator state.

#### Scenario: Render without a selected process

- **WHEN** the registry is loading, successfully empty, or unavailable at startup
- **THEN** the process navigator remains visible and identifies the current collection state
- **AND** the adjacent or stacked body presents the corresponding explanation and recovery actions without fake process details

#### Scenario: Use a compact terminal

- **WHEN** the terminal supports the functional minimum but does not meet the wide-layout threshold
- **THEN** the active navigator stacks above its body and the top-level tabs remain available
- **AND** switching workspaces or crossing layout thresholds preserves each workspace selection

### Requirement: Immediate process-state hierarchy

The selected process, its project, lifecycle state, command, available action, pending work, and relevant recovery message SHALL be visually prominent without collapsing distinct values into one compact identity string or obscuring current output.

#### Scenario: Inspect a selected process

- **WHEN** registry and runtime data are available
- **THEN** the navigator marks the selected project and process using both a marker and visual emphasis
- **AND** running process markers remain green even on the selected row while inactive markers remain hollow
- **AND** the process body presents process, project, status, and command as distinguishable labels with minimally indented values
- **AND** only lifecycle actions applicable to the selected state appear in a dedicated contextual strip immediately above the global shortcut bar

#### Scenario: Observe a pending or failed action

- **WHEN** a lifecycle request is pending or fails
- **THEN** the process summary shows what is pending or what did not happen
- **AND** exposes the available retry or recovery action without replacing the current selection or output
- **AND** keeps raw protocol or transport diagnostics behind the existing details action

### Requirement: Readable bounded output

The output region SHALL prioritize recent ordered events, preserve sequence and truncation behavior, and distinguish stdout from stderr without relying only on color.

#### Scenario: Read mixed process output

- **WHEN** the selected runtime has both stdout and stderr events
- **THEN** the TUI renders them below a visually distinct process-output heading in daemon sequence order with persistent textual stream attribution
- **AND** styling may reinforce but does not replace that attribution
- **AND** the preview uses the available output-region height instead of imposing a smaller fixed line cap

#### Scenario: Older output is unavailable

- **WHEN** the daemon reports truncated output
- **THEN** the output region explains in plain language that earlier lines were discarded and the newest output remains visible
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
- **THEN** a developer or coding agent inspects screenshots of populated, empty, unavailable, pending, failed, compact, minimum-size, and below-minimum states rendered through ttyglass
- **AND** exercises keyboard navigation and browser-driven PTY resize
- **AND** confirms through live diagnostics that the visible TUI state matches the application state

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

### Requirement: Persistent daemon-detection indicator

The TUI SHALL show a compact daemon status at the right edge of its persistent header at every supported terminal size. The status SHALL use a colored bullet followed by explicit text, preserve the header's background across its full width, reflect the latest registry connection outcome already represented by the TUI model, and SHALL NOT imply continuous health monitoring, expose diagnostics, or conflict with the selected managed-process state.

#### Scenario: Check daemon availability

- **WHEN** the TUI is waiting for its initial or explicit registry request
- **THEN** the header shows `• daemon checking`
- **AND** the text remains distinct from managed-process lifecycle state

#### Scenario: Detect a compatible daemon

- **WHEN** the latest registry request succeeds with either an empty or populated registry
- **THEN** the header shows `• daemon online`
- **AND** the label remains visible independently of selected-process state

#### Scenario: Start the daemon from the TUI

- **WHEN** an explicit native daemon launch is pending
- **THEN** the header shows `• daemon starting`
- **AND** does not claim detection until a compatible protocol response arrives

#### Scenario: Fail to detect the daemon

- **WHEN** the latest registry request fails
- **THEN** the header shows `• daemon offline`
- **AND** applicable recovery remains in the state-specific workspace and shortcut bar rather than expanding the header

#### Scenario: Render daemon state without color

- **WHEN** color output is disabled or unavailable
- **THEN** the full daemon text remains readable and carries the complete state meaning without relying on the bullet color
- **AND** the label remains right-aligned without exceeding the terminal width

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

### Requirement: Contextual terminal controls

The TUI SHALL use one contextual shortcut convention with lower-case muted labels, direct key symbols, and border-colored muted dot separators. Opposing arrow keys SHALL be grouped with a space, such as `↑ ↓ move` and `← → tabs`, instead of using slash notation or separate duplicate labels. Long command values SHALL wrap within their region instead of being replaced by an ellipsis.

#### Scenario: Read workspace controls

- **WHEN** an operator opens any workspace or overlay
- **THEN** visible hints use the same grouped key-label grammar, such as `↑ ↓ move`, without giving the first hint a different background
- **AND** only actions available in the current context are advertised

### Requirement: In-place forms and confirmation

Project, persona, and process-action editors SHALL present their applicable fields in one navigable form inside the main TUI. The active editor SHALL use one clear focus frame while the underlying navigator is visually subdued. Entered values SHALL be visually distinct from labels and focus indicators, and submission SHALL open a separate confirmation modal before mutation.

#### Scenario: Edit a project

- **WHEN** an operator opens a project editor
- **THEN** Tab, Shift+Tab, up, and down can move among applicable fields without completing artificial sections
- **AND** the form does not prefix entered values with a synthetic `> ` prompt
- **AND** the editor does not render a second Dovik application header inside the body
- **AND** submitting the form opens a distinct confirmation modal

### Requirement: Scannable project and persona details

Projects and Personas SHALL distinguish entity names, field labels, minimally indented values, section headings, lists, and status markers through a consistent hierarchy. Project protection SHALL appear below the project name instead of beside it, and every repository operation SHALL be listed with its own allowed or blocked state.

#### Scenario: Inspect a project identity

- **WHEN** an operator selects a configured project
- **THEN** the body presents project, repository, persona, Git author, protection, policy, operation permissions, persona switching, and session state as distinct fields or sections
- **AND** every repository operation appears once with a filled marker for allowed or a hollow marker for blocked, without repeating the same state in text
- **AND** protection is not compressed into the project-name line

#### Scenario: Inspect a persona

- **WHEN** an operator selects a saved persona
- **THEN** persona name, ID, GitHub account, host, Git author name, Git author email, and assigned projects use the same label-value-list hierarchy as project details
- **AND** assigned projects are rendered as a readable list rather than an undifferentiated comma-separated value

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
