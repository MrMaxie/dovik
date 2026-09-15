# release-distribution Specification

## Purpose
TBD - created by archiving change prepare-dovik-1-0-release. Update Purpose after archive.
## Requirements
### Requirement: Shared binary version and offline help

Dovik SHALL keep one SemVer source used by every shipped binary. `dovik --version` SHALL print `dovik 1.0.0`, and `dovikd --version` SHALL print `dovikd 1.0.0`. Root and subcommand help SHALL succeed without connecting to a daemon.

#### Scenario: Inspect a binary before installation
- **WHEN** a user runs `--version` or `--help` from an unpacked archive
- **THEN** the selected binary prints matching metadata and exits with status zero
- **AND** it does not open the control transport or read daemon state

### Requirement: Native portable archives

The 1.0.0 release SHALL provide Windows x64 ZIP, Linux x64 tar.gz, macOS x64 tar.gz, and macOS arm64 tar.gz archives built and verified on matching native runners. Each archive SHALL contain version-matched `dovik`, `dovikd`, the optional `gh` proxy, README, changelog, license, third-party license information, and Codex and Claude skills.

#### Scenario: Verify a downloaded archive
- **WHEN** a release archive is downloaded and unpacked outside the source checkout
- **THEN** its SHA-256 matches the published checksum
- **AND** its exact file set, executable permissions, help, versions, and isolated daemon/client smoke test pass

### Requirement: Non-root container distribution

The Linux amd64 image SHALL run `dovikd` as UID 10001, expose no network port, and manage only processes inside its container. A 1.0.0 release SHALL publish image tags `1.0.0`, `1.0`, `1`, and `latest` for the same digest.

#### Scenario: Inspect the released image
- **WHEN** a user runs the image without extra privileges or mounts
- **THEN** the daemon starts as UID 10001 with a container-local Unix socket
- **AND** no host process or container engine is available to it

### Requirement: Tag-gated and independently verified publication

Candidate builds SHALL target an exact source revision and SHALL NOT publish. Only a matching `v1.0.0` tag SHALL authorize release automation to create public artifacts. Published archives, container tags, documentation, and Scoop installation SHALL be verified again without using the source checkout.

#### Scenario: Build a candidate
- **WHEN** an operator dispatches the candidate workflow with an exact master revision
- **THEN** native artifacts and verification evidence are retained as workflow artifacts
- **AND** no GitHub Release, GHCR tag, repository visibility, Pages deployment, or Scoop submission changes
