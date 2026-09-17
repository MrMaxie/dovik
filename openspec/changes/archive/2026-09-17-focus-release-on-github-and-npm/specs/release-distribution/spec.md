## MODIFIED Requirements

### Requirement: Shared binary version and offline help

Dovik SHALL keep one SemVer source used by every shipped binary and package manifest. `dovik --version` SHALL print `dovik <version>`, and `dovikd --version` SHALL print `dovikd <version>`. Root and subcommand help SHALL succeed without connecting to a daemon.

#### Scenario: Inspect a binary before installation

- **WHEN** a user runs `--version` or `--help` from an unpacked archive
- **THEN** the selected binary prints metadata matching the release version and exits with status zero
- **AND** it does not open the control transport or read daemon state

### Requirement: Native portable archives

Each public release SHALL provide Windows x64 ZIP, Linux x64 tar.gz, macOS x64 tar.gz, and macOS arm64 tar.gz archives built and verified on matching native runners. Archive names SHALL derive from the shared release version. Each archive SHALL contain version-matched `dovik`, `dovikd`, the optional `gh` proxy, README, changelog, license, third-party license information, and Codex and Claude skills.

#### Scenario: Verify a downloaded archive

- **WHEN** a release archive is downloaded and unpacked outside the source checkout
- **THEN** its SHA-256 matches the published checksum
- **AND** its exact file set, executable permissions, help, versions, and isolated daemon/client smoke test pass

### Requirement: Tag-gated and independently verified publication

Candidate builds SHALL target an exact source revision and SHALL NOT publish. Only a matching `v<version>` tag on the current `master` commit SHALL authorize release automation. New versions SHALL publish verified native archives and checksums through GitHub Releases, then stage version-matched npm packages assembled from those public archives. The workflow SHALL NOT publish a new Scoop manifest or GHCR image.

#### Scenario: Build a candidate

- **WHEN** an operator dispatches the candidate workflow with an exact master revision
- **THEN** native artifacts and verification evidence are retained as workflow artifacts
- **AND** no GitHub Release or npm package changes

#### Scenario: Publish a release

- **WHEN** an operator pushes a matching version tag after a successful candidate
- **THEN** GitHub publishes and independently verifies all four native archives and checksums
- **AND** the release event stages all five npm packages through trusted publishing
- **AND** no Scoop manifest or GHCR image is created for that version

## REMOVED Requirements

### Requirement: Non-root container distribution

**Reason**: Versioned container publication is no longer one of Dovik's supported release channels. Docker and Podman remain part of source and integration testing, and the existing 1.0.0 image remains available.

**Migration**: Install new releases from npm or the native GitHub Release archives. Build the runtime container from source when container-local supervision is required.
