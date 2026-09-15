# product-documentation Specification

## Purpose
TBD - created by archiving change prepare-dovik-1-0-release. Update Purpose after archive.
## Requirements
### Requirement: Public user documentation

Dovik SHALL provide a build-checked Astro Starlight site at `https://maxie.dev/dovik/` covering installation, first process, daily CLI and TUI use, MCP, identity and governed GitHub operations, native and container platforms, security and private state, updates, releases, checksums, and unsigned-artifact limitations.

#### Scenario: Start from the public landing page
- **WHEN** a new user opens the documentation
- **THEN** they can identify Dovik's responsibility, choose the correct installation path, and reach a first managed process workflow
- **AND** the page does not imply DNS, HTTPS, routing, scheduling, remote execution, or container orchestration

### Requirement: Truthful sanitized TUI preview

The documentation landing page SHALL show a sanitized representation of the real Dovik TUI that is compared with a real ttyglass session. The preview SHALL contain no private path, account, endpoint, credential, or captured process output.

#### Scenario: Review the landing page
- **WHEN** the documentation is checked at desktop and mobile widths
- **THEN** the preview preserves the real TUI hierarchy and recognizable controls
- **AND** remains readable without disclosing local operator data

### Requirement: Versioned agent skills

Every native archive SHALL include Codex and Claude skill entrypoints aligned with the released behavior. The skills SHALL keep Dovik transparent during ordinary work: agents use standard `gh` commands and available lifecycle tools without inspecting personas, daemon state, or routing first. They SHALL never call `gh auth switch`, bypass a rejected governed operation, signal a process outside the registry, or configure identity and registrations without explicit user scope.

#### Scenario: Agent reaches an unconfigured project
- **WHEN** an agent runs `gh` from a repository without configured identity
- **THEN** `gh` behaves like the original GitHub CLI without exposing Dovik or prompting for a persona
- **AND** the agent does not inspect or configure the underlying routing
