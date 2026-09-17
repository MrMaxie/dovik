# Approach

## Make workspace navigation persistent

Replace the process-specific header title with a shared top row containing the Dovik brand, `Processes`, `Identity`, and `Personas` tabs, plus the existing daemon indicator. The active tab uses the established selected treatment and each tab remains readable without color. Left and right arrows move between tabs without changing selection stored inside any tab.

The existing `i` key remains a direct route to Identity and refreshes identity data when already there. Escape returns from Identity or Personas to Processes. Up, down, `j`, and `k` continue to change the selection inside the active navigator.

## Reuse one navigator-and-body contract

Processes always renders its navigator at the left in wide layouts or above the body in compact layouts. Loading, empty, and unavailable states appear in the body while the navigator remains present and accurately describes the collection state. Populated process summaries and output retain their current hierarchy, lifecycle actions, selection, polling, and scroll behavior.

Identity keeps its project navigator and detail body. Personas uses the same layout contract: the navigator lists saved persona names, and the body shows the selected persona's GitHub account, Git author identity, and assigned projects. Empty, loading, and unavailable persona states use the same state treatment as Identity and expose no credentials.

## Preserve responsive and operational behavior

The existing wide and compact breakpoints remain authoritative. The tab row fits inside the fixed one-line header and truncates safely before the daemon indicator. Terminal sizes below the supported minimum retain the current minimum-size message and quit/help behavior.

# Coverage ledger

| Surface | Owner | Existing analogue | Disposition |
|---|---|---|---|
| Top workspace navigation | `internal/tui/presentation.go` | persistent header and selected navigator rows | extend |
| Process collection and body | `internal/tui/presentation.go` | populated process navigator plus identity layout | extend |
| Identity project view | `internal/tui/identity.go` | current identity navigator and details | reuse |
| Persona collection and details | `internal/tui/identity.go` | identity navigator and details | extend |
| Keyboard transitions | `internal/tui/model.go` | existing process and identity key handling | extend |
| Small-terminal fallback | `internal/tui/presentation.go` | current minimum-size state | reuse |

# Trade-offs

- A one-line tab row uses more horizontal space than a single title, but it keeps all primary workspaces discoverable without adding another modal or shortcut-only surface.
- Personas reuses the identity snapshot, so switching to either identity-related tab refreshes both views together and does not introduce another daemon API.
- The `i` shortcut remains for compatibility even though Identity is now visible in the tab row.
