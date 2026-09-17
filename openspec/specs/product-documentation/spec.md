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

Every native archive SHALL include Codex and Claude skill entrypoints aligned with the released behavior. The skills SHALL keep Dovik transparent during ordinary work: agents use the complete standard `gh` command surface and available lifecycle tools without inspecting personas, daemon state, routing, or Dovik's operation catalogue first. Commands unknown to the installed Dovik version SHALL remain the responsibility of GitHub CLI. Agents SHALL never call `gh auth switch`, bypass a rejected governed isolated operation, signal a process outside the registry, or configure identity and registrations without explicit user scope.

#### Scenario: Agent uses GitHub CLI in a configured proxy-level project

- **WHEN** an agent runs an ordinary `gh` command, alias, extension, or newly introduced GitHub CLI command
- **THEN** it behaves like the original GitHub CLI with the configured project persona
- **AND** the agent does not inspect or reason about Dovik routing

#### Scenario: Agent reaches an unconfigured project

- **WHEN** an agent runs `gh` from a repository without configured identity
- **THEN** `gh` behaves like the original GitHub CLI without exposing Dovik or prompting for a persona
- **AND** the agent does not inspect or configure the underlying routing

### Requirement: Shell prompt persona cookbook

The public identity documentation SHALL provide a portable Starship custom-module recipe based on `dovik whoami`. It SHALL describe the persona-name, unconfigured, and unavailable outputs; use repository-aware invocation; cover the default Starship configuration path and `STARSHIP_CONFIG`; and remain applicable across supported operating systems and common shells without embedding workstation-specific paths.

#### Scenario: Add persona status to Starship

- **WHEN** a user follows the documented recipe on Windows, Linux, or macOS
- **THEN** Starship invokes `dovik whoami` only inside a Git repository
- **AND** renders the selected Dovik persona or its concise `?` or `!` state
- **AND** the documentation states that the result reports local configuration rather than live GitHub credential validity

### Requirement: Document TUI workspace navigation

The operator documentation SHALL identify Personas, Projects, and Processes as counted top-level TUI workspaces in dependency order, explain grouped process navigation and contextual controls, and document in-place editing, full retained-output viewing, and shell-free follow mode in both the TUI and CLI.

#### Scenario: Learn the TUI and log workflow

- **WHEN** a user reads the CLI and TUI guide
- **THEN** the guide explains project-grouped process rows, recent output preview, full output scrolling, follow and pause, and in-place forms with explicit confirmation
- **AND** documents that `process logs --follow` works for running or stopped processes and cannot be combined with JSON mode

### Requirement: Consistent TUI key documentation

The operator documentation SHALL describe up and down for entity selection and scrolling, left and right for workspace switching, Enter as the primary action for the selected entity, `o` for full process output, `f` for follow and pause, and `l` for refresh. It SHALL use Personas, Projects, and Processes consistently and distinguish workspace switching from selection inside a workspace.

#### Scenario: Learn the TUI interaction model

- **WHEN** a user reads the CLI and TUI guide
- **THEN** the guide presents one consistent key model for Personas, Projects, and Processes
- **AND** explains the contextual controls for entity actions, retained output, and confirmation
