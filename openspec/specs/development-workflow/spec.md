# development-workflow Specification

## Purpose
TBD - created by archiving change establish-local-process-supervision. Update Purpose after archive.
## Requirements
### Requirement: Reproducible Go toolchain

The repository SHALL define module `github.com/MrMaxie/dovik` and pin Go `1.27.0`, just `1.58.0`, and goimports `0.49.0` through mise without adding Node or npm.

#### Scenario: Install developer tools

- **WHEN** a developer runs `mise install`
- **THEN** mise installs the pinned Go, just, and goimports versions
- **AND** does not install Node or npm for the project

### Requirement: Standard development commands

The Justfile SHALL provide `build`, `test`, `format`, `lint`, `run`, `run-cli`, `openspec-check`, `check`, `test-linux`, and `run-linux`. Format SHALL run `go fmt ./...` and `goimports -w .`; lint SHALL run `go vet ./...`; and check SHALL combine lint, test, build, schema validation, and strict OpenSpec validation.

#### Scenario: Run the repository gate

- **WHEN** a developer runs `just check`
- **THEN** Go vet, Go tests, Go builds, Arcantry schema validation, and strict validation of all OpenSpec artifacts succeed or the command fails

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
