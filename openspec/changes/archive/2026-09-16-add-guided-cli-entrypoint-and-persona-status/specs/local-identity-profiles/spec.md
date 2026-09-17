## MODIFIED Requirements

### Requirement: Consistent operator surfaces and platform evidence

CLI and TUI SHALL expose persona, project context, policy, and session state without secrets. The interactive questionnaire SHALL use the established Dovik terminal palette, SHALL describe proxy-level as transparent persona routing, and SHALL request policy presets, explicit operation exceptions, and alternative personas only when agent-isolation is selected. Native Windows, Linux, macOS, Docker, and Podman verification SHALL be reported separately; compilation alone SHALL NOT establish isolation support.

#### Scenario: Configure transparent proxy-level routing

- **WHEN** an operator configures a project in proxy-level mode
- **THEN** the questionnaire selects the persona and optional repository-local Git author
- **AND** does not present isolated-agent policy or permission questions
- **AND** preserves an existing stored policy for later isolation use

#### Scenario: Configure agent isolation

- **WHEN** an operator selects agent-isolation
- **THEN** the questionnaire requests the starting preset, explicit permission choices, and any permitted alternative personas
- **AND** includes those isolation controls in the confirmation summary

#### Scenario: Use consistent terminal presentation

- **WHEN** the root operator menu or identity questionnaire is rendered
- **THEN** both use the shared Dovik terminal palette and interaction styling
- **AND** do not introduce a parallel visual theme
