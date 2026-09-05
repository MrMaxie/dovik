## ADDED Requirements

### Requirement: Unambiguous TUI preview controls

The browser harness SHALL present one compact preview toolbar whose status and restart action explicitly refer to the TUI child process. It SHALL NOT imply that child liveness represents daemon connectivity or managed-process state.

#### Scenario: View a running TUI child

- **WHEN** the TUI child is running
- **THEN** the toolbar identifies the surface as "TUI preview" and identifies the status as belonging to the TUI
- **AND** the restart action is labeled "Restart TUI"
- **AND** the child PID appears in diagnostic detail rather than the primary toolbar

#### Scenario: Restart the TUI

- **WHEN** a developer activates "Restart TUI"
- **THEN** the harness restarts only its TUI child through the existing restart mechanism
- **AND** does not request a managed-process or daemon restart

#### Scenario: Observe an exited child

- **WHEN** the TUI child exits
- **THEN** the toolbar identifies the exited subject as the TUI and retains the existing applicable restart action
- **AND** does not present that status as a managed project's lifecycle state

### Requirement: On-demand live harness diagnostics

The browser harness SHALL provide a keyboard-accessible collapsible diagnostic panel, collapsed by default. Diagnostic capture and bounded history SHALL continue while collapsed, independently of PTY output.

#### Scenario: Open the preview

- **WHEN** a browser session opens the harness
- **THEN** diagnostics are collapsed and a clearly named disclosure control is available
- **AND** the terminal occupies the space made available by the collapsed panel

#### Scenario: Inspect live diagnostics

- **WHEN** a developer expands diagnostics
- **THEN** the panel shows retained bounded history and new records live, with child PID and the existing clear-log action
- **AND** expanding or collapsing does not clear the diagnostic history or write diagnostics into terminal output

#### Scenario: Operate the disclosure by keyboard

- **WHEN** a developer focuses and activates the diagnostic disclosure using the keyboard
- **THEN** it exposes its expanded or collapsed state and a visible focus indicator
- **AND** the developer can return focus to the terminal

#### Scenario: Resize after toggling diagnostics

- **WHEN** diagnostics expand or collapse
- **THEN** the harness refits the terminal and propagates the available rows and columns to the backing PTY
- **AND** the same TUI process remains active with its selection and output position preserved
- **AND** controls and terminal remain usable at the normal and narrower verification viewports
