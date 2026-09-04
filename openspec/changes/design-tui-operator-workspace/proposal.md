# Why

The current TUI implements the required operator actions, but presents projects, process state, output, notices, details, and shortcuts as one visually flat text document. The browser harness makes the functional behavior observable and also shows that the interface lacks the hierarchy, state emphasis, and color system needed for fast repeated process operation.

Developers need to identify the selected process, understand its state, find the available action, and scan recent output without rereading the whole screen. The presentation should make those tasks immediate while preserving the existing daemon-owned behavior and complete keyboard workflow.

# What changes

- introduce one coherent Dovik TUI visual system with a fixed semantic palette, spacing, borders, emphasis, and reusable presentation styles;
- replace the flat document with a responsive operator workspace that separates process navigation, selected-process state and actions, and bounded output;
- make selection, lifecycle state, pending work, notices, failures, stdout, and stderr visually distinct without relying on color alone;
- retain every existing keyboard action, stable selection behavior, detail disclosure, polling behavior, and terminal restoration contract;
- support true color where available, deterministic downsampling for limited color profiles, and a readable unstyled fallback when color is disabled;
- make browser-harness inspection and screenshots at representative terminal sizes part of acceptance for the visual implementation.

This change builds on the implemented `add-cli-tui-mvp` and `add-browser-tui-harness` changes. It changes TUI presentation only and does not redefine process supervision or the local protocol.

# Out of scope

- new lifecycle, registry, log, daemon, or protocol capabilities;
- mouse-only interactions or removal of keyboard access;
- user-selectable themes, theme files, plugins, animation, charts, or dashboard metrics;
- a browser or desktop product interface;
- copying another product's visual identity or introducing a Cadder dependency or contract;
- changing CLI output, JSON output, process definitions, or environment-value redaction.

## Implementation status

**Status:** Approved
**Created:** 2026-09-04
