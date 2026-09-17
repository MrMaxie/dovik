# Why

Identity is currently hidden behind a shortcut that replaces the process workspace, while personas have no first-class TUI view. The empty process state also abandons the list-and-body composition used by populated process and identity screens, so the main workspace changes structure precisely when users need orientation.

# What changes

- Add persistent `Processes`, `Identity`, and `Personas` tabs to the top of every supported TUI workspace.
- Switch tabs with the left and right arrow keys while preserving existing process and identity shortcuts.
- Present processes through the same navigator-and-body structure in loading, empty, unavailable, and populated states.
- Add a persona navigator and detail body based on the existing identity snapshot, without exposing credentials or secrets.
- Update contextual help, tests, documentation, and the unreleased change summary.

# Out of scope

- Adding identity or persona administration beyond the existing project questionnaire.
- Changing daemon, registry, identity storage, policy, or GitHub routing contracts.
- Adding mouse navigation, a second TUI framework, or new dependencies.
- Committing, pushing, tagging, or publishing the change.

# Approval

The user directly requested this TUI implementation on 2026-09-16. Existing npm, transparent-gh, guided-CLI, and local working-tree changes remain separate and must be preserved.
