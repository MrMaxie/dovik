## ADDED Requirements

### Requirement: Shell prompt persona cookbook

The public identity documentation SHALL provide a portable Starship custom-module recipe based on `dovik whoami`. It SHALL describe the persona-name, unconfigured, and unavailable outputs; use repository-aware invocation; cover the default Starship configuration path and `STARSHIP_CONFIG`; and remain applicable across supported operating systems and common shells without embedding workstation-specific paths.

#### Scenario: Add persona status to Starship

- **WHEN** a user follows the documented recipe on Windows, Linux, or macOS
- **THEN** Starship invokes `dovik whoami` only inside a Git repository
- **AND** renders the selected Dovik persona or its concise `?` or `!` state
- **AND** the documentation states that the result reports local configuration rather than live GitHub credential validity
