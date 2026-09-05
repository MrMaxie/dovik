# Why

The terminal workspace gives empty panels more visual weight than the user's next action. With no registered processes, it shows selection and output regions that have no subject. A registry connection failure can also render as an empty registry because the empty summary returns before rendering the failure notice.

The user approved this direction on 2026-09-05. This proposal records planned work; implementation and acceptance remain outstanding.

# What changes

- Distinguish loading, successfully loaded empty registry, unavailable registry, and populated workspace.
- Replace empty selection and output panels with one clear empty state, CLI setup guidance, and relevant shortcuts.
- Keep the populated navigator and process/output relationship while reducing frames, filled surfaces, uppercase headings, and competing accents.
- Make process identity, lifecycle state, applicable actions, and recovery messages the primary hierarchy.
- Keep keyboard meanings and daemon authority intact.

State handling and presentation belong together because a lighter layout must still distinguish unavailable data from an empty registry and preserve recovery controls. The development browser shell is independently scoped in `clarify-tui-preview-shell`; neither proposal depends on the other.

# Out of scope

- Registry editing inside the TUI.
- New runtime or protocol features.
- Framework migration or new dependencies.
- Browser harness chrome and diagnostic panel changes.
- Version bumps, publication, and releases.
