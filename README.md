# Dovik

Dovik is a local supervisor for development processes. One daemon owns each registered process, its lifecycle, current status, and bounded stdout and stderr. Humans and coding agents use the same local control plane through the CLI, TUI, or MCP.

Dovik also keeps project Git authors and GitHub accounts aligned through explicit identity personas, a transparent `gh` proxy, and optional governed agent sessions.

Dovik 1.1.0 is the current public release.

## What Dovik owns

- Project and process definitions in private per-user state.
- Direct process start, stop, restart, status, and process-tree cleanup.
- Bounded stdout and stderr events for the current or latest process run.
- Local-only client access through a Windows Named Pipe or Unix Domain Socket.
- Project identity routing and governed GitHub CLI execution inside isolated sessions.

Dovik does not provide DNS, HTTPS, reverse proxying, domain management, remote execution, scheduling, or container orchestration. [Cadder](https://github.com/MrMaxie/cadder) remains an independent future integration for endpoint allocation.

## Install

Install Dovik and its matching native package from npm:

```console
npm install --global dovik
dovik --version
dovikd --version
```

The npm package supports Windows x64, Linux x64, macOS x64, and macOS arm64. It installs `dovik` and `dovikd` only. It does not replace or configure your `gh` command.

Portable archives remain available from the [latest GitHub Release](https://github.com/MrMaxie/dovik/releases/latest).

## Quick start from source

Install the pinned tools, build the binaries, and start the daemon:

```console
mise install
just build
just run
```

In another terminal, register a project and process:

```console
dovik project add --id example --root /absolute/path/to/project
dovik process add --project example --id app --command /absolute/path/to/app --arg serve
dovik process start --project example --process app
dovik process status --project example --process app
dovik process logs --project example --process app --tail 100
dovik process logs --project example --process app --follow
```

Run bare `dovik` in an interactive terminal to choose help, project identity configuration, or the TUI. Direct commands remain available as `dovik tui`, `dovik project configure`, and `dovik mcp`.

## Documentation

The release documentation is built from `docs/site` and will be published at [maxie.dev/dovik](https://maxie.dev/dovik/). It covers installation, the first managed process, CLI and TUI workflows, MCP, identity, native and container platforms, security boundaries, updates, and release verification.

Project identity setup remains available in [docs/identity.md](docs/identity.md) while the public site is prepared.

## Development

```console
just check
just test-linux
just test-integration
just test-identity-integration
just docs-check
just package
```

`just check` runs Go vet, unit tests, builds, and strict OpenSpec validation. `just tui-ttyglass` opens the real TUI in a PTY or ConPTY-backed browser terminal for visual verification. Docker-backed integration commands require a running Docker engine. Podman acceptance is reported separately.

## License

Dovik is licensed under the [Apache License 2.0](LICENSE). Third-party dependency terms are summarized in [THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).
