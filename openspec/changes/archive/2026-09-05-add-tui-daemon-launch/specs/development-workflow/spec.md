## ADDED Requirements

### Requirement: Isolated TUI daemon-launch verification

The repository SHALL provide repeatable native Windows development and browser-harness paths that place `dovik.exe` and `dovikd.exe` together and verify independent daemon launch without using the user's default control endpoint or registry. The isolated harness SHALL use development-tagged launch configuration that is unavailable to ordinary production invocation. Test and harness cleanup SHALL be owned outside the TUI and SHALL remove only resources created for that isolated run.

#### Scenario: Exercise daemon launch through the browser TUI

- **WHEN** a developer or coding agent runs `just tui-web` and invokes `Start daemon` in the isolated unavailable state
- **THEN** the actual TUI starts the colocated fixture daemon through the same native launcher used by the product
- **AND** the TUI becomes connected through an isolated user-local Named Pipe and registry
- **AND** the user's default daemon and registry are not started, stopped, read, or changed

#### Scenario: Restart the TUI after fixture daemon launch

- **WHEN** the browser harness restarts or closes its TUI child after the isolated daemon becomes ready
- **THEN** the same fixture daemon remains available independently of the TUI child
- **AND** a replacement TUI reconnects without launching another daemon

#### Scenario: End isolated verification

- **WHEN** the browser harness or native integration test ends
- **THEN** its outer fixture owner stops only the daemon created for that isolated run
- **AND** removes its temporary endpoint, registry, binaries, PTY, diagnostic connection, and browser resources
- **AND** leaves pre-existing user processes and private registry state unchanged
