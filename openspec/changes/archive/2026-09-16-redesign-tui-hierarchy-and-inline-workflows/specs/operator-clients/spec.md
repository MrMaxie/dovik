## ADDED Requirements

### Requirement: Follow retained process output

The CLI SHALL support `dovik process logs --follow` for any registered process. Follow mode SHALL print the retained tail, then request only later sequence events until interrupted, regardless of whether the process is running or stopped. Human output SHALL preserve safe process color styling while retaining textual stream attribution; JSON mode SHALL reject follow mode because it cannot produce one finite JSON document.

#### Scenario: Follow a running process

- **WHEN** an operator runs `dovik process logs --project PROJECT --process PROCESS --follow`
- **THEN** the CLI prints the requested retained tail and subsequently prints new ordered events
- **AND** exits cleanly when its context is cancelled

#### Scenario: Follow a stopped process

- **WHEN** the selected process has retained output but is stopped
- **THEN** the CLI prints that output and continues waiting for later events
- **AND** does not require or start a runtime

#### Scenario: Request JSON follow mode

- **WHEN** `--json` and `--follow` are combined
- **THEN** the CLI rejects the incompatible output modes with an invalid-arguments error
