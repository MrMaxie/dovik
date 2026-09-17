# Context

The TUI already owns a single Bubble Tea model, uses Huh for forms, polls retained output through the shared operator client, and verifies real terminal behavior through ttyglass. The current run loop leaves the main program and starts standalone forms, which causes visible screen replacement and loses spatial context.

# Decisions

## One persistent Bubble Tea program

Process actions, project editing, persona editing, confirmation, and full output are model states rendered by the existing program. Huh forms are embedded through their `Init`, `Update`, and `View` model methods and are sized to the active body instead of owning terminal input, output, or alternate-screen lifecycle.

## Hierarchical process navigator

The registry remains a flat collection keyed by project and process for stable selection. Presentation derives project groups, renders one project row followed by tree-connected process rows, and counts only process rows for navigation.

## Poll-based log following

The daemon protocol remains unchanged. TUI full output and CLI `--follow` repeatedly request events after the last seen sequence. A stopped process still returns its retained tail; following continues until the user exits or the command context ends.

## Safe terminal styling

Process output preserves SGR color styling while stripping terminal control sequences that can move the cursor, clear the screen, alter the title, or otherwise escape the output region. Stream attribution remains textual.

## Explicit confirmation

Project and persona fields remain editable in one navigable Huh group. Submission opens a visually distinct confirmation modal. Only the confirmation action sends a mutation through the operator client.

# Risks and mitigations

- Embedded Huh state may consume global keys unexpectedly. Overlay modes receive input before workspace shortcuts and expose Escape/Ctrl+C as cancel.
- ANSI output may affect layout. Sanitized output is measured and wrapped after unsafe controls are removed.
- Poll-based following can duplicate output. Every request advances from the highest observed sequence.
