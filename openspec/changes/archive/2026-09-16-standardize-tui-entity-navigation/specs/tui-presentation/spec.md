## ADDED Requirements

### Requirement: Consistent entity navigation and primary actions

Every top-level TUI workspace SHALL use the up and down arrow keys for navigator selection, `l` for refresh, and Enter for the selected entity's primary action. Processes SHALL open an applicable lifecycle action menu, Identity SHALL open project configuration, and Personas SHALL open persona editing. Opening a primary action SHALL not mutate state until the operator chooses and confirms a change.

#### Scenario: Navigate every entity list

- **WHEN** an operator presses the up or down arrow key in Processes, Identity, or Personas
- **THEN** the selection moves within the active navigator
- **AND** `j` and `k` are not advertised or handled as selection keys

#### Scenario: Refresh every workspace

- **WHEN** an operator presses `l` in any top-level workspace
- **THEN** Dovik refreshes that workspace through its existing operator client

#### Scenario: Open the selected entity

- **WHEN** an operator presses Enter on a selected process, project identity, or persona
- **THEN** Dovik opens the matching process action menu, project questionnaire, or persona editor
- **AND** Escape or Ctrl+C returns without applying a change
