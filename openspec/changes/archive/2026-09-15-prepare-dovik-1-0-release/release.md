---
impact: major
visibility: public
components:
  - development-workflow
  - local-control-plane
  - managed-process-lifecycle
  - operator-clients
---

## Added

### Install Dovik as a portable local process supervisor

Dovik 1.0.0 provides versioned native archives for Windows, Linux, and macOS, plus a non-root Linux container image. Each distribution includes the daemon, operator client, optional governed GitHub CLI proxy, documentation, licenses, and agent skills.

### Operate Dovik from documented human and agent workflows

The public documentation covers installation, process supervision, CLI and TUI use, MCP integration, project identity, isolated agent sessions, platform setup, security boundaries, upgrades, and release verification.

## Security

### Verify release inputs and preserve local boundaries

Release automation uses exact revisions, allowlisted package and Docker inputs, SHA-256 checksums, attestations, native runner checks, and post-publication re-download tests. Dovik exposes no network control port, does not publish private local state, and does not silently activate the optional `gh` proxy.
