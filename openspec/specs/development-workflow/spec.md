# development-workflow Specification

## Purpose
TBD - created by archiving change establish-local-process-supervision. Update Purpose after archive.
## Requirements
### Requirement: Reproducible developer toolchain

The repository SHALL define module `github.com/MrMaxie/dovik` and pin Go `1.27.0`, Node `24.18.0`, just `1.58.0`, and goimports `0.49.0` through mise. Node and npm SHALL be used only by development tooling and SHALL NOT become runtime or distribution dependencies.

#### Scenario: Install developer tools

- **WHEN** a developer runs `mise install`
- **THEN** mise installs the pinned Go, Node, just, and goimports versions
- **AND** the Dovik runtime and distribution remain independent of Node and npm

### Requirement: Standard development commands

The Justfile SHALL provide `build`, `test`, `format`, `lint`, `run`, `run-cli`, `tui`, `tui-ttyglass`, `openspec-check`, `check`, `test-linux`, `test-integration`, and `run-linux`. Format SHALL run `go fmt ./...` and `goimports -w .`; lint SHALL run `go vet ./...`; and check SHALL combine lint, Go tests, Go builds, schema validation, and strict OpenSpec validation.

#### Scenario: Run the repository gate

- **WHEN** a developer runs `just check`
- **THEN** Go vet, Go tests, Go builds, Arcantry schema validation, and strict validation of all OpenSpec artifacts succeed or the command fails

#### Scenario: Start browser TUI verification

- **WHEN** a developer runs `just tui-ttyglass`
- **THEN** ttyglass starts the current Dovik TUI in a loopback-only browser terminal backed by a real PTY or Windows ConPTY
- **AND** ttyglass presents live TUI diagnostics separately from terminal output

### Requirement: Isolated Linux image

The Dockerfile SHALL provide a test stage that tests, vets, and builds the Linux source and a runtime stage that runs `dovikd` as PID 1 without a published port or host-process access.

#### Scenario: Build the Linux test stage

- **WHEN** a developer runs `just test-linux`
- **THEN** Docker builds the test stage and executes the Go test, vet, and build gates inside Linux

#### Scenario: Run the Linux daemon image

- **WHEN** a developer runs `just run-linux`
- **THEN** Docker runs `dovikd` as the container entrypoint without publishing TCP
- **AND** the daemon can own only processes inside that container

### Requirement: Functional container integration coverage

When the first functional Linux runtime is implemented, the project SHALL add `github.com/testcontainers/testcontainers-go` v0.44.0 as a test dependency through `go get` and SHALL use it only for integration or smoke behavior requiring a real container boundary.

#### Scenario: Exercise the functional Linux runtime

- **WHEN** the functional container integration suite runs with Docker available
- **THEN** Testcontainers builds the local Dockerfile, starts `dovikd`, and executes `dovik` through container exec without exposing TCP
- **AND** verifies registration, start, status, logs, stop, and restart for a fixture process
- **AND** always cleans the container and test resources

#### Scenario: Docker is unavailable

- **WHEN** the functional container integration suite cannot reach a Docker Engine
- **THEN** only that suite is skipped with an explicit reason

#### Scenario: Run tests outside the container boundary

- **WHEN** a test covers a unit, native Windows Job Object, or other behavior that does not require a real container boundary
- **THEN** the test does not use Testcontainers

### Requirement: Ttyglass TUI verification

The repository SHALL use the pinned ttyglass development dependency to run the actual `dovik tui` process through a real platform PTY. It SHALL NOT maintain a project-specific HTTP server, WebSocket protocol, terminal emulator, PTY bridge, browser toolbar, or session lifecycle implementation.

#### Scenario: Exercise terminal behavior from a browser

- **WHEN** a developer or coding agent starts the ttyglass preview
- **THEN** ANSI output, true color, alternate screen, raw input, special keys, Ctrl+C, and browser-driven resize reach the actual Dovik process through ttyglass
- **AND** ttyglass owns browser serving, PTY lifecycle, session retention, diagnostics, restart, and cleanup

#### Scenario: Inspect diagnostics without corrupting the TUI

- **WHEN** Dovik runs inside ttyglass and emits an internal diagnostic record
- **THEN** it uses ttyglass's documented diagnostics endpoint and bearer token
- **AND** the record remains separate from terminal output and contains no credential or private environment value

#### Scenario: Verify questionnaire interruption

- **WHEN** the identity interruption smoke test reaches the questionnaire through a ttyglass session and sends Ctrl+C
- **THEN** the questionnaire exits unsuccessfully without saving project or identity state
- **AND** the test stops only its ttyglass session and isolated daemon fixture
