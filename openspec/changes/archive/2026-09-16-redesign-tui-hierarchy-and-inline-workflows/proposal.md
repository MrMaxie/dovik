# Why

The current TUI repeats collection headings and flat `project / process` labels, clears the terminal to run separate forms, and exposes only a small plain-text output tail. This makes navigation harder to scan, editing feel disconnected from the selected entity, and retained logs difficult to inspect or follow.

# What changes

- Order the top-level workspaces as Personas, Projects, and Processes, include useful collection counts in the tabs, and remove duplicated navigator headings and position counters.
- Render processes as unselectable project groups with selectable child processes and explicit lifecycle markers.
- Standardize lower-case labels and Huh-style keyboard hints across all workspaces.
- Preserve safe ANSI styling in process output, keep the main preview focused on the newest lines, and add an in-place full output viewer.
- Add a shell-free log follower in the TUI and a `dovik process logs --follow` CLI mode that also works for stopped processes.
- Embed process actions, project editing, and persona editing in the existing Bubble Tea program instead of quitting and launching a second full-screen form.
- Present project fields in one navigable form and use a distinct confirmation modal before applying changes.
- Update tests, documentation, and the unreleased change summary.

# Out of scope

- Changing daemon lifecycle or identity authority.
- Adding a shell, terminal emulator, log persistence beyond the existing retained daemon output, or a streaming protocol.
- Adding dependencies, publishing, committing, or pushing this change.

# Approval

The user directly requested this TUI hierarchy, editing, and log workflow redesign on 2026-09-16. Private fixture configuration under `.local` remains local operator state and is not part of the public change.
