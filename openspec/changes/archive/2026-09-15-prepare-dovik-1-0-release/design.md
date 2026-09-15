# Approach

## Version and command metadata

Keep `1.0.0` in one Go package used by both binaries and by release validation. Handle `--version` and all help paths before creating an operational daemon client. `dovik --help`, `dovik <command> --help`, and `dovikd --help` print to standard output and exit successfully without reading state or opening a local transport.

## Portable artifacts

Build each target on its native GitHub-hosted runner. Each archive has one top-level directory and an exact allowlisted payload: `dovik`, `dovikd`, optional `gh`, `README.md`, `CHANGELOG.md`, `LICENSE`, `THIRD_PARTY_LICENSES.md`, and `skills/`. Windows uses `.exe` and ZIP; Linux and macOS use executable files and tar.gz. Generate a SHA-256 file for every archive and attest the archives, checksums, and container image.

The candidate workflow accepts an exact revision and uploads workflow artifacts only. The release workflow runs only for a `v*.*.*` tag, verifies that the tag version equals the source version and changelog, rebuilds from that tagged revision, verifies the result, then creates the GitHub Release and publishes the container tags. Publication is never inferred from a successful candidate.

## Container boundary

Build the Linux amd64 image from an explicit Docker context allowlist. The final image runs `dovikd` as UID 10001, exposes no port, and contains `dovik`, `dovikd`, and the optional `gh` proxy. Version tags are `1.0.0`, `1.0`, `1`, and `latest`. The image manages only processes started inside its own container.

## Documentation and agent guidance

Use Astro Starlight under `docs/site` with the same base-path deployment pattern as the Cadder documentation. Keep the root README as a short product entry. Documentation covers installation, first use, CLI/TUI, MCP, identity, platforms, security boundaries, upgrades, release verification, and the absence of signing/notarization. The landing preview is a sanitized HTML reconstruction checked against the real TUI through ttyglass; it contains no private paths, accounts, endpoints, or process output.

Ship two versioned skill entrypoints with equivalent product rules for Codex and Claude. The skills prefer Dovik status and logs over PID or port guessing, preserve daemon ownership, use the configured identity persona and `gh` proxy, never run `gh auth switch`, and do not create project or process definitions without user scope.

## Scoop

Track `packaging/scoop/dovik.json` as a release template that points at the Windows x64 GitHub Release archive. It exposes `dovik` and `dovikd` only. `checkver` and `autoupdate` follow GitHub releases. The release hash is inserted only from the verified public artifact; a placeholder cannot be submitted to Scoop Main.

## Acceptance boundary

Local checks prove source behavior and deterministic packaging logic. Native runners prove platform behavior and archive usability. Candidate artifacts prove a specific master revision. Public re-download proves the published release. Cross-compilation does not establish native behavior, and unavailable Podman, separate-account, signing, or publication evidence remains explicit.
