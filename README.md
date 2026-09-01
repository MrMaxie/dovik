# Dovik

Dovik is a local supervisor for development processes. Its intended responsibility is to know which processes belong to which local projects, start them under explicit control, observe their lifecycle, collect their output, and expose one local control plane to human and agent clients.

## Current status

This repository currently contains the project foundation and the active OpenSpec change for the first working supervisor. Process registration, lifecycle control, IPC, and log retrieval are specified but are not implemented yet. The `dovikd` and `dovik` binaries are deliberately minimal shells that make this status explicit.

The target version is `1.0.0` under SemVer. It is currently in soak and has not been published as a stable release.

## Responsibility boundary

Dovik manages the lifecycle of local development processes. It is not a reverse proxy, DNS server, domain manager, remote executor, scheduler, or container orchestrator.

[Cadder](https://github.com/MrMaxie/cadder) owns local service addressability, including domains, HTTPS, and routing. A future Dovik integration may optionally request an endpoint from Cadder and pass it to a process through its environment, but Cadder is not a dependency of Dovik and no integration is part of the initial scope.

The first runtime targets native Windows and Linux inside Docker. A containerized Dovik daemon is isolated to its container and does not control host processes.

## Development

Install the pinned Go toolchain and developer tools:

```console
mise install
```

OpenSpec must be installed separately and available on `PATH`. The project does not install it through Node or npm.

Common commands:

```console
just build
just test
just format
just lint
just check
just run
just run-cli
just test-linux
just run-linux
```

`just check` runs Go vet, tests, builds, schema validation, and strict validation of all OpenSpec artifacts. `just test-linux` builds the Docker test stage. `just run-linux` builds and runs the Linux runtime image without publishing a port.

Testcontainers is intentionally not part of the empty foundation. It will be added as a test-only dependency when the first functional Linux runtime integration test exists.
