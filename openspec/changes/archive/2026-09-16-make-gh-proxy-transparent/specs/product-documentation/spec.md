## MODIFIED Requirements

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
