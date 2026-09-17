## MODIFIED Requirements

### Requirement: Document TUI workspace navigation

The operator documentation SHALL identify Personas, Projects, and Processes as counted top-level TUI workspaces in dependency order, explain grouped process navigation and contextual controls, and document in-place editing, full retained-output viewing, and shell-free follow mode in both the TUI and CLI.

#### Scenario: Learn the TUI and log workflow

- **WHEN** a user reads the CLI and TUI guide
- **THEN** the guide explains project-grouped process rows, recent output preview, full output scrolling, follow and pause, and in-place forms with explicit confirmation
- **AND** documents that `process logs --follow` works for running or stopped processes and cannot be combined with JSON mode

### Requirement: Consistent TUI key documentation

The operator documentation SHALL describe up and down for entity selection and scrolling, left and right for workspace switching, Enter as the primary action for the selected entity, `o` for full process output, `f` for follow and pause, and `l` for refresh. It SHALL use Personas, Projects, and Processes consistently and distinguish workspace switching from selection inside a workspace.

#### Scenario: Learn the TUI interaction model

- **WHEN** a user reads the CLI and TUI guide
- **THEN** the guide presents one consistent key model for Personas, Projects, and Processes
- **AND** explains the contextual controls for entity actions, retained output, and confirmation
