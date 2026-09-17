# Why

Running `dovik` without a command currently exits with an argument error even in an operator terminal, while inspecting the selected project persona requires a longer identity command. This makes the common entrypoints harder to discover and prevents a fast, stable shell-prompt integration.

# What changes

- Present a small operator menu for an interactive bare `dovik` invocation, with help, repository configuration, and TUI actions.
- Keep redirected bare invocations deterministic by printing help without contacting the daemon.
- Add `dovik whoami` as a bounded local status query for the selected persona and configuration state.
- Reuse the established Dovik terminal theme and simplify the proxy-level identity questionnaire so isolation-only policy choices appear only for isolated projects.
- Document a portable Starship custom module and update the operator's local Starship configuration without publishing workstation-specific values.

# Out of scope

- Automatically choosing a persona or configuring an unconfigured repository.
- Validating GitHub credentials or contacting GitHub from `dovik whoami`.
- Changing the existing transparent `gh` contract, identity storage, process lifecycle APIs, release version, or package distribution.
- Committing, pushing, tagging, or publishing the change.

# Approval

The user approved this exact implementation plan on 2026-09-16. The existing npm change and transparent-gh working-tree changes remain separate and must be preserved.
