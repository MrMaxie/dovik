# Why

Dovik has a working daemon, CLI, TUI, MCP operator client, and a locally implemented identity system, but it does not yet have a public installation, documentation, versioning, or release contract. Version 1.0.0 needs one reproducible path from an exact source revision to native archives, a container image, consumer documentation, agent guidance, and post-publication verification.

# What changes

Prepare Dovik 1.0.0 as a public Apache-2.0 project:

- support the daemon and clients on native Windows, Linux, and macOS;
- report one shared version from `dovik` and `dovikd` and make help available without contacting the daemon;
- build four portable native archives and one Linux amd64 container image from an exact revision;
- verify archive contents, checksums, executable permissions, help, versions, and isolated consumer use;
- publish user documentation with Astro Starlight at `https://maxie.dev/dovik/`;
- distribute version-matched Codex and Claude skills with the archives;
- provide a Scoop manifest for the Windows archive without activating the optional `gh` proxy;
- add continuous integration, candidate artifact production, tag-gated release publication, attestations, GHCR publication, Pages deployment, and post-release verification.

The existing seven MCP operator tools remain unchanged. Identity administration is not added to MCP.

# Out of scope

- installers, Homebrew, Winget, Authenticode, or macOS notarization;
- Linux arm64 or Windows arm64 artifacts;
- automatic account provisioning, container-engine installation, or PATH modification;
- a Cadder dependency or endpoint-allocation contract;
- DNS, HTTPS, reverse proxying, domains, remote execution, scheduling, or container orchestration;
- a Scoop Main submission before the public release has been independently verified.

# Approval

The implementation plan and this release scope were approved by the user on 2026-09-15. Publishing, repository visibility, commit, push, Pages configuration, tagging, GitHub Release creation, GHCR publication, and a Scoop Main pull request remain separate authorization boundaries.
