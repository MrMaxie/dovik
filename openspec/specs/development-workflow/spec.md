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

The Justfile SHALL provide `build`, `test`, `format`, `lint`, `run`, `run-cli`, `tui`, `tui-web`, `test-tui-web`, `openspec-check`, `check`, `test-linux`, `test-integration`, and `run-linux`. Format SHALL run `go fmt ./...` and `goimports -w .`; lint SHALL run `go vet ./...`; and check SHALL combine lint, Go tests, Go builds, browser harness tests, schema validation, and strict OpenSpec validation.

#### Scenario: Run the repository gate

- **WHEN** a developer runs `just check`
- **THEN** Go vet, Go tests, Go builds, browser harness tests, Arcantry schema validation, and strict validation of all OpenSpec artifacts succeed or the command fails

#### Scenario: Start browser TUI verification

- **WHEN** a developer runs `just tui-web`
- **THEN** the current Dovik TUI is available in a loopback-only browser terminal backed by a real PTY or Windows ConPTY
- **AND** the browser presents live TUI diagnostics separately from the terminal output

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

### Requirement: Unambiguous TUI preview controls

The browser harness SHALL present one compact preview toolbar whose status and restart action explicitly refer to the TUI child process. It SHALL NOT imply that child liveness represents daemon connectivity or managed-process state.

#### Scenario: View a running TUI child

- **WHEN** the TUI child is running
- **THEN** the toolbar identifies the surface as "TUI preview" and identifies the status as belonging to the TUI
- **AND** the restart action is labeled "Restart TUI"
- **AND** the child PID appears in diagnostic detail rather than the primary toolbar

#### Scenario: Restart the TUI

- **WHEN** a developer activates "Restart TUI"
- **THEN** the harness restarts only its TUI child through the existing restart mechanism
- **AND** does not request a managed-process or daemon restart

#### Scenario: Observe an exited child

- **WHEN** the TUI child exits
- **THEN** the toolbar identifies the exited subject as the TUI and retains the existing applicable restart action
- **AND** does not present that status as a managed project's lifecycle state

### Requirement: On-demand live harness diagnostics

The browser harness SHALL provide a keyboard-accessible collapsible diagnostic panel, collapsed by default. Diagnostic capture and bounded history SHALL continue while collapsed, independently of PTY output.

#### Scenario: Open the preview

- **WHEN** a browser session opens the harness
- **THEN** diagnostics are collapsed and a clearly named disclosure control is available
- **AND** the terminal occupies the space made available by the collapsed panel

#### Scenario: Inspect live diagnostics

- **WHEN** a developer expands diagnostics
- **THEN** the panel shows retained bounded history and new records live, with child PID and the existing clear-log action
- **AND** expanding or collapsing does not clear the diagnostic history or write diagnostics into terminal output

#### Scenario: Operate the disclosure by keyboard

- **WHEN** a developer focuses and activates the diagnostic disclosure using the keyboard
- **THEN** it exposes its expanded or collapsed state and a visible focus indicator
- **AND** the developer can return focus to the terminal

#### Scenario: Resize after toggling diagnostics

- **WHEN** diagnostics expand or collapse
- **THEN** the harness refits the terminal and propagates the available rows and columns to the backing PTY
- **AND** the same TUI process remains active with its selection and output position preserved
- **AND** controls and terminal remain usable at the normal and narrower verification viewports

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
