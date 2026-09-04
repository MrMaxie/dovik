## MODIFIED Requirements

### Requirement: Reproducible developer toolchain

The repository SHALL define module `github.com/MrMaxie/dovik` and pin Go `1.27.0`, Node `24.18.0`, just `1.58.0`, and goimports `0.49.0` through mise. Node and npm SHALL be used only by development tooling and SHALL NOT become runtime or distribution dependencies.

#### Scenario: Install developer tools

- **WHEN** a developer runs `mise install`
- **THEN** mise installs the pinned Go, Node, just, and goimports versions
- **AND** the Dovik runtime and distribution remain independent of Node and npm

### Requirement: Standard development commands

The Justfile SHALL provide `build`, `test`, `format`, `lint`, `run`, `run-cli`, `tui`, `tui-web`, `test-tui-web`, `openspec-check`, `check`, `test-linux`, `test-integration`, and `run-linux`. Format SHALL run `go fmt ./...` and `goimports -w .`; lint SHALL run `go vet ./...`; and check SHALL combine lint, Go tests, Go builds, browser harness tests, schema validation, and strict OpenSpec validation.

#### Scenario: Run the repository gate

- **WHEN** a developer runs `just check`
- **THEN** Go vet, Go tests, Go builds, browser harness tests, Arcantry schema validation, and strict validation of all OpenSpec artifacts succeed or the command fails

#### Scenario: Start browser TUI verification

- **WHEN** a developer runs `just tui-web`
- **THEN** the current Dovik TUI is available in a loopback-only browser terminal backed by a real PTY or Windows ConPTY
- **AND** the browser presents live TUI diagnostics separately from the terminal output

## ADDED Requirements

### Requirement: Development-only browser TUI harness

The repository SHALL provide a development-only browser harness that runs the actual `dovik tui` process through a real platform PTY, renders it with a maintained terminal emulator, and does not add an HTTP listener, browser surface, or Node dependency to the Dovik runtime or distribution.

#### Scenario: Exercise terminal behavior from a browser

- **WHEN** a developer or coding agent interacts with the harness terminal
- **THEN** ANSI and true color output, alternate screen, raw input, special keys, Ctrl+C, and TUI-supported mouse events pass between the browser terminal and the actual Dovik process
- **AND** a browser resize updates the backing PTY rows and columns
- **AND** the child inherits the repository working directory and current environment without exposing environment values in the browser UI

#### Scenario: Inspect diagnostics without corrupting the TUI

- **WHEN** the development-tagged TUI emits an internal diagnostic record
- **THEN** the harness displays it live in the diagnostic panel
- **AND** the record is not written into the PTY output or retained after the harness exits

#### Scenario: End a harness session

- **WHEN** the TUI exits, the browser disconnects, or the harness is interrupted
- **THEN** the harness closes the PTY, WebSocket, diagnostic connection, HTTP server, and temporary artifacts
- **AND** it does not leave the launched Dovik process or its descendants running
