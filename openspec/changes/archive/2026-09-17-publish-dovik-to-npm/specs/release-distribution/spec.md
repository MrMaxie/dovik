## ADDED Requirements

### Requirement: Native npm distribution

Dovik SHALL publish an unscoped `dovik` launcher package and version-matched optional native packages for Windows x64, Linux x64, macOS x64, and macOS arm64. The launcher package SHALL expose `dovik` and `dovikd`. It SHALL NOT expose, replace, or configure `gh`; the optional proxy binary remains available only inside the native package for explicit operator setup.

#### Scenario: Install Dovik from npm

- **WHEN** a user installs `dovik` on a supported platform
- **THEN** npm selects the matching native optional package
- **AND** `dovik --version`, `dovikd --version`, and offline help execute the version-matched native binaries
- **AND** the user's existing `gh` command is unchanged

### Requirement: Stage-only trusted npm publication

After the initial manual 1.0.0 publication, new npm versions SHALL be assembled only from verified GitHub Release archives and staged through the exact GitHub-hosted workflow, the protected `npm-production` environment, and short-lived OIDC credentials. The workflow SHALL use `npm stage publish` with provenance and SHALL NOT receive a traditional npm token. A maintainer SHALL approve all platform packages with 2FA before approving the root package.

#### Scenario: Publish an npm release

- **WHEN** all five version-matched packages are ready for publication
- **THEN** release assets, checksums, attestations, deterministic tarballs, and native clean-room installations pass before registry staging
- **AND** the four native packages are approved before the root launcher package
- **AND** npm provenance identifies the Dovik repository and exact publishing workflow
