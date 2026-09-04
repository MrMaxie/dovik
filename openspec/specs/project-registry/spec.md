# project-registry Specification

## Purpose
TBD - created by archiving change establish-local-process-supervision. Update Purpose after archive.
## Requirements
### Requirement: Daemon-owned project registry

The daemon SHALL be the sole writer of a registry containing `ProjectDefinition` records with a stable unique ID and an absolute project root directory.

#### Scenario: Add and list a project

- **WHEN** a client adds a project with a unique ID and an existing absolute root directory
- **THEN** the daemon persists the definition and returns it from project listing

#### Scenario: Reject an invalid project root

- **WHEN** a client provides a relative or nonexistent project root directory
- **THEN** the daemon rejects the definition without changing the registry

### Requirement: Project-owned process definitions

The daemon SHALL store each `ProcessDefinition` under one project with an ID unique in that project, an executable command, ordered arguments, an optional working directory, and environment overrides.

#### Scenario: Apply process defaults

- **WHEN** a process definition omits its working directory
- **THEN** the daemon uses the owning project's root directory
- **AND** starts the command directly without an intermediate shell

#### Scenario: Keep environment values private

- **WHEN** a client lists process definitions or requests process status
- **THEN** the response does not include environment override values

### Requirement: Atomic private persistence

The daemon SHALL persist registry changes as private per-user state through a temporary file, durable flush, and atomic replacement.

#### Scenario: Interrupt a registry update

- **WHEN** registry persistence is interrupted before atomic replacement
- **THEN** the previously complete registry remains readable
- **AND** no partial registry becomes authoritative

#### Scenario: Load a corrupt registry

- **WHEN** daemon startup cannot validate the persisted registry
- **THEN** the daemon reports the corruption explicitly
- **AND** does not replace it or start managed processes from partial data

### Requirement: Safe definition removal

The daemon SHALL reject removal of a process definition with an active runtime and SHALL reject removal of a project that still contains process definitions or active runtimes.

#### Scenario: Remove an empty inactive project

- **WHEN** a client removes a project with no process definitions and no active runtime
- **THEN** the daemon removes it from the registry

#### Scenario: Reject removal with owned definitions

- **WHEN** a client removes a project that still owns at least one process definition
- **THEN** the daemon rejects the operation without changing the registry
