# Why

The three TUI workspaces currently expose inconsistent refresh and selection keys. Identity also uses a separate `c` shortcut for its primary action, while the other entity lists do not share a discoverable Enter-driven interaction.

# What changes

- Use `l` to refresh Processes, Identity, and Personas.
- Use only the up and down arrow keys to change navigator selection.
- Use Enter as the primary action for the selected process, project identity, or persona.
- Open a process action menu from Enter, keep project editing on the existing questionnaire, and add a focused persona editor.
- Update contextual help, tests, documentation, and the unreleased change summary.

# Out of scope

- Editing process definitions or environment overrides.
- Changing daemon protocol, registry persistence, identity routing, or process lifecycle authority.
- Adding mouse navigation or new dependencies.
- Publishing, committing, or pushing the change.

# Approval

The user directly requested the keyboard and primary-action changes on 2026-09-16. The local populated TUI fixture is private operator state and is not part of this public change.
