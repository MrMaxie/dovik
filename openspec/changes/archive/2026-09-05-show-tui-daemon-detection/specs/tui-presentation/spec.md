## ADDED Requirements

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
