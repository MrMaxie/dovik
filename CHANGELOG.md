# Changelog

All notable changes to Dovik are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and Dovik follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-09-15

### Added

- Supervise local development processes from one daemon-owned project registry with controlled start, stop, restart, status, and bounded stdout and stderr access.
- Operate registered processes through a non-interactive CLI, an interactive TUI, or seven structured MCP tools backed by the same versioned local protocol.
- Use a current-user Windows Named Pipe or private Unix Domain Socket without exposing a network control port.
- Start a colocated native Windows daemon explicitly from the TUI while keeping connection-only environments opt-in.
- Configure project identity personas, project-local Git authors, governed GitHub CLI operations, policy presets, and revocable native or Docker/Podman isolated agent sessions.
- Discover the current repository and authenticated GitHub accounts during identity setup, reuse the centrally configured original `gh` executable across projects, and keep repository permissions separate from persona switching.
- Preserve ordinary GitHub CLI behavior through the optional `gh` proxy: unconfigured repositories delegate unchanged, help and interactive commands retain their native terminal behavior, and configured projects select the intended account without a global authentication switch.
- Inspect project identity, policy, sessions, and all eight repository-operation permissions in the daemon-backed TUI, with configuration presented in the same terminal workspace and visual system.
- Install Dovik from portable Windows x64, Linux x64, macOS x64, or macOS arm64 archives, or run the non-root Linux amd64 container image.
- Follow public Starlight documentation and version-matched Codex and Claude skills for human and agent workflows.

### Security

- Keep credentials inside the trusted GitHub CLI execution boundary and select accounts per call without changing the global active GitHub account.
- Reject unrecognized GitHub operations, credential export, authentication changes, repository escapes, aliases, extensions, editors, arbitrary GraphQL, and unsupported REST operations before credential acquisition.
- Authenticate isolated native sessions by operating-system peer identity, reject privileged or incorrect peers, and fail closed when the requested boundary cannot be established.
- Build release artifacts from allowlisted inputs, publish SHA-256 checksums and attestations, and verify packages outside the source checkout.

[1.0.0]: https://github.com/MrMaxie/dovik/releases/tag/v1.0.0
