# Approach

## Standardize navigation

Left and right continue to switch the persistent top-level tabs. Up and down become the only navigator selection keys. `l` refreshes the active workspace, removing the identity-specific refresh behavior from `i`. The existing `i` shortcut remains a direct route from Processes to Identity.

## Make Enter the primary action

Enter on Processes opens a themed action menu for the selected process. The menu obtains fresh status and offers only applicable lifecycle actions before returning to the TUI. Direct lifecycle shortcuts remain available for experienced operators.

Enter on Identity opens the existing project questionnaire for the selected project. Enter on Personas opens a focused editor for the selected persona's public account and Git-author metadata. Persona IDs remain stable, and the editor never reads or displays credentials.

## Preserve authority and state

All mutations continue through the existing operator client and daemon. Opening a menu or editor does not mutate state. Escape and Ctrl+C return without saving. After the secondary form closes, the TUI reloads its current daemon-backed state.

# Coverage ledger

| Surface | Owner | Disposition |
|---|---|---|
| Workspace keys | `internal/tui/model.go` | modify |
| Contextual shortcuts | `internal/tui/presentation.go`, `internal/tui/identity.go` | modify |
| Process action menu | `internal/tui/run.go` | extend |
| Persona editor | `internal/identityui` | extend |
| User guidance | identity and TUI docs | modify |
