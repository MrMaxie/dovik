## ADDED Requirements

### Requirement: Unified stdout and stderr events

The daemon SHALL capture stdout and stderr concurrently into one ordered event buffer. Every event SHALL include a monotonically increasing sequence number, capture time, stream name, and captured bytes.

#### Scenario: Process writes to both streams

- **WHEN** a managed process writes to stdout and stderr
- **THEN** the daemon records events from both streams in one sequence
- **AND** preserves each event's stream attribution

### Requirement: Bounded output retention

The output buffer SHALL enforce configured byte or event limits by discarding the oldest complete events and SHALL report that truncation occurred.

#### Scenario: Output exceeds the limit

- **WHEN** appending an event would exceed the configured buffer limit
- **THEN** the daemon discards the oldest complete events until the buffer is within the limit
- **AND** the next log response indicates that earlier output was truncated

### Requirement: Current or latest runtime access

The daemon SHALL retain output only for the current or most recent runtime of a process definition and SHALL provide sequence-aware tail reads to clients.

#### Scenario: Request a bounded tail

- **WHEN** a client requests the last N available events
- **THEN** the daemon returns at most N events in sequence order from the current or most recent runtime

### Requirement: Non-persistent output

Captured output SHALL remain in memory and SHALL NOT be restored after daemon restart.

#### Scenario: Restart the daemon

- **WHEN** the daemon restarts after capturing process output
- **THEN** prior output events are unavailable
- **AND** registry and runtime reconciliation do not recreate log content
